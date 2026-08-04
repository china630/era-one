//! Minimal HTTP admin (health + status) для mail-api bridge (CM1-7).

use crate::store::MailBackend;
use anyhow::Result;
use std::sync::Arc;
use tokio::io::{AsyncReadExt, AsyncWriteExt};
use tokio::net::TcpListener;

const VERSION: &str = "era-mail-core/0.2.0";

pub async fn serve(addr: &str, store: Arc<dyn MailBackend>) -> Result<()> {
    let listener = TcpListener::bind(addr).await?;
    loop {
        let (mut stream, _) = listener.accept().await?;
        let store = Arc::clone(&store);
        tokio::spawn(async move {
            let mut buf = [0u8; 1024];
            let n = stream.read(&mut buf).await.unwrap_or(0);
            let req = String::from_utf8_lossy(&buf[..n]);
            let (status, body) = if req.contains("GET /healthz") {
                (200, r#"{"status":"ok","service":"era-mail-core"}"#.to_string())
            } else if req.contains("GET /readyz") {
                (200, r#"{"ready":true}"#.to_string())
            } else if req.contains("GET /api/v1/status") {
                let count = store.message_count();
                (
                    200,
                    format!(
                        r#"{{"smtp_ready":true,"imap_ready":true,"messages_stored":{count},"version":"{VERSION}"}}"#
                    ),
                )
            } else {
                (404, r#"{"error":"not found"}"#.to_string())
            };
            let resp = format!(
                "HTTP/1.1 {status} OK\r\nContent-Type: application/json\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{body}",
                body.len()
            );
            let _ = stream.write_all(resp.as_bytes()).await;
        });
    }
}
