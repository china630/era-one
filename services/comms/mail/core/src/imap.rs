//! IMAP4rev1 subset with AUTH, STARTTLS, UID FETCH, SEARCH (R2-C).

use crate::store::MailBackend;
use crate::tls_util;
use anyhow::Result;
use std::sync::Arc;
use tokio::io::{AsyncBufReadExt, AsyncRead, AsyncWrite, AsyncWriteExt, BufReader};
use tokio::net::{TcpListener, TcpStream};
use tokio_rustls::TlsAcceptor;
use tracing::warn;

pub async fn serve(addr: &str, store: Arc<dyn MailBackend>) -> Result<()> {
    let acceptor = tls_util::server_acceptor().ok().flatten();
    let listener = TcpListener::bind(addr).await?;
    loop {
        let (stream, peer) = listener.accept().await?;
        let store = Arc::clone(&store);
        let acc = acceptor.clone();
        tokio::spawn(async move {
            if let Err(e) = handle_tcp_client(stream, store, acc).await {
                warn!(%peer, error = %e, "imap session error");
            }
        });
    }
}

pub async fn serve_on(stream: TcpStream, store: Arc<dyn MailBackend>) -> Result<()> {
    let acceptor = tls_util::server_acceptor().ok().flatten();
    handle_tcp_client(stream, store, acceptor).await
}

async fn handle_tcp_client(
    stream: TcpStream,
    store: Arc<dyn MailBackend>,
    acceptor: Option<TlsAcceptor>,
) -> Result<()> {
    let (reader, writer) = stream.into_split();
    handle_plain_session(reader, writer, store, acceptor).await
}

async fn handle_plain_session(
    reader: tokio::net::tcp::OwnedReadHalf,
    mut writer: tokio::net::tcp::OwnedWriteHalf,
    store: Arc<dyn MailBackend>,
    acceptor: Option<TlsAcceptor>,
) -> Result<()> {
    let mut lines = BufReader::new(reader).lines();
    writer
        .write_all(b"* OK era-mail-core IMAP4rev1 ready\r\n")
        .await?;

    let mut tag = String::new();
    let mut user_mailbox = String::new();
    let mut selected = String::new();
    let mut authenticated = false;

    while let Some(line) = lines.next_line().await? {
        let parts: Vec<&str> = line.split_whitespace().collect();
        if parts.is_empty() {
            continue;
        }
        tag = parts[0].to_string();
        let cmd = parts.get(1).map(|s| s.to_uppercase()).unwrap_or_default();

        if cmd == "STARTTLS" {
            if let Some(acc) = acceptor {
                writer
                    .write_all(format!("{tag} OK Begin TLS negotiation now\r\n").as_bytes())
                    .await?;
                let read_half = lines.into_inner().into_inner();
                let tcp = read_half.reunite(writer)?;
                let tls = acc.accept(tcp).await?;
                return handle_tls_session(tls, store).await;
            }
            writer
                .write_all(format!("{tag} NO TLS not available\r\n").as_bytes())
                .await?;
            continue;
        }

        if !process_imap_command(
            &cmd,
            &parts,
            &tag,
            &mut writer,
            &store,
            &mut authenticated,
            &mut user_mailbox,
            &mut selected,
        )
        .await?
        {
            break;
        }
    }
    Ok(())
}

async fn handle_tls_session<S>(stream: S, store: Arc<dyn MailBackend>) -> Result<()>
where
    S: AsyncRead + AsyncWrite + Unpin + Send + 'static,
{
    let (reader, mut writer) = tokio::io::split(stream);
    let mut lines = BufReader::new(reader).lines();
    writer
        .write_all(b"* OK era-mail-core IMAP4rev1 ready\r\n")
        .await?;

    let mut tag = String::new();
    let mut user_mailbox = String::new();
    let mut selected = String::new();
    let mut authenticated = false;

    while let Some(line) = lines.next_line().await? {
        let parts: Vec<&str> = line.split_whitespace().collect();
        if parts.is_empty() {
            continue;
        }
        tag = parts[0].to_string();
        let cmd = parts.get(1).map(|s| s.to_uppercase()).unwrap_or_default();

        if !process_imap_command(
            &cmd,
            &parts,
            &tag,
            &mut writer,
            &store,
            &mut authenticated,
            &mut user_mailbox,
            &mut selected,
        )
        .await?
        {
            break;
        }
    }
    Ok(())
}

async fn process_imap_command<W>(
    cmd: &str,
    parts: &[&str],
    tag: &str,
    writer: &mut W,
    store: &Arc<dyn MailBackend>,
    authenticated: &mut bool,
    user_mailbox: &mut String,
    selected: &mut String,
) -> Result<bool>
where
    W: AsyncWrite + Unpin,
{
    match cmd {
        "CAPABILITY" => {
            writer
                .write_all(b"* CAPABILITY IMAP4rev1 AUTH=PLAIN STARTTLS\r\n")
                .await?;
            respond_ok(writer, tag).await?;
        }
        "LOGIN" if parts.len() >= 4 => {
            let user = parts[2].trim_matches('"').to_lowercase();
            let pass = parts[3].trim_matches('"');
            if store.verify_login(&user, pass) {
                *authenticated = true;
                *user_mailbox = user;
                *selected = user_mailbox.clone();
                respond_ok(writer, tag).await?;
            } else {
                writer
                    .write_all(format!("{tag} NO LOGIN failed\r\n").as_bytes())
                    .await?;
            }
        }
        "LIST" if *authenticated => {
            writer
                .write_all(b"* LIST () \"/\" INBOX\r\n* LIST () \"/\" Sent\r\n")
                .await?;
            respond_ok(writer, tag).await?;
        }
        "SELECT" | "EXAMINE" if *authenticated && parts.len() >= 3 => {
            let folder = parts[2].trim_matches('"').to_uppercase();
            *selected = if folder == "INBOX" {
                user_mailbox.clone()
            } else {
                parts[2].trim_matches('"').to_lowercase()
            };
            let count = store.list_mailbox(selected)?.len();
            writer
                .write_all(format!("* {count} EXISTS\r\n").as_bytes())
                .await?;
            writer.write_all(b"* 0 RECENT\r\n").await?;
            writer
                .write_all(b"* OK [UIDVALIDITY 1] UIDs valid\r\n")
                .await?;
            writer
                .write_all(b"* FLAGS (\\Seen \\Answered)\r\n")
                .await?;
            respond_ok(writer, tag).await?;
        }
        "SEARCH" if *authenticated => {
            let msgs = store.list_mailbox(selected)?;
            let mut uids: Vec<String> = msgs.iter().map(|m| m.uid.to_string()).collect();
            if parts.len() >= 3 && parts[2].to_uppercase() == "UNSEEN" {
                uids.retain(|_| true);
            }
            writer
                .write_all(format!("* SEARCH {}\r\n", uids.join(" ")).as_bytes())
                .await?;
            respond_ok(writer, tag).await?;
        }
        "FETCH" | "UID" if *authenticated => {
            let (seq, is_uid) = if cmd == "UID" && parts.len() >= 4 {
                (parts[3], true)
            } else if parts.len() >= 3 {
                (parts[2], false)
            } else {
                respond_bad(writer, tag).await?;
                return Ok(true);
            };
            let msgs = store.list_mailbox(selected)?;
            if let Some(msg) = msgs.first() {
                let id = if is_uid || seq.contains("UID") {
                    format!("UID {}", msg.uid)
                } else {
                    "1".to_string()
                };
                let body = String::from_utf8_lossy(&msg.raw);
                writer
                    .write_all(
                        format!(
                            "* {} FETCH ({id} BODY[] {{{}}}\r\n{}\r\n)\r\n",
                            if is_uid { msg.uid } else { 1 },
                            msg.raw.len(),
                            body
                        )
                        .as_bytes(),
                    )
                    .await?;
            }
            respond_ok(writer, tag).await?;
        }
        "LOGOUT" => {
            writer.write_all(b"* BYE era-mail-core\r\n").await?;
            respond_ok(writer, tag).await?;
            return Ok(false);
        }
        "NOOP" => respond_ok(writer, tag).await?,
        _ => respond_bad(writer, tag).await?,
    }
    Ok(true)
}

async fn respond_ok<W: AsyncWrite + Unpin>(writer: &mut W, tag: &str) -> Result<()> {
    writer
        .write_all(format!("{tag} OK\r\n").as_bytes())
        .await?;
    Ok(())
}

async fn respond_bad<W: AsyncWrite + Unpin>(writer: &mut W, tag: &str) -> Result<()> {
    writer
        .write_all(format!("{tag} BAD unknown command\r\n").as_bytes())
        .await?;
    Ok(())
}
