//! Remote store via mail-api internal HTTP (ADR-0029).

use crate::store::{MailBackend, StoredMessage};
use anyhow::{anyhow, Result};
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

fn internal_headers(req: reqwest::blocking::RequestBuilder) -> reqwest::blocking::RequestBuilder {
    if let Ok(tok) = std::env::var("ERA_INTERNAL_TOKEN") {
        if !tok.trim().is_empty() {
            return req.header("X-ERA-Internal-Token", tok);
        }
    }
    req
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

    /// Fetch tenant policy max message bytes for mailbox (G1-4). Falls back to env/default.
    pub fn max_message_bytes_for(&self, mailbox: &str) -> Option<usize> {
        let url = format!(
            "{}/internal/v1/mail/policy?email={}",
            self.base,
            urlencoding::encode(mailbox)
        );
        blocking_call(|| {
            let resp = internal_headers(http_client().get(&url)).send().ok()?;
            if !resp.status().is_success() {
                return None;
            }
            let v: serde_json::Value = resp.json().ok()?;
            v.get("max_message_bytes")
                .and_then(|x| x.as_u64())
                .map(|n| n as usize)
        })
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
        let status_body = blocking_call(|| -> Result<(reqwest::StatusCode, String)> {
            let resp = internal_headers(http_client().post(&url).json(&serde_json::json!({
                "email": mailbox,
                "from": mailbox,
                "raw": raw,
            })))
            .send()?;
            let status = resp.status();
            let text = resp.text().unwrap_or_default();
            Ok((status, text))
        })?;
        if status_body.0.as_u16() == 413 {
            return Err(anyhow!("message too large"));
        }
        if !status_body.0.is_success() {
            return Err(anyhow!("deliver {}: {}", status_body.0, status_body.1));
        }
        let resp: DeliverResp = serde_json::from_str(&status_body.1)
            .map_err(|e| anyhow!("deliver json: {e}"))?;
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
        let resp: ListResp = blocking_call(|| {
            internal_headers(http_client().get(&url))
                .send()?
                .error_for_status()?
                .json()
        })?;
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
            match internal_headers(http_client().post(&url).json(&serde_json::json!({
                "email": mailbox,
                "password": password,
            })))
            .send()
            {
                Ok(r) => r.json::<VerifyResp>().map(|v| v.ok).unwrap_or(false),
                Err(_) => false,
            }
        })
    }

    fn max_message_bytes(&self, mailbox: &str) -> Option<usize> {
        self.max_message_bytes_for(mailbox)
    }
}

pub fn open_backend() -> Result<std::sync::Arc<dyn MailBackend>> {
    let store_mode = std::env::var("ERA_MAIL_STORE").unwrap_or_default();
    if store_mode.eq_ignore_ascii_case("memory") {
        return Ok(std::sync::Arc::new(crate::store::MailStore::new()));
    }
    if let Some(remote) = RemoteStore::from_env() {
        return Ok(std::sync::Arc::new(remote));
    }
    // Without ERA_MAIL_API_URL, in-process memory for unit/e2e tests.
    Ok(std::sync::Arc::new(crate::store::MailStore::new()))
}
