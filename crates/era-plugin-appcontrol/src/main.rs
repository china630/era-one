//! Application Control — monitor + lab decision telemetry (ADR-0012). Kernel WHQL — gate.
#![deny(unsafe_code)]

mod hooks;

use anyhow::Result;
use era_plugin_sdk::{emit, EnforcementStatus};
use hooks::{effect, enforce_mode, hook_status, kernel_hook, local_block};
use serde::Serialize;

#[derive(Debug, Serialize)]
struct WouldBlock {
    domain: String,
    kind: String,
    plugin: String,
    hook: String,
    kernel_hook: String,
    enforce_mode: String,
    effect: String,
    image_path: String,
    would_block: bool,
    blocked: bool,
}

fn main() -> Result<()> {
    let hook = hook_status();
    if let Ok(path) = std::env::var("ERA_SIM_EXEC_PATH") {
        if path.to_lowercase().contains("malware") {
            let wb = true;
            emit(&WouldBlock {
                domain: "enforcement".into(),
                kind: "would_block".into(),
                plugin: "appcontrol".into(),
                hook: hook.into(),
                kernel_hook: kernel_hook().into(),
                enforce_mode: enforce_mode().into(),
                effect: effect().into(),
                image_path: path.clone(),
                would_block: wb,
                blocked: local_block(wb),
            })?;
        }
        let mut status = EnforcementStatus::ready("appcontrol", hook);
        status.detail = format!(
            "sim_check path={path} kernel_hook={} mode={} effect={}",
            kernel_hook(),
            enforce_mode(),
            effect()
        );
        if path.to_lowercase().contains("malware") {
            status.detail.push_str(&format!(
                " would_block=true blocked={}",
                local_block(true)
            ));
        }
        emit(&status)?;
        return Ok(());
    }
    let mut status = EnforcementStatus::ready("appcontrol", hook);
    status.detail = format!(
        "kernel_hook={} effect={} — WHQL minifilter not loaded",
        kernel_hook(),
        effect()
    );
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
        let mut status = EnforcementStatus::ready("appcontrol", hook_status());
        status.detail = format!(
            "kernel_hook={} effect={} — WHQL minifilter not loaded",
            kernel_hook(),
            effect()
        );
        let got = serde_json::to_string(&status).unwrap();
        let want = fs::read_to_string(golden_path("status.golden.json"))
            .unwrap()
            .trim()
            .to_string();
        assert_eq!(got, want);
    }

    #[test]
    fn golden_sim_deny() {
        std::env::remove_var("ERA_ENFORCE_LIVE");
        std::env::remove_var("ERA_ENFORCE_MODE");
        let wb = WouldBlock {
            domain: "enforcement".into(),
            kind: "would_block".into(),
            plugin: "appcontrol".into(),
            hook: hook_status().into(),
            kernel_hook: kernel_hook().into(),
            enforce_mode: enforce_mode().into(),
            effect: effect().into(),
            image_path: r"C:\Temp\malware.exe".into(),
            would_block: true,
            blocked: local_block(true),
        };
        let got = serde_json::to_string(&wb).unwrap();
        let want = fs::read_to_string(golden_path("sim_deny.golden.json"))
            .unwrap()
            .trim()
            .to_string();
        assert_eq!(got, want);
    }
}
