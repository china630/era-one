//! Mail persistence backends (memory + remote mail-api).

use anyhow::{anyhow, Result};
use dashmap::DashMap;
use std::sync::atomic::{AtomicU64, Ordering};

/// Сохранённое сообщение.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct StoredMessage {
    pub id: u64,
    pub uid: u64,
    pub mailbox: String,
    pub raw: Vec<u8>,
}

/// Backend trait for SMTP/IMAP.
pub trait MailBackend: Send + Sync {
    fn deliver(&self, mailbox: &str, raw: &str) -> Result<u64>;
    fn list_mailbox(&self, mailbox: &str) -> Result<Vec<StoredMessage>>;
    fn message_count(&self) -> u64;
    fn verify_login(&self, mailbox: &str, password: &str) -> bool;
    /// Optional tenant policy max message size (bytes). None → env/default.
    fn max_message_bytes(&self, _mailbox: &str) -> Option<usize> {
        None
    }
}

/// Thread-safe in-memory store (unit tests, no ERA_MAIL_API_URL).
pub struct MailStore {
    next_id: AtomicU64,
    mailboxes: DashMap<String, Vec<StoredMessage>>,
    passwords: DashMap<String, String>,
    uid_next: DashMap<String, AtomicU64>,
}

impl MailStore {
    pub fn new() -> Self {
        Self {
            next_id: AtomicU64::new(1),
            mailboxes: DashMap::new(),
            passwords: DashMap::new(),
            uid_next: DashMap::new(),
        }
    }

    /// Sets expected password for mailbox (tests / dev).
    pub fn set_password(&self, mailbox: &str, password: &str) {
        self.passwords
            .insert(mailbox.trim().to_lowercase(), password.to_string());
    }
}

impl MailBackend for MailStore {
    fn deliver(&self, mailbox: &str, raw: &str) -> Result<u64> {
        let mailbox = mailbox.trim().to_lowercase();
        if !mailbox.contains('@') {
            return Err(anyhow!("invalid mailbox"));
        }
        let id = self.next_id.fetch_add(1, Ordering::SeqCst);
        let uid_entry = self.uid_next.entry(mailbox.clone()).or_default();
        let uid = uid_entry.fetch_add(1, Ordering::SeqCst) + 1;
        let msg = StoredMessage {
            id,
            uid,
            mailbox: mailbox.clone(),
            raw: raw.as_bytes().to_vec(),
        };
        self.mailboxes.entry(mailbox).or_default().push(msg);
        Ok(id)
    }

    fn list_mailbox(&self, mailbox: &str) -> Result<Vec<StoredMessage>> {
        let mailbox = mailbox.trim().to_lowercase();
        Ok(self
            .mailboxes
            .get(&mailbox)
            .map(|v| v.clone())
            .unwrap_or_default())
    }

    fn message_count(&self) -> u64 {
        self.next_id.load(Ordering::SeqCst).saturating_sub(1)
    }

    fn verify_login(&self, mailbox: &str, password: &str) -> bool {
        let mailbox = mailbox.trim().to_lowercase();
        match self.passwords.get(&mailbox) {
            Some(expected) => expected.as_str() == password,
            None => true,
        }
    }
}

impl Default for MailStore {
    fn default() -> Self {
        Self::new()
    }
}
