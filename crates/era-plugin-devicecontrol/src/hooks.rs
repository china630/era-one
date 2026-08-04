//! Shared helpers for devicecontrol plugin.
#![deny(unsafe_code)]

use serde::Serialize;

/// Hook status: user-land on Linux when ERA_ENFORCE_LIVE=1, else simulated.
pub fn hook_status() -> &'static str {
    if std::env::var("ERA_ENFORCE_LIVE")
        .map(|v| v == "1")
        .unwrap_or(false)
    {
        if cfg!(target_os = "linux") {
            "user_land"
        } else {
            "simulated"
        }
    } else {
        "simulated"
    }
}

pub fn kernel_hook() -> &'static str {
    "unavailable"
}

pub fn enforce_mode() -> &'static str {
    match std::env::var("ERA_ENFORCE_MODE").ok().as_deref() {
        Some("enforce") => "enforce",
        _ => "monitor",
    }
}

pub fn effect() -> &'static str {
    "telemetry_only"
}

/// USB-событие из ERA_USB_EVENT (attach:CLASS:ID или detach:CLASS:ID).
#[derive(Debug, Clone, Serialize, PartialEq)]
pub struct UsbEvent {
    pub domain: String,
    pub kind: String,
    pub action: String,
    pub device_class: String,
    pub device_id: String,
    pub hook: String,
    pub kernel_hook: String,
    pub enforce_mode: String,
    pub effect: String,
    pub would_block: bool,
    pub blocked: bool,
}

pub fn parse_usb_event(raw: &str) -> Option<UsbEvent> {
    let parts: Vec<&str> = raw.splitn(3, ':').collect();
    if parts.len() != 3 {
        return None;
    }
    let action = parts[0].trim().to_lowercase();
    if action != "attach" && action != "detach" {
        return None;
    }
    let device_class = parts[1].trim().to_string();
    let would_block = device_class.eq_ignore_ascii_case("usb_storage");
    let mode = enforce_mode();
    let blocked = would_block && mode == "enforce";
    Some(UsbEvent {
        domain: "devicecontrol".into(),
        kind: "usb_event".into(),
        action,
        device_class,
        device_id: parts[2].trim().to_string(),
        hook: hook_status().into(),
        kernel_hook: kernel_hook().into(),
        enforce_mode: mode.into(),
        effect: effect().into(),
        would_block,
        blocked,
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;
    use std::path::PathBuf;

    #[test]
    fn default_hook_is_simulated() {
        std::env::remove_var("ERA_ENFORCE_LIVE");
        assert_eq!(hook_status(), "simulated");
        assert_eq!(kernel_hook(), "unavailable");
        assert_eq!(effect(), "telemetry_only");
    }

    #[test]
    fn parse_usb_attach() {
        std::env::remove_var("ERA_ENFORCE_LIVE");
        std::env::remove_var("ERA_ENFORCE_MODE");
        let ev = parse_usb_event("attach:usb_storage:/dev/sdb1").unwrap();
        assert_eq!(ev.action, "attach");
        assert_eq!(ev.device_class, "usb_storage");
        assert_eq!(ev.device_id, "/dev/sdb1");
        assert!(ev.would_block);
        assert!(!ev.blocked);
        assert_eq!(ev.effect, "telemetry_only");
        assert_eq!(ev.hook, "simulated");
    }

    #[test]
    fn parse_usb_invalid() {
        assert!(parse_usb_event("bad").is_none());
    }

    #[test]
    fn golden_usb_attach() {
        std::env::remove_var("ERA_ENFORCE_LIVE");
        std::env::remove_var("ERA_ENFORCE_MODE");
        let ev = parse_usb_event("attach:usb_storage:USB\\VID_1234").unwrap();
        let got = serde_json::to_string(&ev).unwrap();
        let path = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
            .join("testdata")
            .join("usb_attach.golden.json");
        let want = fs::read_to_string(path).unwrap().trim().to_string();
        assert_eq!(got, want);
    }
}
