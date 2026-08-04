//! Kernel hook interface stub (lab). Full minifilter/WHQL is out of Phase 2 claim.

/// Status of the OS kernel enforcement hook.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum KernelHookStatus {
    /// No signed driver / WHQL path — user-land only.
    Unavailable,
    /// Lab user-land hooks (auditd / WDAC query) when ERA_ENFORCE_LIVE=1 on Linux.
    UserLand,
}

impl KernelHookStatus {
    pub fn as_str(self) -> &'static str {
        match self {
            Self::Unavailable => "unavailable",
            Self::UserLand => "user_land",
        }
    }

    pub fn whql_message(self) -> &'static str {
        match self {
            Self::Unavailable => {
                "kernel_hook=unavailable — WHQL-signed minifilter not loaded; see docs/ERA-Manage-WHQL-Program.md"
            }
            Self::UserLand => "kernel_hook=user_land — lab enforce without WHQL driver",
        }
    }
}

/// Probe kernel hook availability (always unavailable until WHQL driver ships).
pub fn probe_kernel_hook() -> KernelHookStatus {
    // Unsigned driver scaffold is intentionally not loaded. Lab may use user-land only.
    if std::env::var("ERA_ENFORCE_LIVE").map(|v| v == "1").unwrap_or(false) && cfg!(target_os = "linux")
    {
        return KernelHookStatus::UserLand;
    }
    KernelHookStatus::Unavailable
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn default_kernel_unavailable() {
        let _g = crate::enforce::user_land::live_env_lock();
        std::env::remove_var("ERA_ENFORCE_LIVE");
        let s = probe_kernel_hook();
        assert_eq!(s, KernelHookStatus::Unavailable);
        assert!(s.whql_message().contains("WHQL"));
    }
}
