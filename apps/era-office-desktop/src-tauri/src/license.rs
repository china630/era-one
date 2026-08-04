//! Offline device license for Solo Documents / Tables (ADR-0010 via `era-license`).
//! Lab keypair seed is fixed for local/dev only.

use ed25519_dalek::SigningKey;
use era_license::verify;
use serde_json::json;

/// Demo save gate (Documents): block count proxy for “pages”.
pub const DEMO_BLOCK_CAP: usize = 5;

/// Demo save gate (Tables): nonempty cell count.
pub const DEMO_CELL_CAP: usize = 25;

/// Demo save gate (Presentations): slide count.
pub const DEMO_SLIDE_CAP: usize = 5;

/// Demo save gate (Projects): task count.
pub const DEMO_TASK_CAP: usize = 15;

pub const MODULE_SOLO: &str = "office-docs-solo";
pub const MODULE_TABLES_SOLO: &str = "office-tables-solo";
pub const MODULE_PRES_SOLO: &str = "office-pres-solo";
pub const MODULE_PROJECTS_SOLO: &str = "office-projects-solo";

/// Lab signing seed (NOT production). Verifying key is derived at runtime.
pub const LAB_SIGNING_SEED: [u8; 32] = [7u8; 32];

#[derive(Debug, Clone, Copy, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum LicenseMode {
    Demo,
    Licensed,
}

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct LicenseStatus {
    pub mode: LicenseMode,
    pub module: String,
    pub demo_block_cap: usize,
    pub current_blocks: usize,
    pub can_save: bool,
    pub message: String,
}

fn now_unix() -> i64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_secs() as i64)
        .unwrap_or(0)
}

pub fn lab_public_key() -> [u8; 32] {
    SigningKey::from_bytes(&LAB_SIGNING_SEED)
        .verifying_key()
        .to_bytes()
}

fn status_for(
    token: Option<&str>,
    module: &str,
    current: usize,
    cap: usize,
    unit: &str,
) -> LicenseStatus {
    let demo = LicenseStatus {
        mode: LicenseMode::Demo,
        module: module.into(),
        demo_block_cap: cap,
        current_blocks: current,
        can_save: current <= cap,
        message: if current <= cap {
            format!("Demo — save allowed up to {cap} {unit}")
        } else {
            format!("Demo limit exceeded ({current}/{cap} {unit}) — set ERA_SOLO_LICENSE")
        },
    };

    let Some(tok) = token.filter(|t| !t.is_empty()) else {
        return demo;
    };

    let claims = match verify(tok, &lab_public_key()) {
        Ok(c) => c,
        Err(_) => {
            return LicenseStatus {
                message: "Invalid license token — running in demo".into(),
                ..demo
            };
        }
    };

    if !claims.valid_at(now_unix()) {
        return LicenseStatus {
            message: "License expired — running in demo".into(),
            ..demo
        };
    }
    if !claims.has_module(module) {
        return LicenseStatus {
            message: format!("License missing module {module} — demo"),
            ..demo
        };
    }

    LicenseStatus {
        mode: LicenseMode::Licensed,
        module: module.into(),
        demo_block_cap: cap,
        current_blocks: current,
        can_save: true,
        message: format!("Licensed ({})", claims.lid),
    }
}

/// Resolve Solo Documents license from optional ERA1 token.
pub fn status(token: Option<&str>, current_blocks: usize) -> LicenseStatus {
    status_for(token, MODULE_SOLO, current_blocks, DEMO_BLOCK_CAP, "blocks")
}

/// Resolve Solo Tables license from optional ERA1 token.
pub fn status_tables(token: Option<&str>, current_cells: usize) -> LicenseStatus {
    status_for(
        token,
        MODULE_TABLES_SOLO,
        current_cells,
        DEMO_CELL_CAP,
        "cells",
    )
}

/// Resolve Solo Presentations license from optional ERA1 token.
pub fn status_pres(token: Option<&str>, slide_count: usize) -> LicenseStatus {
    status_for(
        token,
        MODULE_PRES_SOLO,
        slide_count,
        DEMO_SLIDE_CAP,
        "slides",
    )
}

/// Resolve Solo Projects license from optional ERA1 token.
pub fn status_projects(token: Option<&str>, task_count: usize) -> LicenseStatus {
    status_for(
        token,
        MODULE_PROJECTS_SOLO,
        task_count,
        DEMO_TASK_CAP,
        "tasks",
    )
}

/// Mint a lab token signed with [`LAB_SIGNING_SEED`].
pub fn mint_test_token(modules: &[&str]) -> String {
    use base64::engine::general_purpose::URL_SAFE_NO_PAD;
    use base64::Engine;
    use ed25519_dalek::Signer;

    let sk = SigningKey::from_bytes(&LAB_SIGNING_SEED);
    let now = now_unix();
    let claims = json!({
        "v": 1,
        "lid": "solo-lab-1",
        "cust": "lab",
        "tenant": "solo",
        "edition": "office-solo",
        "modules": modules,
        "max_nodes": 1,
        "deployment": "solo",
        "iat": now,
        "nbf": now - 60,
        "exp": now + 86_400 * 365,
        "grace_days": 7
    });
    let payload = serde_json::to_vec(&claims).expect("claims json");
    let sig = sk.sign(&payload);
    format!(
        "ERA1.{}.{}",
        URL_SAFE_NO_PAD.encode(&payload),
        URL_SAFE_NO_PAD.encode(sig.to_bytes())
    )
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn no_token_is_demo() {
        let st = status(None, 3);
        assert_eq!(st.mode, LicenseMode::Demo);
        assert!(st.can_save);
    }

    #[test]
    fn demo_over_cap_cannot_save() {
        let st = status(None, DEMO_BLOCK_CAP + 1);
        assert!(!st.can_save);
    }

    #[test]
    fn valid_token_licenses() {
        let tok = mint_test_token(&[MODULE_SOLO]);
        let st = status(Some(&tok), DEMO_BLOCK_CAP + 10);
        assert_eq!(st.mode, LicenseMode::Licensed);
        assert!(st.can_save);
    }

    #[test]
    fn wrong_module_stays_demo() {
        let tok = mint_test_token(&["office-tables"]);
        let st = status(Some(&tok), 1);
        assert_eq!(st.mode, LicenseMode::Demo);
    }

    #[test]
    fn tables_module_licenses() {
        let tok = mint_test_token(&[MODULE_TABLES_SOLO]);
        let st = status_tables(Some(&tok), DEMO_CELL_CAP + 5);
        assert_eq!(st.mode, LicenseMode::Licensed);
        assert!(st.can_save);
        let docs = status(Some(&tok), 1);
        assert_eq!(docs.mode, LicenseMode::Demo);
    }
}
