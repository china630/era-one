//! Minimal SMTP receiver with AUTH PLAIN/LOGIN and STARTTLS (R2-C).

use crate::audit_hook::AuditHook;
use crate::store::MailBackend;
use crate::tls_util;
use anyhow::{anyhow, Result};
use std::sync::Arc;
use tokio::io::{AsyncBufReadExt, AsyncRead, AsyncWrite, AsyncWriteExt, BufReader};
use tokio::net::{TcpListener, TcpStream};
use tokio_rustls::TlsAcceptor;
use tracing::{debug, warn};

const DEFAULT_MAX_MESSAGE_BYTES: usize = 25 * 1024 * 1024;

fn max_message_bytes() -> usize {
    std::env::var("ERA_MAIL_MAX_MESSAGE_BYTES")
        .ok()
        .and_then(|s| s.parse().ok())
        .unwrap_or(DEFAULT_MAX_MESSAGE_BYTES)
}

pub async fn serve(addr: &str, store: Arc<dyn MailBackend>, audit: AuditHook) -> Result<()> {
    let acceptor = tls_util::server_acceptor().ok().flatten();
    let listener = TcpListener::bind(addr).await?;
    loop {
        let (stream, peer) = listener.accept().await?;
        let store = Arc::clone(&store);
        let audit = audit.clone();
        let acc = acceptor.clone();
        tokio::spawn(async move {
            if let Err(e) = handle_tcp_client(stream, store, audit, acc).await {
                warn!(%peer, error = %e, "smtp session error");
            }
        });
    }
}

pub async fn serve_on(
    stream: TcpStream,
    store: Arc<dyn MailBackend>,
    audit: AuditHook,
) -> Result<()> {
    let acceptor = tls_util::server_acceptor().ok().flatten();
    handle_tcp_client(stream, store, audit, acceptor).await
}

async fn handle_tcp_client(
    stream: TcpStream,
    store: Arc<dyn MailBackend>,
    audit: AuditHook,
    acceptor: Option<TlsAcceptor>,
) -> Result<()> {
    let (reader, writer) = stream.into_split();
    handle_plain_session(reader, writer, store, audit, acceptor).await
}

async fn handle_plain_session(
    reader: tokio::net::tcp::OwnedReadHalf,
    mut writer: tokio::net::tcp::OwnedWriteHalf,
    store: Arc<dyn MailBackend>,
    audit: AuditHook,
    acceptor: Option<TlsAcceptor>,
) -> Result<()> {
    let mut lines = BufReader::new(reader).lines();
    writer.write_all(b"220 era-mail-core ESMTP ready\r\n").await?;

    let mut mail_from = String::new();
    let mut rcpt_to = String::new();
    let mut in_data = false;
    let mut data_buf = String::new();
    let mut authenticated = false;
    let mut auth_user = String::new();

    while let Some(line) = lines.next_line().await? {
        if in_data {
            if line == "." {
                if rcpt_to.is_empty() {
                    writer.write_all(b"550 no recipient\r\n").await?;
                } else if data_buf.len() > max_message_bytes() {
                    writer.write_all(b"552 message too large\r\n").await?;
                } else {
                    store.deliver(&rcpt_to, &data_buf)?;
                    audit.notify_send(&rcpt_to, &mail_from);
                    debug!(to = %rcpt_to, bytes = data_buf.len(), "message stored");
                    writer.write_all(b"250 OK\r\n").await?;
                }
                in_data = false;
                data_buf.clear();
                rcpt_to.clear();
                mail_from.clear();
                continue;
            }
            data_buf.push_str(&line);
            data_buf.push_str("\r\n");
            continue;
        }

        let upper = line.to_uppercase();
        if upper.starts_with("EHLO") || upper.starts_with("HELO") {
            let mut caps = b"250-era-mail-core\r\n250-AUTH PLAIN LOGIN\r\n".to_vec();
            if acceptor.is_some() {
                caps.extend_from_slice(b"250-STARTTLS\r\n");
            }
            caps.extend_from_slice(b"250 OK\r\n");
            writer.write_all(&caps).await?;
        } else if upper == "STARTTLS" {
            if let Some(acc) = acceptor {
                writer.write_all(b"220 Ready to start TLS\r\n").await?;
                let read_half = lines.into_inner().into_inner();
                let tcp = read_half.reunite(writer)?;
                let tls = acc.accept(tcp).await?;
                return handle_tls_session(tls, store, audit).await;
            }
            writer.write_all(b"502 TLS not available\r\n").await?;
        } else if !process_command(
            &upper,
            &line,
            &mut lines,
            &mut writer,
            &store,
            &mut authenticated,
            &mut auth_user,
            &mut mail_from,
            &mut rcpt_to,
            &mut in_data,
        )
        .await?
        {
            break;
        }
    }
    Ok(())
}

async fn handle_tls_session<S>(
    stream: S,
    store: Arc<dyn MailBackend>,
    audit: AuditHook,
) -> Result<()>
where
    S: AsyncRead + AsyncWrite + Unpin + Send + 'static,
{
    let (reader, mut writer) = tokio::io::split(stream);
    let mut lines = BufReader::new(reader).lines();
    writer.write_all(b"220 era-mail-core ESMTP ready\r\n").await?;

    let mut mail_from = String::new();
    let mut rcpt_to = String::new();
    let mut in_data = false;
    let mut data_buf = String::new();
    let mut authenticated = false;
    let mut auth_user = String::new();

    while let Some(line) = lines.next_line().await? {
        if in_data {
            if line == "." {
                if rcpt_to.is_empty() {
                    writer.write_all(b"550 no recipient\r\n").await?;
                } else if data_buf.len() > max_message_bytes() {
                    writer.write_all(b"552 message too large\r\n").await?;
                } else {
                    store.deliver(&rcpt_to, &data_buf)?;
                    audit.notify_send(&rcpt_to, &mail_from);
                    writer.write_all(b"250 OK\r\n").await?;
                }
                in_data = false;
                data_buf.clear();
                rcpt_to.clear();
                mail_from.clear();
                continue;
            }
            data_buf.push_str(&line);
            data_buf.push_str("\r\n");
            continue;
        }
        let upper = line.to_uppercase();
        if upper.starts_with("EHLO") || upper.starts_with("HELO") {
            writer
                .write_all(b"250-era-mail-core\r\n250-AUTH PLAIN LOGIN\r\n250 OK\r\n")
                .await?;
        } else if !process_command(
            &upper,
            &line,
            &mut lines,
            &mut writer,
            &store,
            &mut authenticated,
            &mut auth_user,
            &mut mail_from,
            &mut rcpt_to,
            &mut in_data,
        )
        .await?
        {
            break;
        }
    }
    Ok(())
}

async fn process_command<W>(
    upper: &str,
    line: &str,
    lines: &mut tokio::io::Lines<BufReader<impl AsyncRead + Unpin>>,
    writer: &mut W,
    store: &Arc<dyn MailBackend>,
    authenticated: &mut bool,
    auth_user: &mut String,
    mail_from: &mut String,
    rcpt_to: &mut String,
    in_data: &mut bool,
) -> Result<bool>
where
    W: AsyncWrite + Unpin,
{
    if upper.starts_with("AUTH PLAIN") {
        let creds = parse_auth_plain(line)?;
        if store.verify_login(&creds.0, &creds.1) {
            *authenticated = true;
            *auth_user = creds.0;
            writer.write_all(b"235 Authentication successful\r\n").await?;
        } else {
            writer.write_all(b"535 authentication failed\r\n").await?;
        }
    } else if upper.starts_with("AUTH LOGIN") {
        writer.write_all(b"334 VXNlcm5hbWU6\r\n").await?;
        let user_line = lines.next_line().await?.unwrap_or_default();
        writer.write_all(b"334 UGFzc3dvcmQ6\r\n").await?;
        let pass_line = lines.next_line().await?.unwrap_or_default();
        let user = decode_base64(&user_line)?;
        let pass = decode_base64(&pass_line)?;
        if store.verify_login(&user, &pass) {
            *authenticated = true;
            *auth_user = user;
            writer.write_all(b"235 Authentication successful\r\n").await?;
        } else {
            writer.write_all(b"535 authentication failed\r\n").await?;
        }
    } else if upper.starts_with("MAIL FROM:") {
        if !*authenticated {
            writer.write_all(b"530 authentication required\r\n").await?;
        } else {
            *mail_from = parse_addr(line)?;
            if mail_from.is_empty() && !auth_user.is_empty() {
                *mail_from = auth_user.clone();
            }
            writer.write_all(b"250 OK\r\n").await?;
        }
    } else if upper.starts_with("RCPT TO:") {
        if !*authenticated {
            writer.write_all(b"530 authentication required\r\n").await?;
        } else {
            *rcpt_to = parse_addr(line)?;
            writer.write_all(b"250 OK\r\n").await?;
        }
    } else if upper == "DATA" {
        if !*authenticated {
            writer.write_all(b"530 authentication required\r\n").await?;
        } else if rcpt_to.is_empty() {
            writer.write_all(b"503 need RCPT\r\n").await?;
        } else {
            *in_data = true;
            writer.write_all(b"354 start mail input\r\n").await?;
        }
    } else if upper == "QUIT" {
        writer.write_all(b"221 bye\r\n").await?;
        return Ok(false);
    } else if upper == "RSET" {
        mail_from.clear();
        rcpt_to.clear();
        *in_data = false;
        writer.write_all(b"250 OK\r\n").await?;
    } else if upper != "STARTTLS" {
        writer.write_all(b"502 command not implemented\r\n").await?;
    }
    Ok(true)
}

fn parse_addr(line: &str) -> Result<String> {
    let start = line.find('<').ok_or_else(|| anyhow!("missing addr"))? + 1;
    let end = line.find('>').ok_or_else(|| anyhow!("missing addr"))?;
    Ok(line[start..end].trim().to_lowercase())
}

fn parse_auth_plain(line: &str) -> Result<(String, String)> {
    let parts: Vec<&str> = line.splitn(3, ' ').collect();
    let b64 = parts.get(2).copied().unwrap_or("");
    let decoded = decode_base64(b64)?;
    let chunks: Vec<&str> = decoded.split('\0').collect();
    if chunks.len() >= 3 {
        return Ok((chunks[1].to_lowercase(), chunks[2].to_string()));
    }
    Err(anyhow!("bad AUTH PLAIN"))
}

fn decode_base64(s: &str) -> Result<String> {
    let bytes = base64_decode(s.trim())?;
    Ok(String::from_utf8_lossy(&bytes).to_string())
}

fn base64_decode(s: &str) -> Result<Vec<u8>> {
    const TABLE: &[u8; 256] = &{
        let mut t = [255u8; 256];
        let mut i = 0u8;
        while i < 26 {
            t[(b'A' + i) as usize] = i;
            t[(b'a' + i) as usize] = i + 26;
            i += 1;
        }
        let mut d = 0u8;
        while d < 10 {
            t[(b'0' + d) as usize] = d + 52;
            d += 1;
        }
        t[b'+' as usize] = 62;
        t[b'/' as usize] = 63;
        t
    };
    let s: String = s.chars().filter(|c| !c.is_whitespace()).collect();
    let mut out = Vec::with_capacity(s.len() * 3 / 4);
    let bytes = s.as_bytes();
    let mut buf = [0u8; 4];
    let mut n = 0usize;
    for &b in bytes {
        if b == b'=' {
            break;
        }
        let v = TABLE[b as usize];
        if v == 255 {
            continue;
        }
        buf[n] = v;
        n += 1;
        if n == 4 {
            out.push((buf[0] << 2) | (buf[1] >> 4));
            out.push((buf[1] << 4) | (buf[2] >> 2));
            out.push((buf[2] << 6) | buf[3]);
            n = 0;
        }
    }
    if n >= 2 {
        out.push((buf[0] << 2) | (buf[1] >> 4));
    }
    if n >= 3 {
        out.push((buf[1] << 4) | (buf[2] >> 2));
    }
    Ok(out)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parse_mail_from() {
        let addr = parse_addr("MAIL FROM:<Alice@Mail.Gov.AZ>").unwrap();
        assert_eq!(addr, "alice@mail.gov.az");
    }
}
