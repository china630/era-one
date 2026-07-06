//! Tamper protection: OS-level watchdog + optional sim (S6-5).

mod watchdog;

use crate::builder;
use crate::capture::is_production_mode;
use crate::config::Config;
use crate::Envelope;
use std::sync::atomic::{AtomicBool, Ordering};

static TAMPER_EMITTED: AtomicBool = AtomicBool::new(false);

/// Проверяет tamper: OS watchdog, затем ERA_TAMPER_SIM (игнорируется в ERA_PRODUCTION=1).
pub fn poll_tamper(cfg: &Config) -> Option<Envelope> {
    if let Some(reason) = watchdog::check(cfg) {
        return emit_once(cfg, &reason);
    }
    if std::env::var("ERA_TAMPER_SIM").is_err() {
        return None;
    }
    if is_production_mode() {
        tracing::warn!("ERA_PRODUCTION=1: игнорируем ERA_TAMPER_SIM");
        return None;
    }
    emit_once(cfg, "simulated tamper (ERA_TAMPER_SIM)")
}

fn emit_once(cfg: &Config, reason: &str) -> Option<Envelope> {
    if TAMPER_EMITTED.swap(true, Ordering::SeqCst) {
        return None;
    }
    tracing::warn!("tamper protection: {reason} — self-heal active");
    Some(builder::tamper_alert_envelope(cfg))
}

#[cfg(test)]
pub fn reset_for_test() {
    TAMPER_EMITTED.store(false, Ordering::SeqCst);
    watchdog::reset_for_test();
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::config::Config;
    use std::sync::Mutex;

    static TAMPER_TEST_LOCK: Mutex<()> = Mutex::new(());

    #[test]
    fn tamper_sim_ignored_in_production() {
        let _g = TAMPER_TEST_LOCK.lock().unwrap();
        reset_for_test();
        std::env::set_var("ERA_PRODUCTION", "1");
        std::env::set_var("ERA_TAMPER_SIM", "1");
        let cfg = Config::dev_defaults();
        assert!(poll_tamper(&cfg).is_none());
        std::env::remove_var("ERA_PRODUCTION");
        std::env::remove_var("ERA_TAMPER_SIM");
        reset_for_test();
    }

    #[test]
    fn tamper_sim_emits_once() {
        let _g = TAMPER_TEST_LOCK.lock().unwrap();
        reset_for_test();
        std::env::set_var("ERA_TAMPER_SIM", "1");
        let cfg = Config::dev_defaults();
        assert!(poll_tamper(&cfg).is_some());
        assert!(poll_tamper(&cfg).is_none());
        std::env::remove_var("ERA_TAMPER_SIM");
        reset_for_test();
    }

    #[test]
    fn lock_removal_emits_alert() {
        let _g = TAMPER_TEST_LOCK.lock().unwrap();
        reset_for_test();
        let dir = std::env::temp_dir().join(format!("era-tamper-mod-{}", std::process::id()));
        let _ = std::fs::create_dir_all(&dir);
        std::env::set_var("ERA_STATE_DIR", dir.to_string_lossy().as_ref());
        let cfg = Config::dev_defaults();
        assert!(crate::tamper::watchdog::check(&cfg).is_none());
        let lock = dir.join(format!("era-agent-{}.lock", cfg.agent_id));
        std::fs::remove_file(&lock).expect("remove lock");
        let env = poll_tamper(&cfg).expect("tamper alert");
        match &env.payload {
            Some(era_proto::era::v1::envelope::Payload::Process(_)) => {}
            other => panic!("expected process payload, got {other:?}"),
        }
        let _ = std::fs::remove_dir_all(&dir);
        std::env::remove_var("ERA_STATE_DIR");
        reset_for_test();
    }
}
