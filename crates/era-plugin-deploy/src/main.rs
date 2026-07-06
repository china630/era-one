//! On-demand deploy/patch — verify OTA token + staged install (Vision P4).
#![deny(unsafe_code)]

mod install;

use anyhow::Result;
use era_agent_core::ota;
use era_plugin_sdk::emit;
use install::{is_installable_ref, stage_package};
use serde::Serialize;
use std::path::PathBuf;

#[derive(Debug, Serialize)]
struct DeployResult {
    domain: String,
    kind: String,
    package_ref: String,
    status: String,
    verified: bool,
    #[serde(skip_serializing_if = "String::is_empty")]
    detail: String,
}

fn staging_root() -> PathBuf {
    std::env::var("ERA_DEPLOY_STAGING")
        .map(PathBuf::from)
        .unwrap_or_else(|_| PathBuf::from("/var/lib/era-deploy/staging"))
}

fn main() -> Result<()> {
    let package_ref = std::env::var("ERA_DEPLOY_PACKAGE_REF")
        .unwrap_or_else(|_| "s3://era-packages/demo/app.msi".into());
    let mut verified = false;
    let mut detail = String::new();

    if let Ok(mirror) = std::env::var("ERA_OTA_MIRROR") {
        let mirror_dir = PathBuf::from(mirror);
        if let Ok(pub_b64) = std::env::var("ERA_VENDOR_PUB") {
            use base64::engine::general_purpose::STANDARD;
            use base64::Engine;
            if let Ok(bytes) = STANDARD.decode(pub_b64.trim()) {
                if let Ok(arr) = <[u8; 32]>::try_from(bytes.as_slice()) {
                    match ota::pull_from_mirror(&mirror_dir, &arr) {
                        Ok(claims) => {
                            verified = true;
                            detail = format!("verified {} v{}", claims.name, claims.version);
                        }
                        Err(e) => detail = format!("verify skipped: {e:#}"),
                    }
                }
            }
        }
    }

    let status = if std::env::var("ERA_DEPLOY_SIMULATE_FAIL")
        .map(|v| v == "1")
        .unwrap_or(false)
    {
        "failed"
    } else if is_installable_ref(&package_ref) {
        match stage_package(&package_ref, &staging_root()) {
            Ok((dest, snap)) => {
                if let Some(s) = snap {
                    detail.push_str(&format!(" rollback={}", s.backup_path));
                }
                detail.push_str(&format!(" staged={}", dest.display()));
                "installed"
            }
            Err(e) => {
                detail = format!("install error: {e:#}");
                "failed"
            }
        }
    } else {
        "simulated_ok"
    };

    emit(&DeployResult {
        domain: "deploy".into(),
        kind: "install_result".into(),
        package_ref,
        status: status.into(),
        verified,
        detail,
    })?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::install::is_installable_ref;

    #[test]
    fn installable_msi_deb() {
        assert!(is_installable_ref("pkg.msi"));
        assert!(is_installable_ref("pkg.deb"));
    }
}
