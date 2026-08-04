//! Проверка process Envelope через enforcement (monitor → detection; enforce → local block).

use crate::config::Config;
use crate::envelope;
use crate::enforce::engine::{BlockResult, EnforceEngine, ExecRequest};
use crate::enforce::envelope::enforcement_detection_envelope;
use crate::enforce::user_land::try_user_land_block;
use crate::Envelope;

pub fn check_process_envelope(
    cfg: &Config,
    eng: &EnforceEngine,
    env: &Envelope,
) -> Option<Envelope> {
    let process = match env.payload.as_ref()? {
        envelope::Payload::Process(p) => p,
        _ => return None,
    };
    let req = ExecRequest {
        image_path: process.image_path.clone(),
        command_line: process.command_line.clone(),
        hash_sha256: process.hash_sha256.clone(),
        signer: String::new(),
        parent_path: String::new(),
    };
    let decision = eng.evaluate_exec(&req);
    let br = eng.apply_block(&decision);
    // User-land SIGTERM stub when LIVE + deny gate; no pid → gate semantics only (allowed=false).
    if !br.allowed && br.effect == "user_land_block" && process.pid > 0 {
        let _ = try_user_land_block(process.pid);
    }
    if br.would_block {
        enforcement_detection_envelope(cfg, &decision, &process.image_path)
    } else {
        None
    }
}

/// Evaluate + return lab block result (for plugins / orchestrator).
pub fn evaluate_exec_block(
    eng: &EnforceEngine,
    req: &ExecRequest,
) -> (crate::enforce::Decision, BlockResult) {
    let decision = eng.evaluate_exec(req);
    let br = eng.apply_block(&decision);
    (decision, br)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::builder;
    use crate::enforce::engine::{Decision, ExecRequest};
    use crate::enforce::policy::{EnforcementMode, EnforcementPolicy};
    use std::fs;
    use std::path::PathBuf;

    #[test]
    fn monitor_emits_detection_on_deny_match() {
        let cfg = Config::dev_defaults();
        let policy =
            EnforcementPolicy::parse_json(include_str!("testdata/enforcement_policy.json"))
                .unwrap();
        let eng = EnforceEngine::new(policy);
        let env = builder::process_envelope(
            &cfg,
            "create",
            1,
            0,
            r"C:\Temp\malware.exe",
            "malware",
            "user",
            false,
        );
        let det = check_process_envelope(&cfg, &eng, &env);
        assert!(det.is_some());
        assert!(matches!(
            eng.evaluate_exec(&ExecRequest {
                image_path: r"C:\Temp\malware.exe".into(),
                command_line: String::new(),
                hash_sha256: String::new(),
                signer: String::new(),
                parent_path: String::new(),
            }),
            Decision::Deny { .. }
        ));
    }

    #[test]
    fn virtual_patch_monitor_emits_detection() {
        let cfg = Config::dev_defaults();
        let policy =
            EnforcementPolicy::parse_json(include_str!("testdata/enforcement_policy.json"))
                .unwrap();
        let eng = EnforceEngine::new(policy);
        let req = ExecRequest {
            image_path: r"C:\App\vulnerable.dll".into(),
            command_line: String::new(),
            hash_sha256: String::new(),
            signer: String::new(),
            parent_path: String::new(),
        };
        let decision = eng.evaluate_exec(&req);
        match &decision {
            Decision::Deny { engine, rule_id, .. } => {
                assert_eq!(engine, "era-virtual-patch");
                assert_eq!(rule_id, "vp-cve-2024-0001");
            }
            _ => panic!("expected deny"),
        }
        let (allowed, would_block) = eng.apply_exec(&decision);
        assert!(allowed && would_block, "monitor must allow+would_block");
        let env = builder::process_envelope(
            &cfg,
            "create",
            1,
            0,
            r"C:\App\vulnerable.dll",
            "vuln",
            "user",
            false,
        );
        let det = check_process_envelope(&cfg, &eng, &env);
        assert!(det.is_some(), "monitor must emit detection Envelope");
    }

    #[test]
    fn golden_enforce_block_vs_monitor_would_block() {
        let _g = crate::enforce::user_land::live_env_lock();
        std::env::remove_var("ERA_ENFORCE_LIVE");
        let mut policy =
            EnforcementPolicy::parse_json(include_str!("testdata/enforcement_policy.json"))
                .unwrap();
        let req = ExecRequest {
            image_path: r"C:\Temp\malware.exe".into(),
            command_line: String::new(),
            hash_sha256: String::new(),
            signer: String::new(),
            parent_path: String::new(),
        };
        let mon = EnforceEngine::new(policy.clone());
        let mon_br = mon.apply_block(&mon.evaluate_exec(&req));
        policy.mode = EnforcementMode::Enforce;
        let enf = EnforceEngine::new(policy);
        let enf_br = enf.apply_block(&enf.evaluate_exec(&req));
        let got = serde_json::json!({
            "monitor": {
                "allowed": mon_br.allowed,
                "would_block": mon_br.would_block,
                "blocked": mon_br.blocked,
                "effect": mon_br.effect,
                "enforce_mode": mon_br.enforce_mode,
                "kernel_hook": mon_br.kernel_hook,
            },
            "enforce": {
                "allowed": enf_br.allowed,
                "would_block": enf_br.would_block,
                "blocked": enf_br.blocked,
                "effect": enf_br.effect,
                "enforce_mode": enf_br.enforce_mode,
                "kernel_hook": enf_br.kernel_hook,
            }
        });
        let path = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
            .join("src/enforce/testdata/enforce_vs_monitor.golden.json");
        if std::env::var("UPDATE_GOLDEN").ok().as_deref() == Some("1") {
            fs::write(&path, serde_json::to_string_pretty(&got).unwrap() + "\n").unwrap();
        }
        let want: serde_json::Value =
            serde_json::from_str(&fs::read_to_string(&path).unwrap()).unwrap();
        assert_eq!(got, want);
        assert!(mon_br.allowed && mon_br.would_block && !mon_br.blocked);
        assert_eq!(mon_br.effect, "telemetry_only");
        // enforce without LIVE: blocked decision flag + process still allowed
        assert!(enf_br.allowed && enf_br.blocked && enf_br.would_block);
        assert_eq!(enf_br.effect, "telemetry_only");
    }

    #[test]
    fn golden_enforce_live_user_land_block() {
        let _g = crate::enforce::user_land::live_env_lock();
        std::env::set_var("ERA_ENFORCE_LIVE", "1");
        let mut policy =
            EnforcementPolicy::parse_json(include_str!("testdata/enforcement_policy.json"))
                .unwrap();
        policy.mode = EnforcementMode::Enforce;
        let eng = EnforceEngine::new(policy);
        let req = ExecRequest {
            image_path: r"C:\Temp\malware.exe".into(),
            command_line: String::new(),
            hash_sha256: String::new(),
            signer: String::new(),
            parent_path: String::new(),
        };
        let br = eng.apply_block(&eng.evaluate_exec(&req));
        let got = serde_json::json!({
            "allowed": br.allowed,
            "would_block": br.would_block,
            "blocked": br.blocked,
            "effect": br.effect,
            "enforce_mode": br.enforce_mode,
        });
        let path = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
            .join("src/enforce/testdata/enforce_live_user_land.golden.json");
        if std::env::var("UPDATE_GOLDEN").ok().as_deref() == Some("1") {
            fs::write(&path, serde_json::to_string_pretty(&got).unwrap() + "\n").unwrap();
        }
        let want: serde_json::Value =
            serde_json::from_str(&fs::read_to_string(&path).unwrap()).unwrap();
        assert_eq!(got, want);
        assert!(!br.allowed && br.blocked && br.effect == "user_land_block");
        std::env::remove_var("ERA_ENFORCE_LIVE");
    }
}
