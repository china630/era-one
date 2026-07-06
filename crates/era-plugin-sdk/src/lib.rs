//! ERA plugin SDK — эмит NDJSON для subprocess ABI (ADR-0019, ADR-0011).
#![deny(unsafe_code)]

use anyhow::Result;
use serde::{Deserialize, Serialize};
use std::io::{self, Write};

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct SoftwareEntry {
    pub name: String,
    pub version: String,
    #[serde(default)]
    pub vendor: String,
    /// registry | dpkg | rpm | brew
    #[serde(default)]
    pub source: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct InventoryRecord {
    pub domain: String,
    pub kind: String,
    pub hostname: String,
    pub platform: String,
    pub os_version: String,
    pub cpu_count: u32,
    pub total_memory_mb: u64,
    #[serde(default)]
    pub detail_json: String,
    #[serde(default)]
    pub fqdn: String,
    #[serde(default)]
    pub os_name: String,
    #[serde(default)]
    pub kernel: String,
    #[serde(default)]
    pub cpu_model: String,
    #[serde(default)]
    pub disk_total_gb: u64,
    #[serde(default)]
    pub serial_number: String,
    #[serde(default)]
    pub board_serial: String,
    #[serde(default)]
    pub manufacturer: String,
    #[serde(default)]
    pub model: String,
    #[serde(default)]
    pub mac_addrs: Vec<String>,
    #[serde(default)]
    pub ip_addrs: Vec<String>,
    #[serde(default)]
    pub software: Vec<SoftwareEntry>,
}

/// Vuln snapshot для cron-плагина era-plugin-vuln (L-05).
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct VulnSnapshot {
    pub domain: String,
    pub kind: String,
    pub hostname: String,
    pub platform: String,
    #[serde(default)]
    pub software: Vec<SoftwareEntry>,
    #[serde(default)]
    pub scanned_at: String,
}

impl VulnSnapshot {
    pub fn host_scan(hostname: &str, platform: &str, software: Vec<SoftwareEntry>) -> Self {
        Self {
            domain: "vuln".into(),
            kind: "host_snapshot".into(),
            hostname: hostname.to_string(),
            platform: platform.to_string(),
            software,
            scanned_at: String::new(),
        }
    }
}

impl InventoryRecord {
    pub fn host_snapshot(
        hostname: &str,
        platform: &str,
        os_version: &str,
        cpu_count: u32,
        total_memory_mb: u64,
        detail_json: &str,
    ) -> Self {
        Self {
            domain: "inventory".into(),
            kind: "host_snapshot".into(),
            hostname: hostname.to_string(),
            platform: platform.to_string(),
            os_version: os_version.to_string(),
            cpu_count,
            total_memory_mb,
            detail_json: detail_json.to_string(),
            os_name: platform.to_string(),
            ..Default::default()
        }
    }
}

/// Статус enforcement-плагина (monitor/simulated hook).
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct EnforcementStatus {
    pub domain: String,
    pub kind: String,
    pub plugin: String,
    pub mode: String,
    #[serde(default)]
    pub hook: String,
    #[serde(default)]
    pub detail: String,
}

impl EnforcementStatus {
    pub fn ready(plugin: &str, hook: &str) -> Self {
        Self {
            domain: "enforcement".into(),
            kind: "plugin_status".into(),
            plugin: plugin.to_string(),
            mode: "monitor".into(),
            hook: hook.to_string(),
            detail: String::new(),
        }
    }
}

impl Default for InventoryRecord {
    fn default() -> Self {
        Self {
            domain: "inventory".into(),
            kind: "host_snapshot".into(),
            hostname: String::new(),
            platform: String::new(),
            os_version: String::new(),
            cpu_count: 0,
            total_memory_mb: 0,
            detail_json: String::new(),
            fqdn: String::new(),
            os_name: String::new(),
            kernel: String::new(),
            cpu_model: String::new(),
            disk_total_gb: 0,
            serial_number: String::new(),
            board_serial: String::new(),
            manufacturer: String::new(),
            model: String::new(),
            mac_addrs: Vec::new(),
            ip_addrs: Vec::new(),
            software: Vec::new(),
        }
    }
}

/// Пишет одну NDJSON-строку в stdout.
pub fn emit(record: &impl Serialize) -> Result<()> {
    let line = serde_json::to_string(record)?;
    let mut out = io::stdout().lock();
    writeln!(out, "{line}")?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn inventory_record_json_stable() {
        let r = InventoryRecord::host_snapshot("h1", "linux", "6.1", 4, 8192, "{}");
        let j = serde_json::to_string(&r).unwrap();
        assert!(j.contains("\"domain\":\"inventory\""));
    }
}
