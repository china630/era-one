//! Fire-and-forget audit webhook → mail-api (CM1-7, AC-C7).

use tracing::warn;

#[derive(Clone)]
pub struct AuditHook {
    target: Option<(String, String)>, // (host:port, path)
    default_tenant: String,
}

impl AuditHook {
    pub fn from_env() -> Self {
        let target = std::env::var("ERA_MAIL_AUDIT_URL")
            .ok()
            .and_then(parse_audit_url);
        Self {
            target,
            default_tenant: std::env::var("ERA_MAIL_DEFAULT_TENANT")
                .unwrap_or_else(|_| "t-demo".into()),
        }
    }

    pub fn notify_send(&self, mailbox: &str, mail_from: &str) {
        let Some((host, path)) = self.target.clone() else {
            return;
        };
        let mailbox = mailbox.to_string();
        let mail_from = mail_from.to_string();
        let tenant = self.default_tenant.clone();
        tokio::spawn(async move {
            if let Err(e) = post_audit(&host, &path, &tenant, &mailbox, &mail_from).await {
                warn!(error = %e, "audit webhook failed");
            }
        });
    }
}

fn parse_audit_url(raw: String) -> Option<(String, String)> {
    let raw = raw.trim();
    let without_scheme = raw
        .strip_prefix("http://")
        .or_else(|| raw.strip_prefix("https://"))
        .unwrap_or(raw);
    let (authority, path) = match without_scheme.split_once('/') {
        Some((a, p)) => (a.to_string(), format!("/{p}")),
        None => (without_scheme.to_string(), "/internal/v1/audit".into()),
    };
    if authority.is_empty() {
        return None;
    }
    Some((authority, path))
}

async fn post_audit(
    host: &str,
    path: &str,
    tenant: &str,
    mailbox: &str,
    mail_from: &str,
) -> anyhow::Result<()> {
    let body = format!(
        r#"{{"tenant_id":"{tenant}","mailbox":"{mailbox}","action":"send","mail_from":"{mail_from}"}}"#
    );
    let mut stream = tokio::net::TcpStream::connect(host).await?;
    let req = format!(
        "POST {path} HTTP/1.1\r\nHost: {host}\r\nContent-Type: application/json\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{body}",
        body.len()
    );
    tokio::io::AsyncWriteExt::write_all(&mut stream, req.as_bytes()).await?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parse_audit_url_with_path() {
        let p = parse_audit_url("http://127.0.0.1:8150/internal/v1/audit".into()).unwrap();
        assert_eq!(p.0, "127.0.0.1:8150");
        assert_eq!(p.1, "/internal/v1/audit");
    }
}
