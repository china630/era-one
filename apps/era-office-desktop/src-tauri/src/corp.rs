//! Corporate shell helpers — Workspace URL + deep links (S4/B-corp v1).
//! Targets: Browser ❌ · Solo ❌ · Corporate ✅
//!
//! v1 does **not** re-host docs-engine locally: WebView loads tenant Workspace;
//! SSO is the tenant `/login` in that WebView (same JWT as browser).

use serde::{Deserialize, Serialize};
use thiserror::Error;
use url::Url;

use crate::config::DesktopConfig;

#[derive(Debug, Error)]
pub enum CorpError {
    #[error("{0}")]
    Msg(String),
}

/// Result of parsing `era-office://open?…`
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct CorpOpenTarget {
    /// Absolute http(s) URL to navigate, or path under server_url.
    pub href: String,
}

/// Build navigation URL from config + optional relative path (`/docs/…`).
pub fn workspace_href(cfg: &DesktopConfig, path: Option<&str>) -> Result<String, CorpError> {
    let base = cfg
        .server_url
        .as_ref()
        .ok_or_else(|| CorpError::Msg("server_url not configured".into()))?;
    let base = base.trim_end_matches('/');
    match path {
        None | Some("") | Some("/") => Ok(format!("{base}/drive/")),
        Some(p) if p.starts_with("http://") || p.starts_with("https://") => Ok(p.to_string()),
        Some(p) => {
            let p = if p.starts_with('/') {
                p.to_string()
            } else {
                format!("/{p}")
            };
            Ok(format!("{base}{p}"))
        }
    }
}

/// Parse deep link / CLI arg.
///
/// Supported:
/// - `era-office://open?url=https%3A%2F%2F…`
/// - `era-office://open?path=/docs/{id}`
/// - `era-office:open?path=/drive/`
pub fn parse_open_url(raw: &str) -> Result<Option<CorpOpenTarget>, CorpError> {
    let t = raw.trim();
    if t.is_empty() {
        return Ok(None);
    }
    if t.starts_with("http://") || t.starts_with("https://") {
        return Ok(Some(CorpOpenTarget { href: t.to_string() }));
    }
    if !(t.starts_with("era-office:") || t.starts_with("era-office://")) {
        return Ok(None);
    }
    let normalized = if t.starts_with("era-office://") {
        t.to_string()
    } else {
        t.replacen("era-office:", "era-office://", 1)
    };
    let url = Url::parse(&normalized)
        .map_err(|e| CorpError::Msg(format!("bad deep link: {e}")))?;
    let is_open = url.host_str() == Some("open") || url.path().trim_matches('/') == "open";
    if !is_open {
        return Err(CorpError::Msg(
            "deep link must be era-office://open?…".into(),
        ));
    }
    if let Some((_, v)) = url.query_pairs().find(|(k, _)| k == "url") {
        let href = v.to_string();
        if !(href.starts_with("http://") || href.starts_with("https://")) {
            return Err(CorpError::Msg("url= must be http(s)".into()));
        }
        return Ok(Some(CorpOpenTarget { href }));
    }
    if let Some((_, v)) = url.query_pairs().find(|(k, _)| k == "path") {
        return Ok(Some(CorpOpenTarget {
            href: v.to_string(),
        }));
    }
    Ok(Some(CorpOpenTarget {
        href: "/drive/".into(),
    }))
}

/// Resolve deep-link target against config (relative path → absolute).
pub fn resolve_open(cfg: &DesktopConfig, target: &CorpOpenTarget) -> Result<String, CorpError> {
    if target.href.starts_with("http://") || target.href.starts_with("https://") {
        return Ok(target.href.clone());
    }
    workspace_href(cfg, Some(&target.href))
}

/// Scan process args for first deep link / http URL.
pub fn open_target_from_args<I, S>(args: I) -> Result<Option<CorpOpenTarget>, CorpError>
where
    I: IntoIterator<Item = S>,
    S: AsRef<str>,
{
    for a in args {
        if let Some(t) = parse_open_url(a.as_ref())? {
            return Ok(Some(t));
        }
    }
    Ok(None)
}

/// First existing filesystem path from argv (skip flags / deep links).
pub fn file_path_from_args<I, S>(args: I) -> Option<String>
where
    I: IntoIterator<Item = S>,
    S: AsRef<str>,
{
    let mut skip_next = false;
    for (i, a) in args.into_iter().enumerate() {
        let a = a.as_ref();
        if i == 0 {
            // exe path
            continue;
        }
        if skip_next {
            skip_next = false;
            continue;
        }
        if a == "--sku" {
            skip_next = true;
            continue;
        }
        if a.starts_with("--sku=") || a.starts_with('-') {
            continue;
        }
        if a.starts_with("era-office:") || a.starts_with("http://") || a.starts_with("https://") {
            continue;
        }
        let p = std::path::Path::new(a);
        if p.is_file() {
            return Some(a.to_string());
        }
        // Association may pass path before file exists briefly — accept known extensions
        if crate::sku::Sku::from_path_ext(a).is_some() {
            return Some(a.to_string());
        }
    }
    None
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::config::{DesktopConfig, Profile};

    fn corp_cfg() -> DesktopConfig {
        DesktopConfig {
            profile: Profile::Corporate,
            server_url: Some("https://office.example.gov".into()),
        }
    }

    #[test]
    fn workspace_default_drive() {
        let h = workspace_href(&corp_cfg(), None).unwrap();
        assert_eq!(h, "https://office.example.gov/drive/");
    }

    #[test]
    fn deep_link_path() {
        let t = parse_open_url("era-office://open?path=/docs/abc")
            .unwrap()
            .unwrap();
        let href = resolve_open(&corp_cfg(), &t).unwrap();
        assert_eq!(href, "https://office.example.gov/docs/abc");
    }

    #[test]
    fn deep_link_absolute_url() {
        let t = parse_open_url("era-office://open?url=https%3A%2F%2Foffice.example.gov%2Ftables%2F1")
            .unwrap()
            .unwrap();
        assert_eq!(t.href, "https://office.example.gov/tables/1");
    }

    #[test]
    fn args_scan() {
        let t = open_target_from_args(["era-office-desktop", "era-office://open?path=/drive/"])
            .unwrap()
            .unwrap();
        assert_eq!(t.href, "/drive/");
    }
}
