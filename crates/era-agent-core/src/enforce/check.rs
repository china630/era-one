//! Проверка process Envelope через enforcement (monitor → detection).

use crate::config::Config;
use crate::envelope;
use crate::enforce::engine::{EnforceEngine, ExecRequest};
use crate::enforce::envelope::enforcement_detection_envelope;
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
    let (_, would_block) = eng.apply_exec(&decision);
    if would_block {
        enforcement_detection_envelope(cfg, &decision, &process.image_path)
    } else {
        None
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::builder;
    use crate::enforce::engine::{Decision, ExecRequest};
    use crate::enforce::policy::EnforcementPolicy;

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
}
