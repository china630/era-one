//! Offline policy matching (ADR-0012 §3).

use crate::enforce::kernel::{probe_kernel_hook, KernelHookStatus};
use crate::enforce::policy::{
    AppRule, EnforcementMode, EnforcementPolicy, FailMode, RuleAction, VirtualPatchRule,
};
use crate::enforce::user_land::enforce_live_enabled;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ExecRequest {
    pub image_path: String,
    pub command_line: String,
    pub hash_sha256: String,
    pub signer: String,
    pub parent_path: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DeviceRequest {
    pub device_class: String,
    pub device_id: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Decision {
    Allow,
    Deny {
        rule_id: String,
        engine: String,
        summary: String,
    },
}

/// Outcome of applying monitor/enforce for a deny (lab decision + telemetry).
#[derive(Debug, Clone, PartialEq, Eq, serde::Serialize)]
pub struct BlockResult {
    /// Process/device may proceed (monitor or VP monitor-only).
    pub allowed: bool,
    /// Policy matched deny (detection should fire).
    pub would_block: bool,
    /// Decision flag for enforce mode (telemetry; not OS kill unless LIVE).
    pub blocked: bool,
    /// `telemetry_only` by default; `user_land_block` when ERA_ENFORCE_LIVE=1 + enforce deny (non-VP).
    /// Kernel WHQL remains separate (⏸).
    pub effect: String,
    pub enforce_mode: String,
    pub kernel_hook: String,
    pub hook_message: String,
}

pub struct EnforceEngine {
    policy: EnforcementPolicy,
}

impl EnforceEngine {
    pub fn new(policy: EnforcementPolicy) -> Self {
        Self { policy }
    }

    pub fn policy(&self) -> &EnforcementPolicy {
        &self.policy
    }

    pub fn mode(&self) -> EnforcementMode {
        self.policy.mode
    }

    pub fn evaluate_exec(&self, req: &ExecRequest) -> Decision {
        for rule in &self.policy.virtual_patches {
            if rule.action == RuleAction::Deny && rule_matches_exec(rule, req) {
                return Decision::Deny {
                    rule_id: rule.id.clone(),
                    engine: "era-virtual-patch".into(),
                    summary: format!("virtual patch {} block {}", rule.cve_id, req.image_path),
                };
            }
        }
        for rule in &self.policy.app_rules {
            if rule.action == RuleAction::Deny && app_rule_matches(rule, req) {
                return Decision::Deny {
                    rule_id: rule.id.clone(),
                    engine: "era-appcontrol".into(),
                    summary: format!("appcontrol deny {}", req.image_path),
                };
            }
        }
        Decision::Allow
    }

    pub fn evaluate_device(&self, req: &DeviceRequest) -> Decision {
        for rule in &self.policy.device_rules {
            if rule.action == RuleAction::Deny
                && rule.device_class.eq_ignore_ascii_case(&req.device_class)
            {
                return Decision::Deny {
                    rule_id: rule.id.clone(),
                    engine: "era-devicecontrol".into(),
                    summary: format!("devicecontrol deny {} {}", req.device_class, req.device_id),
                };
            }
        }
        Decision::Allow
    }

    /// Применяет режим monitor/enforce и fail-open. Возвращает (разрешён_запуск, would_block).
    pub fn apply_exec(&self, decision: &Decision) -> (bool, bool) {
        let br = self.apply_block(decision);
        (br.allowed, br.would_block)
    }

    /// Lab enforce: decision + detection.
    /// Default: `effect=telemetry_only`, `allowed=true` (honesty — no fake OS kill).
    /// When `ERA_ENFORCE_LIVE=1` + Enforce + non-VP deny: `allowed=false`, `effect=user_land_block`
    /// (user-land gate; kernel WHQL still ⏸).
    pub fn apply_block(&self, decision: &Decision) -> BlockResult {
        let hook = probe_kernel_hook();
        let mode = match self.policy.mode {
            EnforcementMode::Monitor => "monitor",
            EnforcementMode::Enforce => "enforce",
        };
        match decision {
            Decision::Allow => BlockResult {
                allowed: true,
                would_block: false,
                blocked: false,
                effect: "telemetry_only".into(),
                enforce_mode: mode.into(),
                kernel_hook: hook.as_str().into(),
                hook_message: hook.whql_message().into(),
            },
            Decision::Deny { engine, .. } => {
                let is_vp = engine == "era-virtual-patch";
                let enforce_active = self.policy.mode == EnforcementMode::Enforce && !is_vp;
                let live_block = enforce_active && enforce_live_enabled();
                let effect = if live_block {
                    "user_land_block"
                } else {
                    "telemetry_only"
                };
                BlockResult {
                    // LIVE gate: deny process proceed; otherwise telemetry-only honesty.
                    allowed: !live_block,
                    would_block: true,
                    blocked: enforce_active,
                    effect: effect.into(),
                    enforce_mode: mode.into(),
                    kernel_hook: if is_vp {
                        KernelHookStatus::Unavailable.as_str().into()
                    } else {
                        hook.as_str().into()
                    },
                    hook_message: if is_vp {
                        "virtual_patch monitor-only until kernel WHQL".into()
                    } else if live_block {
                        format!("{} (effect=user_land_block)", hook.whql_message())
                    } else {
                        format!("{} (effect=telemetry_only)", hook.whql_message())
                    },
                }
            }
        }
    }

    pub fn on_hook_error(&self) -> bool {
        matches!(self.policy.fail_mode, FailMode::Open)
    }
}

fn app_rule_matches(rule: &AppRule, req: &ExecRequest) -> bool {
    if !rule.path.is_empty() && !glob_match(&rule.path, &req.image_path) {
        return false;
    }
    if !rule.hash_sha256.is_empty()
        && !req.hash_sha256.eq_ignore_ascii_case(&rule.hash_sha256)
    {
        return false;
    }
    if !rule.signer.is_empty() && !req.signer.eq_ignore_ascii_case(&rule.signer) {
        return false;
    }
    if !rule.parent_path.is_empty() && !glob_match(&rule.parent_path, &req.parent_path) {
        return false;
    }
    !rule.path.is_empty()
        || !rule.hash_sha256.is_empty()
        || !rule.signer.is_empty()
        || !rule.parent_path.is_empty()
}

fn rule_matches_exec(rule: &VirtualPatchRule, req: &ExecRequest) -> bool {
    if rule.path.is_empty() {
        return false;
    }
    glob_match(&rule.path, &req.image_path)
}

/// Простой glob: `*` prefix/suffix/infix.
pub fn glob_match(pattern: &str, value: &str) -> bool {
    let p = pattern.to_lowercase();
    let v = value.to_lowercase();
    if p == "*" {
        return true;
    }
    if let Some(suffix) = p.strip_prefix('*') {
        return v.ends_with(suffix);
    }
    if let Some(prefix) = p.strip_suffix('*') {
        return v.starts_with(prefix);
    }
    if p.contains('*') {
        let parts: Vec<&str> = p.split('*').collect();
        let mut pos = 0usize;
        for (i, part) in parts.iter().enumerate() {
            if part.is_empty() {
                continue;
            }
            if let Some(idx) = v[pos..].find(part) {
                pos += idx + part.len();
            } else if i == 0 && !v.starts_with(part) {
                return false;
            } else if i == parts.len() - 1 && !v.ends_with(part) {
                return false;
            } else {
                return false;
            }
        }
        return true;
    }
    p == v
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::enforce::policy::EnforcementPolicy;

    fn sample_policy() -> EnforcementPolicy {
        EnforcementPolicy::parse_json(include_str!("testdata/enforcement_policy.json"))
            .expect("policy")
    }

    #[test]
    fn deny_malware_golden() {
        let eng = EnforceEngine::new(sample_policy());
        let req = ExecRequest {
            image_path: r"C:\Temp\malware.exe".into(),
            command_line: "malware.exe".into(),
            hash_sha256: String::new(),
            signer: String::new(),
            parent_path: String::new(),
        };
        let d = eng.evaluate_exec(&req);
        assert!(matches!(d, Decision::Deny { .. }));
        let (allowed, _) = eng.apply_exec(&d);
        assert!(allowed); // monitor
    }

    #[test]
    fn allow_legit_binary() {
        let eng = EnforceEngine::new(sample_policy());
        let req = ExecRequest {
            image_path: r"C:\Windows\System32\notepad.exe".into(),
            command_line: "notepad".into(),
            hash_sha256: String::new(),
            signer: String::new(),
            parent_path: String::new(),
        };
        assert_eq!(eng.evaluate_exec(&req), Decision::Allow);
    }

    #[test]
    fn virtual_patch_blocks_vector() {
        let eng = EnforceEngine::new(sample_policy());
        let req = ExecRequest {
            image_path: r"C:\App\vulnerable.dll".into(),
            command_line: String::new(),
            hash_sha256: String::new(),
            signer: String::new(),
            parent_path: String::new(),
        };
        let d = eng.evaluate_exec(&req);
        assert!(matches!(d, Decision::Deny { .. }));
    }

    #[test]
    fn usb_deny_monitor() {
        let eng = EnforceEngine::new(sample_policy());
        let d = eng.evaluate_device(&DeviceRequest {
            device_class: "usb_storage".into(),
            device_id: "USB\\VID_1234".into(),
        });
        assert!(matches!(d, Decision::Deny { .. }));
    }

    #[test]
    fn fuzz_parse_does_not_panic_on_garbage_paths() {
        let eng = EnforceEngine::new(sample_policy());
        for pat in ["*", "**", "*a*b*", "\\*\\*"] {
            let _ = glob_match(pat, r"C:\x\y.exe");
        }
        let req = ExecRequest {
            image_path: "\0\0".into(),
            command_line: String::new(),
            hash_sha256: String::new(),
            signer: String::new(),
            parent_path: String::new(),
        };
        let _ = eng.evaluate_exec(&req);
    }

    #[test]
    fn lab_enforce_blocks_app_not_virtual_patch() {
        let _g = crate::enforce::user_land::live_env_lock();
        std::env::remove_var("ERA_ENFORCE_LIVE");
        let mut policy = sample_policy();
        policy.mode = EnforcementMode::Enforce;
        let eng = EnforceEngine::new(policy);
        let malware = eng.evaluate_exec(&ExecRequest {
            image_path: r"C:\Temp\malware.exe".into(),
            command_line: String::new(),
            hash_sha256: String::new(),
            signer: String::new(),
            parent_path: String::new(),
        });
        let br = eng.apply_block(&malware);
        assert!(br.allowed, "process still proceeds (telemetry_only)");
        assert!(br.blocked && br.would_block);
        assert_eq!(br.effect, "telemetry_only");
        assert_eq!(br.kernel_hook, "unavailable");
        assert!(br.hook_message.contains("WHQL"));

        let vp = eng.evaluate_exec(&ExecRequest {
            image_path: r"C:\App\vulnerable.dll".into(),
            command_line: String::new(),
            hash_sha256: String::new(),
            signer: String::new(),
            parent_path: String::new(),
        });
        let vp_br = eng.apply_block(&vp);
        assert!(vp_br.allowed && vp_br.would_block && !vp_br.blocked);
        assert_eq!(vp_br.effect, "telemetry_only");
        assert!(vp_br.hook_message.contains("virtual_patch"));
    }

    #[test]
    fn live_enforce_user_land_block_non_vp() {
        let _g = crate::enforce::user_land::live_env_lock();
        std::env::set_var("ERA_ENFORCE_LIVE", "1");
        let mut policy = sample_policy();
        policy.mode = EnforcementMode::Enforce;
        let eng = EnforceEngine::new(policy);
        let malware = eng.evaluate_exec(&ExecRequest {
            image_path: r"C:\Temp\malware.exe".into(),
            command_line: String::new(),
            hash_sha256: String::new(),
            signer: String::new(),
            parent_path: String::new(),
        });
        let br = eng.apply_block(&malware);
        assert!(!br.allowed);
        assert!(br.blocked && br.would_block);
        assert_eq!(br.effect, "user_land_block");
        let vp = eng.evaluate_exec(&ExecRequest {
            image_path: r"C:\App\vulnerable.dll".into(),
            command_line: String::new(),
            hash_sha256: String::new(),
            signer: String::new(),
            parent_path: String::new(),
        });
        let vp_br = eng.apply_block(&vp);
        assert!(vp_br.allowed && !vp_br.blocked);
        assert_eq!(vp_br.effect, "telemetry_only");
        std::env::remove_var("ERA_ENFORCE_LIVE");
    }
}
