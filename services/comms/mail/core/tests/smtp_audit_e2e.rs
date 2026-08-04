//! E2E: SMTP DATA triggers audit webhook (AC-C7 path).
//! Requires ERA_MAIL_AUDIT_URL pointing at mail-api audit endpoint.

use era_mail_core::{audit_hook::AuditHook, smtp, store::MailStore, MailBackend};
use std::sync::Arc;
use std::time::Duration;
use tokio::io::{AsyncBufReadExt, AsyncWriteExt, BufReader};
use tokio::net::TcpStream;
use tokio::time::{sleep, timeout};

#[tokio::test]
async fn smtp_triggers_audit_webhook() {
    let audit_url = match std::env::var("ERA_MAIL_AUDIT_URL") {
        Ok(u) if !u.is_empty() => u,
        _ => {
            eprintln!("skip smtp_triggers_audit_webhook: ERA_MAIL_AUDIT_URL not set");
            return;
        }
    };
    std::env::set_var("ERA_MAIL_AUDIT_URL", &audit_url);

    let store = Arc::new(MailStore::new());
    let audit = AuditHook::from_env();

    let smtp_l = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let smtp_port = smtp_l.local_addr().unwrap().port();

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

    timeout(
        Duration::from_secs(5),
        smtp_send(smtp_port, "alice@mail.gov.az", "bob@mail.gov.az"),
    )
    .await
    .expect("smtp timeout")
    .expect("smtp failed");

    sleep(Duration::from_millis(300)).await;
    assert_eq!(store.message_count(), 1);
}

async fn smtp_send(port: u16, from: &str, to: &str) -> std::io::Result<()> {
    let s = TcpStream::connect(format!("127.0.0.1:{port}")).await?;
    let (r, mut w) = s.into_split();
    let mut lines = BufReader::new(r).lines();
    read_line(&mut lines).await?;
    write_cmd(&mut w, "EHLO audit.e2e").await?;
    loop {
        let line = read_line(&mut lines).await?;
        if line.starts_with("250 ") && !line.contains('-') {
            break;
        }
    }
    let auth = format!("\0{from}\0pass");
    write_cmd(&mut w, &format!("AUTH PLAIN {}", base64_encode(auth.as_bytes()))).await?;
    read_line(&mut lines).await?;
    write_cmd(&mut w, &format!("MAIL FROM:<{from}>")).await?;
    read_line(&mut lines).await?;
    write_cmd(&mut w, &format!("RCPT TO:<{to}>")).await?;
    read_line(&mut lines).await?;
    write_cmd(&mut w, "DATA").await?;
    read_line(&mut lines).await?;
    write_cmd(&mut w, "Subject: audit\r\n\r\nAudit path test\r\n.").await?;
    read_line(&mut lines).await?;
    write_cmd(&mut w, "QUIT").await?;
    Ok(())
}

async fn write_cmd(w: &mut tokio::net::tcp::OwnedWriteHalf, cmd: &str) -> std::io::Result<()> {
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
