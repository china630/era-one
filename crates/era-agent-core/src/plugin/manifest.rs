//! Манифест плагина (Vision §5.3).

use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum PluginMode {
    Realtime,
    Cron,
    OnDemand,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PluginManifest {
    pub name: String,
    pub mode: PluginMode,
    pub license_module: String,
    #[serde(default = "default_schedule")]
    pub schedule_secs: u64,
    #[serde(default)]
    pub budget_hint_mb: u64,
    /// Путь к бинарю плагина (относительно ERA_PLUGIN_DIR или абсолютный).
    pub binary: String,
}

fn default_schedule() -> u64 {
    3600
}

impl PluginManifest {
    pub fn inventory_dev() -> Self {
        Self {
            name: "inventory".into(),
            mode: PluginMode::Cron,
            license_module: "manage".into(),
            schedule_secs: 3600,
            budget_hint_mb: 20,
            binary: "era-plugin-inventory".into(),
        }
    }

    pub fn appcontrol_dev() -> Self {
        Self {
            name: "appcontrol".into(),
            mode: PluginMode::OnDemand,
            license_module: "manage".into(),
            schedule_secs: 0,
            budget_hint_mb: 10,
            binary: "era-plugin-appcontrol".into(),
        }
    }

    pub fn devicecontrol_dev() -> Self {
        Self {
            name: "devicecontrol".into(),
            mode: PluginMode::OnDemand,
            license_module: "manage".into(),
            schedule_secs: 0,
            budget_hint_mb: 8,
            binary: "era-plugin-devicecontrol".into(),
        }
    }

    pub fn bitlocker_dev() -> Self {
        Self {
            name: "bitlocker".into(),
            mode: PluginMode::OnDemand,
            license_module: "manage".into(),
            schedule_secs: 0,
            budget_hint_mb: 5,
            binary: "era-plugin-bitlocker".into(),
        }
    }

    pub fn deploy_dev() -> Self {
        Self {
            name: "deploy".into(),
            mode: PluginMode::OnDemand,
            license_module: "manage".into(),
            schedule_secs: 0,
            budget_hint_mb: 15,
            binary: "era-plugin-deploy".into(),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn manifest_roundtrip() {
        let m = PluginManifest::inventory_dev();
        let j = serde_json::to_string(&m).expect("json");
        let back: PluginManifest = serde_json::from_str(&j).expect("parse");
        assert_eq!(back.name, "inventory");
        assert_eq!(back.mode, PluginMode::Cron);
    }
}
