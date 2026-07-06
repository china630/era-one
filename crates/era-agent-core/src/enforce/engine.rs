//! Offline policy matching (ADR-0012 §3).

use crate::enforce::policy::{
    AppRule, DeviceRule, EnforcementMode, EnforcementPolicy, FailMode, RuleAction,
    VirtualPatchRule,
};

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

    /// Применяет режим monitor/enforce и fail-open. Возвращает (разрешён_запуск, блокировать_в_enforce).
    pub fn apply_exec(&self, decision: &Decision) -> (bool, bool) {
        match decision {
            Decision::Allow => (true, false),
            Decision::Deny { .. } => match self.policy.mode {
                EnforcementMode::Monitor => (true, true),
                EnforcementMode::Enforce => (false, true),
            },
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
}
