//! ERA plugin-inventory — полный host snapshot → NDJSON (ADR-0011 §5a).
#![deny(unsafe_code)]

mod hw;
mod software;

use anyhow::Result;
use era_plugin_sdk::{emit, InventoryRecord};
use hw::collect_hw;
use software::collect_software;
use sysinfo::{Disks, System};

fn main() -> Result<()> {
    let mut sys = System::new_all();
    sys.refresh_all();
    let hostname = System::host_name().unwrap_or_else(|| "unknown".into());
    let platform = if cfg!(target_os = "windows") {
        "windows"
    } else if cfg!(target_os = "macos") {
        "macos"
    } else {
        "linux"
    };
    let os_version = System::os_version().unwrap_or_default();
    let cpu_count = sys.cpus().len() as u32;
    let total_memory_mb = sys.total_memory() / (1024 * 1024);
    let hw = collect_hw(&sys);
    let disks = Disks::new_with_refreshed_list();
    let detail_json = serde_json::json!({
        "disk_count": disks.len(),
        "cpu_model": hw.cpu_model,
    });
    let mut rec = InventoryRecord::host_snapshot(
        &hostname,
        platform,
        &os_version,
        cpu_count,
        total_memory_mb,
        &detail_json.to_string(),
    );
    rec.fqdn = hostname.clone();
    rec.os_name = hw.os_name;
    rec.kernel = hw.kernel;
    rec.cpu_model = hw.cpu_model;
    rec.disk_total_gb = hw.disk_total_gb;
    rec.serial_number = hw.serial_number;
    rec.board_serial = hw.board_serial;
    rec.manufacturer = hw.manufacturer;
    rec.model = hw.model;
    rec.mac_addrs = hw.mac_addrs;
    rec.ip_addrs = hw.ip_addrs;
    rec.software = collect_software(platform);
    emit(&rec)?;
    Ok(())
}
