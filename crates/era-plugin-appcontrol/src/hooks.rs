//! Shared helpers for appcontrol plugin.
#![deny(unsafe_code)]

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

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn default_hook_is_simulated() {
        std::env::remove_var("ERA_ENFORCE_LIVE");
        assert_eq!(hook_status(), "simulated");
    }

    #[test]
    fn enforce_live_linux_uses_audit() {
        std::env::set_var("ERA_ENFORCE_LIVE", "1");
        if cfg!(target_os = "linux") {
            assert_eq!(hook_status(), "audit");
        } else {
            assert_eq!(hook_status(), "simulated");
        }
        std::env::remove_var("ERA_ENFORCE_LIVE");
    }
}
