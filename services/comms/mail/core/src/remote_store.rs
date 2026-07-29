//! Remote store via mail-api internal HTTP (ADR-0029).

use crate::store::{MailBackend, StoredMessage};
use anyhow::Result;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::OnceLock;
use std::time::Duration;

fn http_client() -> &'static reqwest::blocking::Client {
    static CLIENT: OnceLock<reqwest::blocking::Client> = OnceLock::new();
    CLIENT.get_or_init(|| {
        reqwest::blocking::Client::builder()
            .timeout(Duration::from_secs(5))
            .build()
            .unwrap_or_default()
    })
}

/// Must run from sync main before tokio runtime (reqwest::blocking invariant).
pub fn init_blocking_http() {
    let _ = http_client();
}

fn blocking_call<F, T>(f: F) -> T
where
    F: FnOnce() -> T,
{
    if tokio::runtime::Handle::try_current().is_ok() {
        tokio::task::block_in_place(f)
    } else {
        f()
    }
}

/// HTTP-backed store delegating to Go mail-api.
pub struct RemoteStore {
    base: String,
    count: AtomicU64,
}

impl RemoteStore {
    pub fn new(base: String) -> Self {
        let base = base.trim_end_matches('/').to_string();
        Self {
            base,
            count: AtomicU64::new(0),
        }
    }

    pub fn from_env() -> Option<Self> {
        let base = std::env::var("ERA_MAIL_API_URL").ok()?;
        if base.trim().is_empty() {
            return None;
        }
        Some(Self::new(base))
    }
}

#[derive(serde::Deserialize)]
struct DeliverResp {
    id: u64,
}

#[derive(serde::Deserialize)]
struct ListResp {
    messages: Vec<RemoteMsg>,
}

#[derive(serde::Deserialize)]
struct RemoteMsg {
    id: u64,
    uid: u64,
    raw: String,
}

#[derive(serde::Deserialize)]
struct VerifyResp {
    ok: bool,
}

impl MailBackend for RemoteStore {
    fn deliver(&self, mailbox: &str, raw: &str) -> Result<u64> {
        let url = format!("{}/internal/v1/mail/deliver", self.base);
        let mailbox = mailbox.to_string();
        let raw = raw.to_string();
        let resp: DeliverResp = blocking_call(|| {
            http_client()
                .post(&url)
                .json(&serde_json::json!({
                    "email": mailbox,
                    "from": mailbox,
                    "raw": raw,
                }))
                .send()?
                .error_for_status()?
                .json()
        })?;
        self.count.fetch_add(1, Ordering::SeqCst);
        Ok(resp.id)
    }

    fn list_mailbox(&self, mailbox: &str) -> Result<Vec<StoredMessage>> {
        let url = format!(
            "{}/internal/v1/mail/list?email={}",
            self.base,
            urlencoding::encode(mailbox)
        );
        let mailbox_lc = mailbox.to_lowercase();
        let resp: ListResp =
            blocking_call(|| http_client().get(&url).send()?.error_for_status()?.json())?;
        Ok(resp
            .messages
            .into_iter()
            .map(|m| StoredMessage {
                id: m.id,
                uid: m.uid,
                mailbox: mailbox_lc.clone(),
                raw: m.raw.into_bytes(),
            })
            .collect())
    }

    fn message_count(&self) -> u64 {
        self.count.load(Ordering::SeqCst)
    }

    fn verify_login(&self, mailbox: &str, password: &str) -> bool {
        let url = format!("{}/internal/v1/auth/verify", self.base);
        let mailbox = mailbox.to_string();
        let password = password.to_string();
        blocking_call(|| {
            match http_client()
                .post(&url)
                .json(&serde_json::json!({
                    "email": mailbox,
                    "password": password,
                }))
                .send()
            {
                Ok(r) => r.json::<VerifyResp>().map(|v| v.ok).unwrap_or(false),
                Err(_) => false,
            }
        })
    }
}

pub fn open_backend() -> Result<std::sync::Arc<dyn MailBackend>> {
    if let Some(remote) = RemoteStore::from_env() {
        return Ok(std::sync::Arc::new(remote));
    }
    Ok(std::sync::Arc::new(crate::store::MailStore::new()))
}
