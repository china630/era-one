//! License-gate: запуск плагинов только при разрешённом модуле (ADR-0010).

use crate::config::Config;
use era_license::{verify, Claims};
use std::time::{SystemTime, UNIX_EPOCH};

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum GateDecision {
    Allow,
    DenyNoToken,
    DenyModule,
    DenyInvalid,
    DenyExpired,
}

/// Проверяет, разрешён ли `license_module` для плагина.
pub fn check_module(cfg: &Config, license_module: &str) -> GateDecision {
    let Some(token) = cfg.license_token.as_ref() else {
        // dev/air-gap без токена: разрешаем только core-модули и inventory в dev
        if cfg.dev_allow_plugins {
            return GateDecision::Allow;
        }
        return GateDecision::DenyNoToken;
    };
    let pub_key = match cfg.vendor_public_key_bytes() {
        Ok(k) => k,
        Err(_) => return GateDecision::DenyInvalid,
    };
    match verify(token, &pub_key) {
        Ok(claims) => {
            let now = SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .map(|d| d.as_secs() as i64)
                .unwrap_or(0);
            if !claims.valid_at(now) {
                return GateDecision::DenyExpired;
            }
            if claims.has_module(license_module) || license_module == "core" {
                GateDecision::Allow
            } else {
                GateDecision::DenyModule
            }
        }
        Err(_) => GateDecision::DenyInvalid,
    }
}

/// Хелпер для тестов: claims с модулем manage.
pub fn dev_claims_with_manage() -> Claims {
    Claims {
        v: 1,
        lid: "lic-dev".into(),
        cust: "dev".into(),
        tenant: "tenant-dev".into(),
        edition: "core".into(),
        modules: vec!["manage".into(), "ai".into()],
        max_nodes: 1000,
        deployment: "deploy-dev".into(),
        iat: 0,
        nbf: 0,
        exp: i64::MAX / 2,
        grace_days: 30,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::config::Config;

    #[test]
    fn dev_allow_without_token() {
        let mut cfg = Config::dev_defaults();
        cfg.dev_allow_plugins = true;
        assert_eq!(
            check_module(&cfg, "manage"),
            GateDecision::Allow
        );
    }

    #[test]
    fn deny_without_token_prod_mode() {
        let mut cfg = Config::dev_defaults();
        cfg.dev_allow_plugins = false;
        assert_eq!(
            check_module(&cfg, "manage"),
            GateDecision::DenyNoToken
        );
    }
}
