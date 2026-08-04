//! E2E: SMTP STARTTLS → AUTH → MAIL → DATA (R2-C).

use era_mail_core::{audit_hook::AuditHook, smtp, store::MailStore, MailBackend};
use rustls::pki_types::{CertificateDer, ServerName};
use rustls::{ClientConfig, RootCertStore};
use std::sync::Arc;
use std::time::Duration;
use tokio::io::{AsyncBufReadExt, AsyncWriteExt, BufReader};
use tokio::net::TcpStream;
use tokio::time::timeout;
use tokio_rustls::TlsConnector;

fn test_connector(trust: CertificateDer<'static>) -> TlsConnector {
    let _ = rustls::crypto::ring::default_provider().install_default();
    let mut roots = RootCertStore::empty();
    roots.add(trust).unwrap();
    let cfg = ClientConfig::builder()
        .with_root_certificates(roots)
        .with_no_client_auth();
    TlsConnector::from(Arc::new(cfg))
}

fn server_cert_from_dir(dir: &std::path::Path) -> CertificateDer<'static> {
    let pem = std::fs::read(dir.join("server.crt")).unwrap();
    let mut buf = pem.as_slice();
    let certs: Vec<CertificateDer<'static>> =
        rustls_pemfile::certs(&mut buf).collect::<Result<Vec<_>, _>>().unwrap();
    certs.into_iter().next().expect("cert")
}
fn write_tls_fixtures(dir: &std::path::Path) {
    let cert = rcgen::generate_simple_self_signed(vec!["localhost".into()]).unwrap();
    std::fs::write(dir.join("server.crt"), cert.cert.pem()).unwrap();
    std::fs::write(dir.join("server.key"), cert.key_pair.serialize_pem()).unwrap();
}

#[tokio::test]
async fn smtp_starttls_auth_deliver() {
    let dir = tempfile::tempdir().unwrap();
    write_tls_fixtures(dir.path());
    unsafe {
        std::env::set_var("ERA_MAIL_TLS_CERT", dir.path().join("server.crt"));
        std::env::set_var("ERA_MAIL_TLS_KEY", dir.path().join("server.key"));
    }

    let store = Arc::new(MailStore::new());
    let audit = AuditHook::from_env();
    let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let port = listener.local_addr().unwrap().port();

    let store_s = Arc::clone(&store);
    let audit_s = audit.clone();
    tokio::spawn(async move {
        loop {
            let (s, _) = listener.accept().await.unwrap();
            let st = Arc::clone(&store_s);
            let au = audit_s.clone();
            tokio::spawn(async move {
                smtp::serve_on(s, st, au).await.ok();
            });
        }
    });

    timeout(
        Duration::from_secs(10),
        smtp_tls_send(dir.path(), port, "alice@mail.gov.az", "bob@mail.gov.az"),
    )
    .await
    .expect("timeout")
    .expect("smtp tls failed");

    let msgs = store.list_mailbox("bob@mail.gov.az").unwrap();
    assert_eq!(msgs.len(), 1);
    assert!(String::from_utf8_lossy(&msgs[0].raw).contains("TLS hello"));
}

#[tokio::test]
async fn smtp_oversized_data_returns_552() {
    unsafe {
        std::env::remove_var("ERA_MAIL_TLS_CERT");
        std::env::remove_var("ERA_MAIL_TLS_KEY");
        std::env::set_var("ERA_MAIL_MAX_MESSAGE_BYTES", "64");
    }
    let store = Arc::new(MailStore::new());
    let audit = AuditHook::from_env();
    let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let port = listener.local_addr().unwrap().port();

    let store_s = Arc::clone(&store);
    let audit_s = audit.clone();
    tokio::spawn(async move {
        loop {
            let (s, _) = listener.accept().await.unwrap();
            let st = Arc::clone(&store_s);
            let au = audit_s.clone();
            tokio::spawn(async move {
                smtp::serve_on(s, st, au).await.ok();
            });
        }
    });

    let resp = timeout(
        Duration::from_secs(10),
        smtp_send_oversized(port, "alice@mail.gov.az", "bob@mail.gov.az"),
    )
    .await
    .expect("timeout")
    .expect("smtp failed");

    assert!(resp.contains("552"), "expected 552, got: {resp}");
    assert_eq!(store.message_count(), 0);
}

async fn smtp_tls_send(
    cert_dir: &std::path::Path,
    port: u16,
    from: &str,
    to: &str,
) -> std::io::Result<()> {
    let s = TcpStream::connect(format!("127.0.0.1:{port}")).await?;
    let (r, mut w) = s.into_split();
    let mut lines = BufReader::new(r).lines();
    read_line(&mut lines).await?;
    write_cmd(&mut w, "EHLO tls.test").await?;
    let mut saw_starttls = false;
    loop {
        let line = read_line(&mut lines).await?;
        if line.contains("STARTTLS") {
            saw_starttls = true;
        }
        if line.starts_with("250 ") && !line.contains('-') {
            break;
        }
    }
    assert!(saw_starttls, "STARTTLS not advertised");
    write_cmd(&mut w, "STARTTLS").await?;
    read_line(&mut lines).await?;

    let read_half = lines.into_inner().into_inner();
    let tcp = read_half.reunite(w).map_err(|e| {
        std::io::Error::new(std::io::ErrorKind::Other, e.to_string())
    })?;
    let trust = server_cert_from_dir(cert_dir);
    let connector = test_connector(trust);
    let sn = ServerName::try_from("localhost").map_err(|e| {
        std::io::Error::new(std::io::ErrorKind::InvalidInput, e.to_string())
    })?;
    let mut tls = connector.connect(sn, tcp).await.map_err(|e| {
        std::io::Error::new(std::io::ErrorKind::Other, e.to_string())
    })?;

    let (r2, mut w2) = tokio::io::split(&mut tls);
    let mut lines2 = BufReader::new(r2).lines();
    read_line_split(&mut lines2).await?;
    write_cmd_split(&mut w2, "EHLO tls.test").await?;
    loop {
        let line = read_line_split(&mut lines2).await?;
        if line.starts_with("250 ") && !line.contains('-') {
            break;
        }
    }
    let auth = format!("\0{from}\0pass");
    let b64 = base64_encode(auth.as_bytes());
    write_cmd_split(&mut w2, &format!("AUTH PLAIN {b64}")).await?;
    read_line_split(&mut lines2).await?;
    write_cmd_split(&mut w2, &format!("MAIL FROM:<{from}>")).await?;
    read_line_split(&mut lines2).await?;
    write_cmd_split(&mut w2, &format!("RCPT TO:<{to}>")).await?;
    read_line_split(&mut lines2).await?;
    write_cmd_split(&mut w2, "DATA").await?;
    read_line_split(&mut lines2).await?;
    write_cmd_split(&mut w2, "Subject: tls\r\n\r\nTLS hello\r\n.").await?;
    let resp = read_line_split(&mut lines2).await?;
    assert!(resp.contains("250"), "deliver failed: {resp}");
    Ok(())
}

async fn smtp_send_oversized(port: u16, from: &str, to: &str) -> std::io::Result<String> {
    let s = TcpStream::connect(format!("127.0.0.1:{port}")).await?;
    let (r, mut w) = s.into_split();
    let mut lines = BufReader::new(r).lines();
    read_line(&mut lines).await?;
    write_cmd(&mut w, "EHLO e2e.test").await?;
    loop {
        let line = read_line(&mut lines).await?;
        if line.starts_with("250 ") && !line.contains('-') {
            break;
        }
    }
    let auth = format!("\0{from}\0pass");
    let b64 = base64_encode(auth.as_bytes());
    write_cmd(&mut w, &format!("AUTH PLAIN {b64}")).await?;
    read_line(&mut lines).await?;
    write_cmd(&mut w, &format!("MAIL FROM:<{from}>")).await?;
    read_line(&mut lines).await?;
    write_cmd(&mut w, &format!("RCPT TO:<{to}>")).await?;
    read_line(&mut lines).await?;
    write_cmd(&mut w, "DATA").await?;
    read_line(&mut lines).await?;
    let huge = "X".repeat(128);
    write_cmd(&mut w, &format!("{huge}\r\n.")).await?;
    read_line(&mut lines).await
}

async fn write_cmd(w: &mut tokio::net::tcp::OwnedWriteHalf, cmd: &str) -> std::io::Result<()> {
    w.write_all(format!("{cmd}\r\n").as_bytes()).await
}

async fn write_cmd_split<W: AsyncWriteExt + Unpin>(w: &mut W, cmd: &str) -> std::io::Result<()> {
    w.write_all(format!("{cmd}\r\n").as_bytes()).await
}

async fn read_line(
    lines: &mut tokio::io::Lines<BufReader<tokio::net::tcp::OwnedReadHalf>>,
) -> std::io::Result<String> {
    lines
        .next_line()
        .await?
        .ok_or_else(|| std::io::Error::new(std::io::ErrorKind::UnexpectedEof, "closed"))
}

async fn read_line_split<R: AsyncBufReadExt + Unpin>(
    lines: &mut tokio::io::Lines<R>,
) -> std::io::Result<String> {
    lines
        .next_line()
        .await?
        .ok_or_else(|| std::io::Error::new(std::io::ErrorKind::UnexpectedEof, "closed"))
}

fn base64_encode(data: &[u8]) -> String {
    const TABLE: &[u8] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    let mut out = String::new();
    for chunk in data.chunks(3) {
        let b0 = chunk[0] as u32;
        let b1 = if chunk.len() > 1 { chunk[1] as u32 } else { 0 };
        let b2 = if chunk.len() > 2 { chunk[2] as u32 } else { 0 };
        let n = (b0 << 16) | (b1 << 8) | b2;
        out.push(TABLE[((n >> 18) & 63) as usize] as char);
        out.push(TABLE[((n >> 12) & 63) as usize] as char);
        if chunk.len() > 1 {
            out.push(TABLE[((n >> 6) & 63) as usize] as char);
        } else {
            out.push('=');
        }
        if chunk.len() > 2 {
            out.push(TABLE[(n & 63) as usize] as char);
        } else {
            out.push('=');
        }
    }
    out
}
