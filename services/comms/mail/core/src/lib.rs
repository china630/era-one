//! ERA Mail Server — in-memory SMTP/IMAP core with optional remote store (ADR-0029).

pub mod admin;
pub mod audit_hook;
pub mod config;
pub mod imap;
pub mod remote_store;
pub mod smtp;
pub mod store;
pub mod tls_util;

pub use config::Config;
pub use store::{MailBackend, MailStore, StoredMessage};

use anyhow::Result;
use audit_hook::AuditHook;
use std::sync::Arc;
use tokio::task::JoinHandle;
use tracing::info;

/// Запускает SMTP, IMAP и admin HTTP listeners.
pub async fn run(cfg: Config, store: Arc<dyn MailBackend>) -> Result<()> {
    let audit = AuditHook::from_env();

    let smtp_cfg = cfg.clone();
    let smtp_store = Arc::clone(&store);
    let smtp_audit = audit.clone();
    let smtp: JoinHandle<Result<()>> = tokio::spawn(async move {
        smtp::serve(&smtp_cfg.smtp_addr(), smtp_store, smtp_audit).await
    });

    let imap_cfg = cfg.clone();
    let imap_store = Arc::clone(&store);
    let imap: JoinHandle<Result<()>> = tokio::spawn(async move {
        imap::serve(&imap_cfg.imap_addr(), imap_store).await
    });

    let admin_cfg = cfg.clone();
    let admin_store = Arc::clone(&store);
    let admin: JoinHandle<Result<()>> = tokio::spawn(async move {
        admin::serve(&admin_cfg.admin_addr(), admin_store).await
    });

    info!(
        smtp = %cfg.smtp_addr(),
        imap = %cfg.imap_addr(),
        admin = %cfg.admin_addr(),
        remote = %std::env::var("ERA_MAIL_API_URL").unwrap_or_else(|_| "no".into()),
        "era-mail-core started"
    );

    let (s, i, a) = tokio::try_join!(smtp, imap, admin)?;
    s?;
    i?;
    a?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn store_deliver_and_fetch() {
        let s = MailStore::new();
        let id = s
            .deliver("bob@mail.gov.az", "From: alice@test\r\n\r\nHello")
            .unwrap();
        let msgs = s.list_mailbox("bob@mail.gov.az").unwrap();
        assert_eq!(msgs.len(), 1);
        assert_eq!(msgs[0].id, id);
    }

    #[test]
    fn store_password_gate() {
        let s = MailStore::new();
        s.set_password("alice@mail.gov.az", "secret");
        assert!(s.verify_login("alice@mail.gov.az", "secret"));
        assert!(!s.verify_login("alice@mail.gov.az", "wrong"));
    }
}
