//! Device Control / USB — monitor stub (ADR-0016 §2).
#![deny(unsafe_code)]

mod hooks;

use anyhow::Result;
use era_plugin_sdk::{emit, EnforcementStatus};
use hooks::{hook_status, parse_usb_event};

fn main() -> Result<()> {
    if let Ok(raw) = std::env::var("ERA_USB_EVENT") {
        if let Some(ev) = parse_usb_event(&raw) {
            emit(&ev)?;
        }
    }

    let mut status = EnforcementStatus::ready("devicecontrol", hook_status());
    if let Ok(class) = std::env::var("ERA_SIM_USB_CLASS") {
        status.detail = format!("device_class={class}");
        if class.eq_ignore_ascii_case("usb_storage") {
            status.detail.push_str(" would_block=true");
        }
    }
    emit(&status)?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use era_plugin_sdk::EnforcementStatus;
    use crate::hooks::{hook_status, parse_usb_event};

    #[test]
    fn usb_storage_simulation() {
        std::env::remove_var("ERA_ENFORCE_LIVE");
        let s = EnforcementStatus::ready("devicecontrol", hook_status());
        assert_eq!(s.plugin, "devicecontrol");
        assert_eq!(s.mode, "monitor");
    }

    #[test]
    fn usb_event_from_env_format() {
        let ev = parse_usb_event("detach:usb_storage:disk-7").unwrap();
        assert_eq!(ev.action, "detach");
    }
}
