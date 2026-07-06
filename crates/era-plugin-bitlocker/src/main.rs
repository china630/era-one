//! BitLocker on-demand — статус тома + escrow request (ключи только через CP API).
#![deny(unsafe_code)]

use anyhow::Result;
use era_plugin_sdk::emit;
use serde::Serialize;

#[derive(Debug, Serialize)]
struct BitlockerStatus {
    domain: String,
    kind: String,
    volume_id: String,
    protection: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    escrow_pending: String,
}

/// Читает protection из ERA_BITLOCKER_STATUS или дефолт on.
fn read_protection() -> String {
    match std::env::var("ERA_BITLOCKER_STATUS") {
        Ok(v) => {
            let v = v.trim().to_lowercase();
            if v.is_empty() {
                "on".into()
            } else {
                v
            }
        }
        Err(_) => "on".into(),
    }
}

fn main() -> Result<()> {
    let volume = std::env::var("ERA_BITLOCKER_VOLUME").unwrap_or_else(|_| "C:".into());
    let mut rec = BitlockerStatus {
        domain: "bitlocker".into(),
        kind: "volume_status".into(),
        volume_id: volume,
        protection: read_protection(),
        escrow_pending: String::new(),
    };
    if std::env::var("ERA_BITLOCKER_ESCROW_REQUEST")
        .map(|v| v == "1")
        .unwrap_or(false)
    {
        rec.escrow_pending = "awaiting_cp_upload".into();
    }
    emit(&rec)?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn status_from_env() {
        std::env::set_var("ERA_BITLOCKER_STATUS", "off");
        assert_eq!(read_protection(), "off");
        std::env::remove_var("ERA_BITLOCKER_STATUS");
        assert_eq!(read_protection(), "on");
    }

    #[test]
    fn status_json_stable() {
        let rec = BitlockerStatus {
            domain: "bitlocker".into(),
            kind: "volume_status".into(),
            volume_id: "C:".into(),
            protection: "on".into(),
            escrow_pending: String::new(),
        };
        let j = serde_json::to_string(&rec).unwrap();
        assert!(j.contains("\"protection\":\"on\""));
    }
}
