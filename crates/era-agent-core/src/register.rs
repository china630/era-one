//! Heartbeat регистрации в control-plane (F2-7 asset inventory).

use crate::config::Config;
use std::time::Duration;
use tracing::{debug, warn};

pub fn spawn_heartbeat(cfg: Config) {
    let url = cfg.control_plane_url.clone();
    if url.is_empty() {
        return;
    }
    tokio::spawn(async move {
        let client = ureq::Agent::new();
        let platform = if cfg!(target_os = "windows") {
            "windows"
        } else if cfg!(target_os = "macos") {
            "macos"
        } else {
            "linux"
        };
        loop {
            let body = serde_json::json!({
                "agent_id": cfg.agent_id,
                "tenant_id": cfg.tenant_id,
                "node_id": cfg.node_id,
                "hostname": cfg.hostname,
                "platform": platform,
                "agent_version": crate::AGENT_VERSION,
            });
            let endpoint = format!("{url}/api/v1/assets/register");
            match client.post(&endpoint).send_json(body) {
                Ok(resp) if resp.status() / 100 == 2 => {
                    debug!("asset register OK coverage updated");
                }
                Ok(resp) => {
                    warn!("asset register HTTP {}", resp.status());
                }
                Err(e) => {
                    warn!("asset register failed: {e}");
                }
            }
            tokio::time::sleep(Duration::from_secs(cfg.heartbeat_secs)).await;
        }
    });
}
