//! Загрузка enforcement policy из control-plane (offline cache на агенте).

use crate::enforce::policy::EnforcementPolicy;
use anyhow::{Context, Result};

pub fn fetch_policy(control_plane_url: &str) -> Result<EnforcementPolicy> {
    let url = format!(
        "{}/api/v1/enforcement/policy",
        control_plane_url.trim_end_matches('/')
    );
    let resp = ureq::get(&url)
        .set("X-ERA-Actor", "era-agent")
        .set("X-ERA-Role", "admin")
        .call()
        .context("GET enforcement policy")?;
    let body = resp.into_string().context("read policy body")?;
    let parsed: serde_json::Value = serde_json::from_str(&body).context("policy json")?;
    if let Some(p) = parsed.get("policy") {
        serde_json::from_value(p.clone()).context("policy field")
    } else {
        serde_json::from_str(&body).context("policy root")
    }
}
