//! Запуск subprocess плагина и чтение NDJSON stdout.

use crate::config::Config;
use crate::plugin::PluginManifest;
use anyhow::{Context, Result};
use std::path::PathBuf;
use std::process::Command;
use tracing::debug;

pub struct PluginRunner {
    plugin_dir: PathBuf,
}

impl PluginRunner {
    pub fn new(cfg: &Config) -> Self {
        Self {
            plugin_dir: cfg.plugin_dir.clone(),
        }
    }

    pub fn resolve_binary(&self, manifest: &PluginManifest) -> PathBuf {
        let p = PathBuf::from(&manifest.binary);
        if p.is_absolute() {
            return p;
        }
        self.plugin_dir.join(&manifest.binary)
    }

    /// Выполняет плагин синхронно; возвращает строки stdout.
    pub fn exec(&self, manifest: &PluginManifest) -> Result<Vec<String>> {
        let bin = self.resolve_binary(manifest);
        debug!(path = %bin.display(), "plugin exec");
        let output = Command::new(&bin)
            .env("ERA_PLUGIN_NAME", &manifest.name)
            .output()
            .with_context(|| format!("exec {}", bin.display()))?;
        if !output.status.success() {
            let stderr = String::from_utf8_lossy(&output.stderr);
            anyhow::bail!(
                "plugin {} exit {}: {}",
                manifest.name,
                output.status,
                stderr.trim()
            );
        }
        let stdout = String::from_utf8_lossy(&output.stdout);
        Ok(stdout
            .lines()
            .map(str::trim)
            .filter(|l| !l.is_empty())
            .map(str::to_string)
            .collect())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::plugin::{PluginManifest, PluginMode};
    use std::env;

    #[test]
    fn exec_inventory_if_present() {
        let bin = env::var("CARGO_BIN_EXE_era-plugin-inventory").ok();
        if bin.is_none() {
            return;
        }
        let mut cfg = Config::dev_defaults();
        cfg.plugin_dir = PathBuf::from(".");
        let mut m = PluginManifest::inventory_dev();
        m.binary = bin.unwrap();
        let runner = PluginRunner::new(&cfg);
        let lines = runner.exec(&m).expect("inventory plugin");
        assert!(!lines.is_empty());
    }
}
