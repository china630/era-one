//! Конфигурация агента из переменных окружения.

use crate::plugin::PluginManifest;
use anyhow::{Context, Result};
use era_license::LicenseError;
use std::path::PathBuf;

#[derive(Debug, Clone)]
pub struct Config {
    pub gateway_addr: String,
    pub tenant_id: String,
    pub node_id: String,
    pub agent_id: String,
    pub hostname: String,
    pub pseudonym_key: String,
    pub batch_size: usize,
    pub poll_interval_secs: u64,
    pub control_plane_url: String,
    pub heartbeat_secs: u64,
    pub license_token: Option<String>,
    pub vendor_pub_b64: Option<String>,
    pub plugin_dir: PathBuf,
    pub ota_mirror_dir: Option<PathBuf>,
    /// Dev: разрешить плагины без license token (только локальная разработка).
    pub dev_allow_plugins: bool,
}

impl Config {
    pub fn from_env() -> Result<Self> {
        Ok(Self {
            gateway_addr: env("ERA_GATEWAY_ADDR", "http://127.0.0.1:50051"),
            tenant_id: env("ERA_TENANT_ID", "tenant-dev"),
            node_id: env("ERA_NODE_ID", "node-01"),
            agent_id: env("ERA_AGENT_ID", "agent-0001"),
            hostname: env("ERA_HOSTNAME", &hostname_default()),
            pseudonym_key: env("ERA_PSEUDONYM_KEY", "dev-tenant-key-for-tests-only"),
            batch_size: env("ERA_BATCH_SIZE", "500").parse().context("ERA_BATCH_SIZE")?,
            poll_interval_secs: env("ERA_POLL_SECS", "2").parse().context("ERA_POLL_SECS")?,
            control_plane_url: env("ERA_CONTROL_PLANE_URL", "http://127.0.0.1:8090"),
            heartbeat_secs: env("ERA_HEARTBEAT_SECS", "30").parse().context("ERA_HEARTBEAT_SECS")?,
            license_token: optional_env("ERA_LICENSE_TOKEN"),
            vendor_pub_b64: optional_env("ERA_VENDOR_PUB"),
            plugin_dir: PathBuf::from(env("ERA_PLUGIN_DIR", ".")),
            ota_mirror_dir: optional_env("ERA_OTA_MIRROR").map(PathBuf::from),
            dev_allow_plugins: Self::dev_allow_plugins_from_env(),
        })
    }

    fn dev_allow_plugins_from_env() -> bool {
        if std::env::var("ERA_PRODUCTION")
            .map(|v| v == "1" || v.eq_ignore_ascii_case("true"))
            .unwrap_or(false)
        {
            return std::env::var("ERA_DEV_ALLOW_PLUGINS")
                .map(|v| v == "1" || v.eq_ignore_ascii_case("true"))
                .unwrap_or(false);
        }
        env("ERA_DEV_ALLOW_PLUGINS", "1") == "1"
    }

    pub fn dev_defaults() -> Self {
        Self {
            gateway_addr: "http://127.0.0.1:50051".into(),
            tenant_id: "tenant-dev".into(),
            node_id: "node-01".into(),
            agent_id: "agent-0001".into(),
            hostname: "win-host-01".into(),
            pseudonym_key: "dev-tenant-key-for-tests-only".into(),
            batch_size: 500,
            poll_interval_secs: 2,
            control_plane_url: "http://127.0.0.1:8090".into(),
            heartbeat_secs: 30,
            license_token: None,
            vendor_pub_b64: None,
            plugin_dir: PathBuf::from("."),
            ota_mirror_dir: None,
            dev_allow_plugins: true,
        }
    }

    pub fn vendor_public_key_bytes(&self) -> Result<[u8; 32], era_license::LicenseError> {
        let b64 = self
            .vendor_pub_b64
            .as_deref()
            .ok_or(LicenseError::Key)?;
        use base64::engine::general_purpose::STANDARD;
        use base64::Engine;
        let bytes = STANDARD
            .decode(b64.trim())
            .or_else(|_| {
                use base64::engine::general_purpose::URL_SAFE_NO_PAD;
                URL_SAFE_NO_PAD.decode(b64.trim())
            })
            .map_err(|_| LicenseError::Key)?;
        bytes
            .as_slice()
            .try_into()
            .map_err(|_| LicenseError::Key)
    }

    pub fn load_plugin_manifests(&self) -> Vec<PluginManifest> {
        if let Ok(path) = std::env::var("ERA_PLUGIN_MANIFEST") {
            if let Ok(data) = std::fs::read_to_string(&path) {
                if let Ok(m) = serde_json::from_str::<PluginManifest>(&data) {
                    return vec![m];
                }
            }
        }
        if std::env::var("ERA_ENABLE_INVENTORY_PLUGIN")
            .map(|v| v == "1" || v.eq_ignore_ascii_case("true"))
            .unwrap_or(true)
        {
            vec![PluginManifest::inventory_dev()]
        } else {
            vec![]
        }
    }
}

fn env(key: &str, default: &str) -> String {
    std::env::var(key).unwrap_or_else(|_| default.to_string())
}

fn optional_env(key: &str) -> Option<String> {
    std::env::var(key).ok().filter(|s| !s.is_empty())
}

fn hostname_default() -> String {
    std::env::var("COMPUTERNAME")
        .or_else(|_| std::env::var("HOSTNAME"))
        .unwrap_or_else(|_| "unknown-host".into())
}
