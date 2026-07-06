//! Deploy install helpers — staging copy + rollback snapshot.
#![deny(unsafe_code)]

use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};
use std::fs;
use std::path::{Path, PathBuf};

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct RollbackSnapshot {
    pub package_ref: String,
    pub staged_path: String,
    pub backup_path: String,
}

/// Копирует пакет в staging; при существующем файле — backup для rollback.
pub fn stage_package(package_ref: &str, staging_root: &Path) -> Result<(PathBuf, Option<RollbackSnapshot>)> {
    let src = resolve_package_path(package_ref)?;
    if !src.is_file() {
        anyhow::bail!("package not found: {}", src.display());
    }
    fs::create_dir_all(staging_root).with_context(|| format!("mkdir {}", staging_root.display()))?;
    let name = src
        .file_name()
        .and_then(|n| n.to_str())
        .unwrap_or("package.bin");
    let dest = staging_root.join(name);
    let mut snap = None;
    if dest.exists() {
        let backup = staging_root.join(format!("{name}.rollback.bak"));
        fs::copy(&dest, &backup)?;
        snap = Some(RollbackSnapshot {
            package_ref: package_ref.to_string(),
            staged_path: dest.display().to_string(),
            backup_path: backup.display().to_string(),
        });
    }
    fs::copy(&src, &dest)?;
    Ok((dest, snap))
}

/// Восстанавливает staged файл из rollback snapshot.
pub fn rollback(snapshot: &RollbackSnapshot) -> Result<()> {
    let staged = PathBuf::from(&snapshot.staged_path);
    let backup = PathBuf::from(&snapshot.backup_path);
    if backup.is_file() {
        fs::copy(&backup, &staged)?;
        fs::remove_file(&backup)?;
    } else if staged.is_file() {
        fs::remove_file(&staged)?;
    }
    Ok(())
}

fn resolve_package_path(package_ref: &str) -> Result<PathBuf> {
    let p = package_ref.trim();
    if p.starts_with("file://") {
        return Ok(PathBuf::from(p.trim_start_matches("file://")));
    }
    if p.starts_with("s3://") {
        // lab: локальный mirror через ERA_DEPLOY_LOCAL_MIRROR
        if let Ok(mirror) = std::env::var("ERA_DEPLOY_LOCAL_MIRROR") {
            let key = p.trim_start_matches("s3://");
            return Ok(PathBuf::from(mirror).join(key));
        }
    }
    Ok(PathBuf::from(p))
}

pub fn is_installable_ref(package_ref: &str) -> bool {
    let lower = package_ref.to_lowercase();
    lower.ends_with(".deb") || lower.ends_with(".msi") || lower.ends_with(".rpm")
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Write;

    #[test]
    fn stage_and_rollback_deb() {
        let base = std::env::temp_dir().join(format!("era-deploy-test-{}", std::process::id()));
        let _ = fs::remove_dir_all(&base);
        fs::create_dir_all(&base).unwrap();
        let pkg = base.join("app_1.0.deb");
        {
            let mut f = fs::File::create(&pkg).unwrap();
            writeln!(f, "fake deb v1").unwrap();
        }
        let staging = base.join("staging");
        let (dest, snap1) = stage_package(pkg.to_str().unwrap(), &staging).unwrap();
        assert!(dest.is_file());
        assert!(snap1.is_none());

        {
            let mut f = fs::File::create(&pkg).unwrap();
            writeln!(f, "fake deb v2").unwrap();
        }
        let (_, snap2) = stage_package(pkg.to_str().unwrap(), &staging).unwrap();
        assert!(snap2.is_some());
        rollback(&snap2.unwrap()).unwrap();
        let content = fs::read_to_string(&dest).unwrap();
        assert!(content.contains("v1"));
        let _ = fs::remove_dir_all(&base);
    }

    #[test]
    fn installable_refs() {
        assert!(is_installable_ref("s3://bucket/app.msi"));
        assert!(!is_installable_ref("s3://bucket/app.tar.gz"));
    }
}
