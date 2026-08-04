//! User-land enforce stub (not WHQL kernel).
//! When `ERA_ENFORCE_LIVE=1`, lab gate may set `effect=user_land_block` and optionally SIGTERM.

use std::sync::{Mutex, MutexGuard};

/// Serializes tests that mutate `ERA_ENFORCE_LIVE` (process-global env).
pub fn live_env_lock() -> MutexGuard<'static, ()> {
    static LOCK: Mutex<()> = Mutex::new(());
    LOCK.lock().unwrap_or_else(|e| e.into_inner())
}

/// True when ERA_ENFORCE_LIVE is 1/true (case-insensitive).
pub fn enforce_live_enabled() -> bool {
    match std::env::var("ERA_ENFORCE_LIVE") {
        Ok(v) => {
            let v = v.trim().to_ascii_lowercase();
            v == "1" || v == "true" || v == "yes"
        }
        Err(_) => false,
    }
}

/// Best-effort user-land block: on Linux with pid>0 and ERA_ENFORCE_LIVE=1, send SIGTERM.
/// Returns true if the signal was delivered (kill==0). No-op / false otherwise.
/// Gate semantics (`allowed=false`) are set by `apply_block` even when pid is absent.
pub fn try_user_land_block(pid: u64) -> bool {
    if pid == 0 || !enforce_live_enabled() {
        return false;
    }
    #[cfg(target_os = "linux")]
    {
        // Allowlisted: POSIX kill for lab user-land stub (crate denies unsafe by default).
        send_sigterm(pid)
    }
    #[cfg(not(target_os = "linux"))]
    {
        let _ = pid;
        false
    }
}

#[cfg(target_os = "linux")]
#[allow(unsafe_code)]
fn send_sigterm(pid: u64) -> bool {
    extern "C" {
        fn kill(pid: i32, sig: i32) -> i32;
    }
    const SIGTERM: i32 = 15;
    // SAFETY: kill is a POSIX syscall; pid is caller-supplied process id.
    unsafe { kill(pid as i32, SIGTERM) == 0 }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn live_flag_and_noop_without_pid() {
        let _g = live_env_lock();
        std::env::remove_var("ERA_ENFORCE_LIVE");
        assert!(!enforce_live_enabled());
        assert!(!try_user_land_block(0));
        std::env::set_var("ERA_ENFORCE_LIVE", "1");
        assert!(enforce_live_enabled());
        assert!(!try_user_land_block(0), "pid=0 must not signal");
        std::env::remove_var("ERA_ENFORCE_LIVE");
    }
}
