//! E2E: SMTP DATA → IMAP FETCH over TCP (AC-C1).

use era_mail_core::{audit_hook::AuditHook, imap, smtp, store::MailStore, MailBackend};
use std::sync::Arc;
use std::time::Duration;
use tokio::io::{AsyncBufReadExt, AsyncWriteExt, BufReader};
use tokio::net::TcpStream;
use tokio::time::timeout;

#[tokio::test]
async fn smtp_to_imap_e2e() {
    let store = Arc::new(MailStore::new());
    let audit = AuditHook::from_env();

    let smtp_l = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let imap_l = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let smtp_port = smtp_l.local_addr().unwrap().port();
    let imap_port = imap_l.local_addr().unwrap().port();

    let store_s = Arc::clone(&store);
    let audit_s = audit.clone();
    tokio::spawn(async move {
        loop {
            let (s, _) = smtp_l.accept().await.unwrap();
            let st = Arc::clone(&store_s);
            let au = audit_s.clone();
            tokio::spawn(async move {
                smtp::serve_on(s, st, au).await.ok();
            });
        }
    });

    let store_i = Arc::clone(&store);
    tokio::spawn(async move {
        loop {
            let (s, _) = imap_l.accept().await.unwrap();
            let st = Arc::clone(&store_i);
            tokio::spawn(async move {
                imap::serve_on(s, st).await.ok();
            });
        }
    });

    timeout(
        Duration::from_secs(5),
        smtp_send(smtp_port, "alice@mail.gov.az", "bob@mail.gov.az"),
    )
    .await
    .expect("smtp timeout")
    .expect("smtp failed");

    let body = timeout(
        Duration::from_secs(5),
        imap_fetch_first(imap_port, "bob@mail.gov.az"),
    )
    .await
    .expect("imap timeout")
    .expect("imap failed");

    assert!(body.contains("Hello ERA E2E"));
    assert_eq!(store.message_count(), 1);
}

#[tokio::test]
async fn imap_login_deny() {
    let store = Arc::new(MailStore::new());
    store.set_password("bob@mail.gov.az", "correct");
    let imap_l = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let imap_port = imap_l.local_addr().unwrap().port();

    let store_i = Arc::clone(&store);
    tokio::spawn(async move {
        loop {
            let (s, _) = imap_l.accept().await.unwrap();
            let st = Arc::clone(&store_i);
            tokio::spawn(async move {
                imap::serve_on(s, st).await.ok();
            });
        }
    });

    let s = TcpStream::connect(format!("127.0.0.1:{imap_port}")).await.unwrap();
    let (r, mut w) = s.into_split();
    let mut lines = BufReader::new(r).lines();
    read_line(&mut lines).await.unwrap();
    write_cmd(&mut w, r#"a1 LOGIN "bob@mail.gov.az" "wrong""#).await.unwrap();
    let resp = read_line(&mut lines).await.unwrap();
    assert!(resp.contains("NO"), "expected NO for bad password, got: {resp}");
}

async fn smtp_send(port: u16, from: &str, to: &str) -> std::io::Result<()> {
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
    write_cmd(
        &mut w,
        "Subject: e2e\r\n\r\nHello ERA E2E\r\n.",
    )
    .await?;
    read_line(&mut lines).await?;
    write_cmd(&mut w, "QUIT").await?;
    Ok(())
}

async fn imap_fetch_first(port: u16, user: &str) -> std::io::Result<String> {
    let s = TcpStream::connect(format!("127.0.0.1:{port}")).await?;
    let (r, mut w) = s.into_split();
    let mut lines = BufReader::new(r).lines();
    read_line(&mut lines).await?;
    write_cmd(&mut w, &format!(r#"a1 LOGIN "{user}" "pass""#)).await?;
    read_line(&mut lines).await?;
    write_cmd(&mut w, r#"a2 SELECT "INBOX""#).await?;
    while let Some(line) = lines.next_line().await? {
        if line.starts_with("a2 OK") {
            break;
        }
    }
    write_cmd(&mut w, "a3 FETCH 1 BODY[]").await?;
    let mut body = String::new();
    while let Some(line) = lines.next_line().await? {
        body.push_str(&line);
        body.push('\n');
        if line.starts_with("a3 OK") {
            break;
        }
    }
    Ok(body)
}

async fn write_cmd(w: &mut tokio::net::tcp::OwnedWriteHalf, cmd: &str) -> std::io::Result<()> {
    w.write_all(format!("{cmd}\r\n").as_bytes()).await
}

async fn read_line(lines: &mut tokio::io::Lines<BufReader<tokio::net::tcp::OwnedReadHalf>>) -> std::io::Result<String> {
    lines.next_line().await?.ok_or_else(|| {
        std::io::Error::new(std::io::ErrorKind::UnexpectedEof, "closed")
    })
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
