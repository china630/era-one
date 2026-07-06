//! Shared helpers for devicecontrol plugin.
#![deny(unsafe_code)]

use serde::Serialize;

/// Hook status: audit on Linux when ERA_ENFORCE_LIVE=1, else simulated.
pub fn hook_status() -> &'static str {
    if std::env::var("ERA_ENFORCE_LIVE")
        .map(|v| v == "1")
        .unwrap_or(false)
    {
        if cfg!(target_os = "linux") {
            "audit"
        } else {
            "simulated"
        }
    } else {
        "simulated"
    }
}

/// USB-событие из ERA_USB_EVENT (attach:CLASS:ID или detach:CLASS:ID).
#[derive(Debug, Clone, Serialize, PartialEq)]
pub struct UsbEvent {
    pub domain: String,
    pub kind: String,
    pub action: String,
    pub device_class: String,
    pub device_id: String,
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
    Some(UsbEvent {
        domain: "devicecontrol".into(),
        kind: "usb_event".into(),
        action,
        device_class: parts[1].trim().to_string(),
        device_id: parts[2].trim().to_string(),
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn default_hook_is_simulated() {
        std::env::remove_var("ERA_ENFORCE_LIVE");
        assert_eq!(hook_status(), "simulated");
    }

    #[test]
    fn parse_usb_attach() {
        let ev = parse_usb_event("attach:usb_storage:/dev/sdb1").unwrap();
        assert_eq!(ev.action, "attach");
        assert_eq!(ev.device_class, "usb_storage");
        assert_eq!(ev.device_id, "/dev/sdb1");
    }

    #[test]
    fn parse_usb_invalid() {
        assert!(parse_usb_event("bad").is_none());
    }
}
