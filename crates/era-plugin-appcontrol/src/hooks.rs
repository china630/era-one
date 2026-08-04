//! Shared helpers for appcontrol plugin.
#![deny(unsafe_code)]

/// Hook status: user-land on Linux when ERA_ENFORCE_LIVE=1; else simulated.
/// Kernel minifilter remains unavailable until WHQL (see ERA-Manage-WHQL-Program).
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

/// Lab effect — always telemetry_only until WHQL OS path.
pub fn effect() -> &'static str {
    "telemetry_only"
}

/// Decision flag for enforce mode (not OS kill).
pub fn local_block(would_block: bool) -> bool {
    would_block && enforce_mode() == "enforce"
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn default_hook_is_simulated() {
        std::env::remove_var("ERA_ENFORCE_LIVE");
        assert_eq!(hook_status(), "simulated");
        assert_eq!(kernel_hook(), "unavailable");
        assert_eq!(effect(), "telemetry_only");
    }

    #[test]
    fn local_block_only_in_enforce() {
        std::env::set_var("ERA_ENFORCE_MODE", "monitor");
        assert!(!local_block(true));
        std::env::set_var("ERA_ENFORCE_MODE", "enforce");
        assert!(local_block(true));
        std::env::remove_var("ERA_ENFORCE_MODE");
    }
}
