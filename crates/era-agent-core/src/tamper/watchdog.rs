//! OS-level tamper watchdog: file lock + periodic self-check (S6-5).

use crate::config::Config;
use std::fs::{self, OpenOptions};
use std::io::Write;
use std::path::PathBuf;
use std::sync::atomic::{AtomicBool, AtomicU64, Ordering};
use std::time::{SystemTime, UNIX_EPOCH};

static LAST_CHECK: AtomicU64 = AtomicU64::new(0);
static LOCK_HELD: AtomicBool = AtomicBool::new(false);

/// check возвращает причину tamper или None.
pub fn check(cfg: &Config) -> Option<String> {
    if std::env::var("ERA_TAMPER_DISABLE").is_ok() {
        return None;
    }
    let lock_path = lock_file_path(cfg);
    if LOCK_HELD.load(Ordering::Relaxed) && !lock_path.exists() {
        return Some("service stop detected: tamper lock file removed".into());
    }
    if let Some(reason) = verify_lock(&lock_path) {
        return Some(reason);
    }
    if let Err(e) = acquire_lock(&lock_path) {
        return Some(format!("lock unavailable: {e}"));
    }
    LOCK_HELD.store(true, Ordering::Relaxed);
    periodic_self_check()
}

fn lock_file_path(cfg: &Config) -> PathBuf {
    if let Ok(p) = std::env::var("ERA_TAMPER_LOCK") {
        return PathBuf::from(p);
    }
    let base = std::env::var("ERA_STATE_DIR").unwrap_or_else(|_| ".".into());
    PathBuf::from(base).join(format!("era-agent-{}.lock", cfg.agent_id))
}

fn acquire_lock(path: &PathBuf) -> std::io::Result<()> {
    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent)?;
    }
    let mut f = OpenOptions::new()
        .create(true)
        .write(true)
        .truncate(false)
        .open(path)?;
    writeln!(f, "pid={}", std::process::id())?;
    Ok(())
}

fn verify_lock(path: &PathBuf) -> Option<String> {
    if !path.exists() {
        return None;
    }
    let data = fs::read_to_string(path).ok()?;
    if data.contains("pid=") {
        return None;
    }
    Some("tamper lock file corrupted".into())
}

fn periodic_self_check() -> Option<String> {
    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_secs())
        .unwrap_or(0);
    let last = LAST_CHECK.load(Ordering::Relaxed);
    if last != 0 && now.saturating_sub(last) > 3600 {
        if std::env::var("ERA_TAMPER_INTEGRITY_FAIL").is_ok() {
            return Some("binary integrity check failed".into());
        }
    }
    LAST_CHECK.store(now, Ordering::Relaxed);
    None
}

#[cfg(test)]
pub fn reset_for_test() {
    LAST_CHECK.store(0, Ordering::Relaxed);
    LOCK_HELD.store(false, Ordering::Relaxed);
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::config::Config;

    #[test]
    fn lock_file_created() {
        reset_for_test();
        let dir = std::env::temp_dir().join(format!("era-tamper-{}", std::process::id()));
        let _ = fs::create_dir_all(&dir);
        std::env::set_var("ERA_STATE_DIR", dir.to_string_lossy().as_ref());
        let cfg = Config::dev_defaults();
        assert!(check(&cfg).is_none());
        let lock = lock_file_path(&cfg);
        assert!(lock.exists());
        let _ = fs::remove_dir_all(&dir);
        std::env::remove_var("ERA_STATE_DIR");
    }

    #[test]
    fn lock_removal_triggers_tamper() {
        reset_for_test();
        let dir = std::env::temp_dir().join(format!("era-tamper-rm-{}", std::process::id()));
        let _ = fs::create_dir_all(&dir);
        std::env::set_var("ERA_STATE_DIR", dir.to_string_lossy().as_ref());
        let cfg = Config::dev_defaults();
        assert!(check(&cfg).is_none());
        let lock = lock_file_path(&cfg);
        fs::remove_file(&lock).expect("remove lock");
        let reason = check(&cfg).expect("tamper on lock removal");
        assert!(reason.contains("lock file removed"));
        let _ = fs::remove_dir_all(&dir);
        std::env::remove_var("ERA_STATE_DIR");
        reset_for_test();
    }
}
