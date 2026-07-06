//! Сборка Envelope из capture (ADR-0001, OCSF class 1007 Process Activity).

use crate::config::Config;
use crate::{
    envelope, AuthEvent, Envelope, EventCategory, FileEvent, NetworkEvent, OcsfMeta, Platform,
    ProcessEvent, RawEvent, Severity, Source,
};
use prost_types::Timestamp;
use std::time::SystemTime;
use ulid::Ulid;

pub fn process_envelope(
    cfg: &Config,
    action: &str,
    pid: u64,
    ppid: u64,
    image_path: &str,
    command_line: &str,
    user: &str,
    is_elevated: bool,
) -> Envelope {
    let platform = if cfg!(target_os = "windows") {
        Platform::Windows as i32
    } else if cfg!(target_os = "macos") {
        Platform::Macos as i32
    } else {
        Platform::Linux as i32
    };

    Envelope {
        schema_version: "1.0.0".into(),
        event_id: Ulid::new().to_bytes().to_vec(),
        observed_at: Some(Timestamp::from(SystemTime::now())),
        source: Some(Source {
            tenant_id: cfg.tenant_id.clone(),
            node_id: cfg.node_id.clone(),
            hostname: cfg.hostname.clone(),
            agent_id: cfg.agent_id.clone(),
            agent_version: crate::AGENT_VERSION.into(),
            platform,
            ..Default::default()
        }),
        severity: Severity::Medium as i32,
        category: EventCategory::Process as i32,
        ocsf: Some(OcsfMeta {
            class_uid: 1007,
            category_uid: 1,
            activity_id: 1,
        }),
        pii_sanitized: false,
        payload: Some(envelope::Payload::Process(ProcessEvent {
            action: action.into(),
            pid,
            ppid,
            image_path: image_path.into(),
            command_line: command_line.into(),
            user: user.into(),
            is_elevated,
            ..Default::default()
        })),
        ..Default::default()
    }
}

pub fn network_envelope(cfg: &Config, dst_ip: &str, dst_port: u32) -> Envelope {
    base_envelope(cfg, EventCategory::Network, Severity::Medium, 4001, envelope::Payload::Network(
        NetworkEvent {
            direction: "outbound".into(),
            protocol: "tcp".into(),
            src_ip: "10.0.0.12".into(),
            src_port: 49152,
            dst_ip: dst_ip.into(),
            dst_port,
            ..Default::default()
        },
    ))
}

pub fn auth_envelope(cfg: &Config, user: &str, success: bool) -> Envelope {
    base_envelope(cfg, EventCategory::Auth, Severity::Medium, 3002, envelope::Payload::Auth(
        AuthEvent {
            action: if success { "logon".into() } else { "failed".into() },
            user: user.into(),
            logon_type: "interactive".into(),
            success,
            ..Default::default()
        },
    ))
}

pub fn file_envelope(cfg: &Config, path: &str) -> Envelope {
    base_envelope(cfg, EventCategory::File, Severity::Low, 1001, envelope::Payload::File(
        FileEvent {
            action: "create".into(),
            path: path.into(),
            ..Default::default()
        },
    ))
}

pub fn tamper_alert_envelope(cfg: &Config) -> Envelope {
    process_envelope(
        cfg,
        "tamper",
        std::process::id() as u64,
        0,
        "era-agent.exe",
        "tamper protection: unauthorized termination attempt detected",
        "SYSTEM",
        true,
    )
}

/// Inventory snapshot от plugin (domain=inventory → RawEvent).
pub fn inventory_envelope(
    cfg: &Config,
    hostname: &str,
    platform: &str,
    os_version: &str,
    cpu_count: u32,
    total_memory_mb: u64,
    detail_json: &str,
) -> Envelope {
    use prost_types::{value::Kind, Value};
    use std::collections::BTreeMap;

    let mut fields = BTreeMap::new();
    fields.insert(
        "hostname".into(),
        Value {
            kind: Some(Kind::StringValue(
                if hostname.is_empty() {
                    cfg.hostname.clone()
                } else {
                    hostname.to_string()
                },
            )),
        },
    );
    fields.insert(
        "platform".into(),
        Value {
            kind: Some(Kind::StringValue(platform.to_string())),
        },
    );
    fields.insert(
        "os_version".into(),
        Value {
            kind: Some(Kind::StringValue(os_version.to_string())),
        },
    );
    fields.insert(
        "cpu_count".into(),
        Value {
            kind: Some(Kind::NumberValue(cpu_count as f64)),
        },
    );
    fields.insert(
        "total_memory_mb".into(),
        Value {
            kind: Some(Kind::NumberValue(total_memory_mb as f64)),
        },
    );
    if !detail_json.is_empty() {
        fields.insert(
            "detail".into(),
            Value {
                kind: Some(Kind::StringValue(detail_json.to_string())),
            },
        );
    }
    inventory_envelope_from_fields(cfg, &fields)
}

/// Degraded-сигнал: источник capture недоступен (P0-3).
pub fn capture_degraded_envelope(cfg: &Config, source: &str, reason: &str) -> Envelope {
    use prost_types::{Struct, Value};
    let mut fields = std::collections::BTreeMap::new();
    fields.insert(
        "status".into(),
        Value {
            kind: Some(prost_types::value::Kind::StringValue("degraded".into())),
        },
    );
    fields.insert(
        "source".into(),
        Value {
            kind: Some(prost_types::value::Kind::StringValue(source.into())),
        },
    );
    fields.insert(
        "reason".into(),
        Value {
            kind: Some(prost_types::value::Kind::StringValue(reason.into())),
        },
    );
    base_envelope(
        cfg,
        EventCategory::Module,
        Severity::Medium,
        5002,
        envelope::Payload::Raw(RawEvent {
            source_type: "era.agent.capture_health".into(),
            fields: Some(Struct { fields }),
        }),
    )
}

/// Полный ITAM snapshot (ADR-0011) из полей плагина.
pub fn inventory_envelope_from_fields(
    cfg: &Config,
    fields: &std::collections::BTreeMap<String, prost_types::Value>,
) -> Envelope {
    use prost_types::Struct;
    base_envelope(
        cfg,
        EventCategory::Module,
        Severity::Info,
        5001,
        envelope::Payload::Raw(RawEvent {
            source_type: "era.inventory.host_snapshot".into(),
            fields: Some(Struct {
                fields: fields.clone(),
            }),
        }),
    )
}

fn base_envelope(cfg: &Config, category: EventCategory, severity: Severity, class_uid: u32, payload: envelope::Payload) -> Envelope {
    let platform = if cfg!(target_os = "windows") {
        Platform::Windows as i32
    } else if cfg!(target_os = "macos") {
        Platform::Macos as i32
    } else {
        Platform::Linux as i32
    };
    Envelope {
        schema_version: "1.0.0".into(),
        event_id: Ulid::new().to_bytes().to_vec(),
        observed_at: Some(Timestamp::from(SystemTime::now())),
        source: Some(Source {
            tenant_id: cfg.tenant_id.clone(),
            node_id: cfg.node_id.clone(),
            hostname: cfg.hostname.clone(),
            agent_id: cfg.agent_id.clone(),
            agent_version: crate::AGENT_VERSION.into(),
            platform,
            ..Default::default()
        }),
        severity: severity as i32,
        category: category as i32,
        ocsf: Some(OcsfMeta {
            class_uid,
            category_uid: 1,
            activity_id: 1,
        }),
        pii_sanitized: false,
        payload: Some(payload),
        ..Default::default()
    }
}
