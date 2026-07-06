//! ERA plugin-vuln — cron snapshot установленного ПО для server-side CVE match (L-05).
#![deny(unsafe_code)]

use anyhow::Result;
use era_plugin_sdk::{emit, SoftwareEntry, VulnSnapshot};
use sysinfo::System;

fn main() -> Result<()> {
    let hostname = System::host_name().unwrap_or_else(|| "unknown".into());
    let platform = if cfg!(target_os = "windows") {
        "windows"
    } else if cfg!(target_os = "macos") {
        "macos"
    } else {
        "linux"
    };
    let software = collect_software(platform);
    let rec = VulnSnapshot::host_scan(&hostname, platform, software);
    emit(&rec)?;
    Ok(())
}

fn collect_software(platform: &str) -> Vec<SoftwareEntry> {
    let os_version = System::os_version().unwrap_or_default();
    vec![SoftwareEntry {
        name: format!("{platform}-os"),
        version: os_version,
        vendor: "era".into(),
        source: "snapshot".into(),
    }]
}

#[cfg(test)]
mod tests {
    use super::collect_software;

    #[test]
    fn vuln_snapshot_collects_os_entry() {
        let sw = collect_software("linux");
        assert_eq!(sw.len(), 1);
        assert!(sw[0].name.contains("linux"));
    }
}
