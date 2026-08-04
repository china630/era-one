//! Desktop profile config (Solo vs Corporate). S4/B-corp.
//! Targets: Browser ❌ · Solo ✅ · Corporate ✅

use std::fs;
use std::path::{Path, PathBuf};

use serde::{Deserialize, Serialize};
use thiserror::Error;

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, Default)]
#[serde(rename_all = "snake_case")]
pub enum Profile {
    #[default]
    Solo,
    Corporate,
}

impl Profile {
    pub fn parse(s: &str) -> Option<Self> {
        match s.trim().to_ascii_lowercase().as_str() {
            "solo" | "local" => Some(Self::Solo),
            "corporate" | "corp" | "tenant" => Some(Self::Corporate),
            _ => None,
        }
    }

    pub fn as_str(self) -> &'static str {
        match self {
            Self::Solo => "solo",
            Self::Corporate => "corporate",
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct DesktopConfig {
    pub profile: Profile,
    /// Tenant Workspace base URL (Corporate). Example: `https://office.example.gov`
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub server_url: Option<String>,
}

impl Default for DesktopConfig {
    fn default() -> Self {
        Self {
            profile: Profile::Solo,
            server_url: None,
        }
    }
}

#[derive(Debug, Error)]
pub enum ConfigError {
    #[error("{0}")]
    Msg(String),
    #[error(transparent)]
    Io(#[from] std::io::Error),
    #[error(transparent)]
    Json(#[from] serde_json::Error),
}

impl DesktopConfig {
    pub fn normalize_server_url(raw: &str) -> Result<String, ConfigError> {
        let t = raw.trim().trim_end_matches('/');
        if t.is_empty() {
            return Err(ConfigError::Msg("server_url is empty".into()));
        }
        let with_scheme = if t.contains("://") {
            t.to_string()
        } else {
            format!("https://{t}")
        };
        let url = url::Url::parse(&with_scheme)
            .map_err(|e| ConfigError::Msg(format!("invalid server_url: {e}")))?;
        if url.scheme() != "http" && url.scheme() != "https" {
            return Err(ConfigError::Msg(
                "server_url must be http(s)".into(),
            ));
        }
        if url.host_str().is_none() {
            return Err(ConfigError::Msg("server_url missing host".into()));
        }
        Ok(with_scheme.trim_end_matches('/').to_string())
    }

    pub fn with_env_overrides(mut self) -> Self {
        if let Ok(p) = std::env::var("ERA_OFFICE_PROFILE") {
            if let Some(prof) = Profile::parse(&p) {
                self.profile = prof;
            }
        }
        if let Ok(u) = std::env::var("ERA_OFFICE_SERVER_URL") {
            if !u.trim().is_empty() {
                if let Ok(n) = Self::normalize_server_url(&u) {
                    self.server_url = Some(n);
                }
            }
        }
        self
    }

    pub fn corporate_ready(&self) -> bool {
        self.profile == Profile::Corporate
            && self
                .server_url
                .as_ref()
                .map(|s| !s.is_empty())
                .unwrap_or(false)
    }
}

pub fn default_config_path() -> Option<PathBuf> {
    dirs::config_dir().map(|d| d.join("era-office-desktop").join("config.json"))
}

pub fn load_from_path(path: &Path) -> Result<DesktopConfig, ConfigError> {
    if !path.exists() {
        return Ok(DesktopConfig::default().with_env_overrides());
    }
    let raw = fs::read_to_string(path)?;
    let cfg: DesktopConfig = serde_json::from_str(&raw)?;
    Ok(cfg.with_env_overrides())
}

pub fn save_to_path(path: &Path, cfg: &DesktopConfig) -> Result<(), ConfigError> {
    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent)?;
    }
    let json = serde_json::to_string_pretty(cfg)?;
    fs::write(path, json)?;
    Ok(())
}

pub fn load() -> Result<DesktopConfig, ConfigError> {
    match default_config_path() {
        Some(p) => load_from_path(&p),
        None => Ok(DesktopConfig::default().with_env_overrides()),
    }
}

pub fn save(cfg: &DesktopConfig) -> Result<(), ConfigError> {
    let path = default_config_path()
        .ok_or_else(|| ConfigError::Msg("no config dir".into()))?;
    save_to_path(&path, cfg)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn normalize_adds_https() {
        let u = DesktopConfig::normalize_server_url("office.example.gov/").unwrap();
        assert_eq!(u, "https://office.example.gov");
    }

    #[test]
    fn normalize_keeps_http() {
        let u = DesktopConfig::normalize_server_url("http://127.0.0.1:8170").unwrap();
        assert_eq!(u, "http://127.0.0.1:8170");
    }

    #[test]
    fn reject_bad_scheme() {
        assert!(DesktopConfig::normalize_server_url("ftp://x").is_err());
    }

    #[test]
    fn roundtrip_file() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("config.json");
        let cfg = DesktopConfig {
            profile: Profile::Corporate,
            server_url: Some("https://tenant.example".into()),
        };
        save_to_path(&path, &cfg).unwrap();
        let loaded = load_from_path(&path).unwrap();
        assert_eq!(loaded.profile, Profile::Corporate);
        assert_eq!(loaded.server_url.as_deref(), Some("https://tenant.example"));
        assert!(loaded.corporate_ready());
    }

    #[test]
    fn profile_parse() {
        assert_eq!(Profile::parse("CORP"), Some(Profile::Corporate));
        assert_eq!(Profile::parse("solo"), Some(Profile::Solo));
    }
}
