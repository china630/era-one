//! HW/network идентификаторы для ITAM snapshot.

use sysinfo::{Disks, Networks, System};

pub struct HwSnapshot {
    pub cpu_model: String,
    pub disk_total_gb: u64,
    pub serial_number: String,
    pub board_serial: String,
    pub manufacturer: String,
    pub model: String,
    pub mac_addrs: Vec<String>,
    pub ip_addrs: Vec<String>,
    pub kernel: String,
    pub os_name: String,
}

pub fn collect_hw(sys: &System) -> HwSnapshot {
    let disks = Disks::new_with_refreshed_list();
    let disk_total_gb: u64 = disks.iter().map(|d| d.total_space()).sum::<u64>() / (1024 * 1024 * 1024);
    let nets = Networks::new_with_refreshed_list();
    let mut mac_addrs = Vec::new();
    let mut ip_addrs = Vec::new();
    for (name, data) in &nets {
        if name == "lo" || name.starts_with("lo") {
            continue;
        }
        let macs = data.mac_address();
        if !macs.to_string().is_empty() && macs.to_string() != "00:00:00:00:00:00" {
            mac_addrs.push(macs.to_string());
        }
        for ip in data.ip_networks() {
            ip_addrs.push(ip.addr.to_string());
        }
    }
    let (serial_number, board_serial, manufacturer, model) = read_dmi_ids();
    let cpu_model = sys
        .cpus()
        .first()
        .map(|c| c.brand().to_string())
        .unwrap_or_default();
    HwSnapshot {
        cpu_model,
        disk_total_gb,
        serial_number,
        board_serial,
        manufacturer,
        model,
        mac_addrs,
        ip_addrs,
        kernel: System::kernel_version().unwrap_or_default(),
        os_name: System::name().unwrap_or_default(),
    }
}

fn read_dmi_ids() -> (String, String, String, String) {
    let read = |p: &str| {
        std::fs::read_to_string(p)
            .map(|s| s.trim().to_string())
            .unwrap_or_default()
    };
    (
        read("/sys/class/dmi/id/product_serial"),
        read("/sys/class/dmi/id/board_serial"),
        read("/sys/class/dmi/id/sys_vendor"),
        read("/sys/class/dmi/id/product_name"),
    )
}
