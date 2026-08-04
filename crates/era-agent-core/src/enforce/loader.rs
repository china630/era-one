//! Загрузка enforcement policy из control-plane (offline cache на агенте).

use crate::enforce::policy::EnforcementPolicy;
use anyhow::{bail, Context, Result};

/// Строит заголовки для GET enforcement policy (без forge `X-ERA-Role: admin`).
/// В production/strict обязателен `ERA_API_KEY` или `ERA_AGENT_TOKEN`.
pub fn policy_auth_headers() -> Result<Vec<(String, String)>> {
    let mut headers = vec![("X-ERA-Actor".into(), "era-agent".into())];
    match agent_bearer_token() {
        Some(token) => {
            headers.push(("Authorization".into(), format!("Bearer {token}")));
        }
        None if requires_agent_token() => {
            bail!("ERA_API_KEY or ERA_AGENT_TOKEN required for policy fetch in production/strict");
        }
        None => {
            // Lab (TrustDev): actor alone; role header is not a credential.
        }
    }
    Ok(headers)
}

fn agent_bearer_token() -> Option<String> {
    for key in ["ERA_API_KEY", "ERA_AGENT_TOKEN"] {
        if let Ok(v) = std::env::var(key) {
            let t = v.trim();
            if !t.is_empty() {
                return Some(t.to_string());
            }
        }
    }
    None
}

fn requires_agent_token() -> bool {
    env_truthy("ERA_PRODUCTION")
        || env_truthy("ERA_LICENSE_STRICT")
        || env_truthy("ERA_ENV_PRODUCTION")
        || std::env::var("ERA_ENV")
            .map(|v| v.eq_ignore_ascii_case("production"))
            .unwrap_or(false)
}

fn env_truthy(k: &str) -> bool {
    std::env::var(k)
        .map(|v| {
            let v = v.trim().to_ascii_lowercase();
            v == "1" || v == "true" || v == "yes"
        })
        .unwrap_or(false)
}

pub fn fetch_policy(control_plane_url: &str) -> Result<EnforcementPolicy> {
    let url = format!(
        "{}/api/v1/enforcement/policy",
        control_plane_url.trim_end_matches('/')
    );
    let mut req = ureq::get(&url);
    for (k, v) in policy_auth_headers()? {
        req = req.set(&k, &v);
    }
    let resp = req.call().context("GET enforcement policy")?;
    let body = resp.into_string().context("read policy body")?;
    let parsed: serde_json::Value = serde_json::from_str(&body).context("policy json")?;
    if let Some(p) = parsed.get("policy") {
        serde_json::from_value(p.clone()).context("policy field")
    } else {
        serde_json::from_str(&body).context("policy root")
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn lab_headers_actor_only_no_forged_admin() {
        std::env::remove_var("ERA_API_KEY");
        std::env::remove_var("ERA_AGENT_TOKEN");
        std::env::remove_var("ERA_PRODUCTION");
        std::env::remove_var("ERA_LICENSE_STRICT");
        std::env::remove_var("ERA_ENV");
        std::env::remove_var("ERA_ENV_PRODUCTION");
        let h = policy_auth_headers().expect("lab ok");
        assert!(h.iter().any(|(k, v)| k == "X-ERA-Actor" && v == "era-agent"));
        assert!(!h.iter().any(|(k, _)| k == "X-ERA-Role"));
        assert!(!h.iter().any(|(k, _)| k == "Authorization"));
    }

    #[test]
    fn bearer_from_api_key() {
        std::env::set_var("ERA_API_KEY", "secret-key");
        std::env::remove_var("ERA_AGENT_TOKEN");
        std::env::remove_var("ERA_PRODUCTION");
        let h = policy_auth_headers().expect("ok");
        assert!(h
            .iter()
            .any(|(k, v)| k == "Authorization" && v == "Bearer secret-key"));
        std::env::remove_var("ERA_API_KEY");
    }

    #[test]
    fn bearer_from_agent_token_when_no_api_key() {
        std::env::remove_var("ERA_API_KEY");
        std::env::set_var("ERA_AGENT_TOKEN", "agent-tok");
        std::env::remove_var("ERA_PRODUCTION");
        let h = policy_auth_headers().expect("ok");
        assert!(h
            .iter()
            .any(|(k, v)| k == "Authorization" && v == "Bearer agent-tok"));
        std::env::remove_var("ERA_AGENT_TOKEN");
    }

    #[test]
    fn production_requires_token() {
        std::env::remove_var("ERA_API_KEY");
        std::env::remove_var("ERA_AGENT_TOKEN");
        std::env::set_var("ERA_PRODUCTION", "1");
        let err = policy_auth_headers().expect_err("must fail closed");
        assert!(err.to_string().contains("ERA_API_KEY") || err.to_string().contains("ERA_AGENT_TOKEN"));
        std::env::remove_var("ERA_PRODUCTION");
    }
}
