//! Solo Store SKU mode — one binary, many listings.
//! Targets: Browser ❌ · Solo ✅ · Corporate ❌ (SKU only affects Solo entry)

use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, Default)]
#[serde(rename_all = "snake_case")]
pub enum Sku {
    /// Full hub with all four products.
    #[default]
    Suite,
    Docs,
    Tables,
    Presentations,
    Projects,
}

impl Sku {
    pub fn parse(s: &str) -> Option<Self> {
        match s.trim().to_ascii_lowercase().as_str() {
            "suite" | "office" | "all" | "" => Some(Self::Suite),
            "docs" | "documents" | "doc" => Some(Self::Docs),
            "tables" | "table" | "sheets" => Some(Self::Tables),
            "presentations" | "pres" | "slides" => Some(Self::Presentations),
            "projects" | "project" | "board" => Some(Self::Projects),
            _ => None,
        }
    }

    pub fn as_str(self) -> &'static str {
        match self {
            Self::Suite => "suite",
            Self::Docs => "docs",
            Self::Tables => "tables",
            Self::Presentations => "presentations",
            Self::Projects => "projects",
        }
    }

    pub fn window_title(self) -> &'static str {
        match self {
            Self::Suite => "ERA Office · Solo",
            Self::Docs => "ERA Documents · Solo",
            Self::Tables => "ERA Tables · Solo",
            Self::Presentations => "ERA Presentations · Solo",
            Self::Projects => "ERA Projects · Solo",
        }
    }

    /// Path under bridge origin (no host), e.g. `/docs/solo`.
    pub fn bridge_path(self) -> &'static str {
        match self {
            Self::Suite => "/",
            Self::Docs => "/docs/solo",
            Self::Tables => "/tables/solo",
            Self::Presentations => "/presentations/solo",
            Self::Projects => "/projects/solo",
        }
    }

    pub fn from_env_and_args() -> Self {
        if let Ok(v) = std::env::var("ERA_OFFICE_SKU") {
            if let Some(s) = Self::parse(&v) {
                return s;
            }
        }
        for a in std::env::args().skip(1) {
            if let Some(rest) = a.strip_prefix("--sku=") {
                if let Some(s) = Self::parse(rest) {
                    return s;
                }
            }
            if a == "--sku" {
                continue;
            }
            // `--sku docs` form: previous was --sku handled below
        }
        let args: Vec<String> = std::env::args().skip(1).collect();
        for i in 0..args.len() {
            if args[i] == "--sku" {
                if let Some(v) = args.get(i + 1) {
                    if let Some(s) = Self::parse(v) {
                        return s;
                    }
                }
            }
        }
        Self::Suite
    }

    /// Infer SKU from a filesystem path extension.
    pub fn from_path_ext(path: &str) -> Option<Self> {
        let ext = std::path::Path::new(path)
            .extension()
            .and_then(|e| e.to_str())
            .unwrap_or("")
            .to_ascii_lowercase();
        match ext.as_str() {
            "erad" | "docx" => Some(Self::Docs),
            "erat" | "xlsx" | "ods" => Some(Self::Tables),
            "erap" | "pptx" | "odp" => Some(Self::Presentations),
            "eraj" => Some(Self::Projects),
            _ => None,
        }
    }

    /// Infer from deep-link path like `/tables/…` or `/docs/…`.
    pub fn from_open_path(path: &str) -> Option<Self> {
        let p = path.trim();
        if p.starts_with("/docs") {
            Some(Self::Docs)
        } else if p.starts_with("/tables") {
            Some(Self::Tables)
        } else if p.starts_with("/presentations") {
            Some(Self::Presentations)
        } else if p.starts_with("/projects") {
            Some(Self::Projects)
        } else if p.starts_with("/drive") || p == "/" || p.is_empty() {
            Some(Self::Suite)
        } else {
            None
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parse_aliases() {
        assert_eq!(Sku::parse("DOCUMENTS"), Some(Sku::Docs));
        assert_eq!(Sku::parse("pres"), Some(Sku::Presentations));
        assert_eq!(Sku::parse("suite"), Some(Sku::Suite));
        assert!(Sku::parse("nope").is_none());
    }

    #[test]
    fn from_ext() {
        assert_eq!(Sku::from_path_ext("a.erad"), Some(Sku::Docs));
        assert_eq!(Sku::from_path_ext("b.ERAT"), Some(Sku::Tables));
        assert_eq!(Sku::from_path_ext("c.erap"), Some(Sku::Presentations));
        assert_eq!(Sku::from_path_ext("d.eraj"), Some(Sku::Projects));
    }

    #[test]
    fn from_open_path() {
        assert_eq!(Sku::from_open_path("/tables/solo"), Some(Sku::Tables));
        assert_eq!(Sku::from_open_path("/docs/x"), Some(Sku::Docs));
    }
}
