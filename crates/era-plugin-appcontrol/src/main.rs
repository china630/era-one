//! Application Control — monitor stub (ADR-0012). Боевой enforce — [gate: external].
#![deny(unsafe_code)]

mod hooks;

use anyhow::Result;
use era_plugin_sdk::{emit, EnforcementStatus};
use hooks::hook_status;

fn main() -> Result<()> {
    let mut status = EnforcementStatus::ready("appcontrol", hook_status());
    if let Ok(path) = std::env::var("ERA_SIM_EXEC_PATH") {
        status.detail = format!("sim_check path={path}");
        if path.to_lowercase().contains("malware") {
            status.detail.push_str(" would_block=true");
        }
    }
    emit(&status)?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;
    use std::path::PathBuf;

    fn golden_path(name: &str) -> PathBuf {
        PathBuf::from(env!("CARGO_MANIFEST_DIR"))
            .join("testdata")
            .join(name)
    }

    #[test]
    fn golden_status_emission() {
        std::env::remove_var("ERA_ENFORCE_LIVE");
        let status = EnforcementStatus::ready("appcontrol", hook_status());
        let got = serde_json::to_string(&status).unwrap();
        let want = fs::read_to_string(golden_path("status.golden.json"))
            .unwrap()
            .trim()
            .to_string();
        assert_eq!(got, want);
    }
}
