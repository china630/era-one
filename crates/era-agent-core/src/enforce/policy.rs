//! Enforcement policy model (JSON from control-plane bundle).

use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum EnforcementMode {
    Monitor,
    Enforce,
}

impl Default for EnforcementMode {
    fn default() -> Self {
        Self::Monitor
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum FailMode {
    Open,
    Closed,
}

impl Default for FailMode {
    fn default() -> Self {
        Self::Open
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum RuleAction {
    Allow,
    Deny,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct AppRule {
    pub id: String,
    pub action: RuleAction,
    #[serde(default)]
    pub path: String,
    #[serde(default)]
    pub hash_sha256: String,
    #[serde(default)]
    pub signer: String,
    #[serde(default)]
    pub parent_path: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct DeviceRule {
    pub id: String,
    pub action: RuleAction,
    pub device_class: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct VirtualPatchRule {
    pub id: String,
    pub cve_id: String,
    pub action: RuleAction,
    #[serde(default)]
    pub path: String,
    #[serde(default)]
    pub vector: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct EnforcementPolicy {
    pub version: String,
    #[serde(default)]
    pub mode: EnforcementMode,
    #[serde(default)]
    pub fail_mode: FailMode,
    #[serde(default)]
    pub app_rules: Vec<AppRule>,
    #[serde(default)]
    pub device_rules: Vec<DeviceRule>,
    #[serde(default, rename = "virtual_patches")]
    pub virtual_patches: Vec<VirtualPatchRule>,
}

impl Default for EnforcementPolicy {
    fn default() -> Self {
        Self {
            version: "0.0.0".into(),
            mode: EnforcementMode::Monitor,
            fail_mode: FailMode::Open,
            app_rules: Vec::new(),
            device_rules: Vec::new(),
            virtual_patches: Vec::new(),
        }
    }
}

impl EnforcementPolicy {
    pub fn parse_json(s: &str) -> Result<Self, serde_json::Error> {
        serde_json::from_str(s)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn fuzz_policy_parse_no_panic() {
        for s in [
            "",
            "{",
            "null",
            r#"{"version":"1.0"}"#,
            r#"{"mode":"monitor","app_rules":[{"id":"x"}]}"#,
            r#"{"virtual_patches":[{"id":"v","cve_id":"CVE-1","action":"deny","path":"*"}]}"#,
        ] {
            let _ = EnforcementPolicy::parse_json(s);
        }
    }
}
