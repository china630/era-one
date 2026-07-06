//! Generic JSON/syslog → Envelope для стороннего EDR/SIEM.

use std::collections::BTreeMap;
use std::time::SystemTime;

use era_proto::{
    envelope, EventCategory, Envelope, OcsfMeta, Platform, RawEvent, Severity, Source,
};
use prost_types::{value::Kind, Struct, Value};
use prost_types::Timestamp;
use serde::Deserialize;
use ulid::Ulid;

pub const GENERIC_SOURCE_TYPE: &str = "era.byo-edr.generic";

/// Конфиг коллектора (tenant/node из политики control-plane).
#[derive(Debug, Clone)]
pub struct ByoEdrConfig {
    pub tenant_id: String,
    pub node_id: String,
    pub hostname: String,
    pub agent_id: String,
    /// Для golden-тестов: фиксированный ULID (16 байт).
    pub fixed_event_id: Option<[u8; 16]>,
    /// Для golden-тестов: фиксированное время.
    pub fixed_observed_at: Option<Timestamp>,
}

impl Default for ByoEdrConfig {
    fn default() -> Self {
        Self {
            tenant_id: "default".into(),
            node_id: "byo-edr-node".into(),
            hostname: "collector".into(),
            agent_id: "byo-edr-collector".into(),
            fixed_event_id: None,
            fixed_observed_at: None,
        }
    }
}

#[derive(Debug, thiserror::Error)]
pub enum ByoEdrError {
    #[error("empty input")]
    Empty,
    #[error("invalid json: {0}")]
    Json(#[from] serde_json::Error),
    #[error("unrecognized syslog format")]
    UnrecognizedSyslog,
}

#[derive(Debug, Deserialize)]
struct GenericJsonEvent {
    #[serde(default)]
    event_type: String,
    #[serde(default)]
    severity: String,
    #[serde(default)]
    summary: String,
    #[serde(default)]
    user: String,
    #[serde(default)]
    host: String,
    #[serde(default)]
    src_ip: String,
    #[serde(default)]
    category: String,
}

/// Парсит одну строку JSON feed (NDJSON) в Envelope.
pub fn parse_json_line(line: &str, cfg: &ByoEdrConfig) -> Result<Envelope, ByoEdrError> {
    let line = line.trim();
    if line.is_empty() {
        return Err(ByoEdrError::Empty);
    }
    let ev: GenericJsonEvent = serde_json::from_str(line)?;
    let summary = if ev.summary.is_empty() {
        if ev.event_type.is_empty() {
            "byo-edr alert".into()
        } else {
            ev.event_type.clone()
        }
    } else {
        ev.summary.clone()
    };
    let mut fields = BTreeMap::new();
    insert_str(&mut fields, "summary", &summary);
    if !ev.event_type.is_empty() {
        insert_str(&mut fields, "event_type", &ev.event_type);
    }
    if !ev.user.is_empty() {
        insert_str(&mut fields, "user", &ev.user);
    }
    if !ev.host.is_empty() {
        insert_str(&mut fields, "host", &ev.host);
    }
    if !ev.src_ip.is_empty() {
        insert_str(&mut fields, "src_ip", &ev.src_ip);
    }
    if !ev.category.is_empty() {
        insert_str(&mut fields, "category", &ev.category);
    }
    Ok(build_envelope(
        cfg,
        map_severity(&ev.severity),
        map_category(&ev.category),
        summary,
        fields,
    ))
}

/// Парсит syslog/CEF строку в Envelope.
pub fn parse_syslog_line(line: &str, cfg: &ByoEdrConfig) -> Result<Envelope, ByoEdrError> {
    let line = line.trim();
    if line.is_empty() {
        return Err(ByoEdrError::Empty);
    }
    if let Some(rest) = line.strip_prefix("CEF:") {
        return parse_cef(rest, cfg);
    }
    // Простой syslog: "<pri>timestamp host msg"
    let msg = line.splitn(3, ' ').last().unwrap_or(line);
    let mut fields = BTreeMap::new();
    insert_str(&mut fields, "summary", msg);
    insert_str(&mut fields, "raw", line);
    Ok(build_envelope(
        cfg,
        Severity::Medium,
        EventCategory::Module,
        msg.to_string(),
        fields,
    ))
}

fn parse_cef(rest: &str, cfg: &ByoEdrConfig) -> Result<Envelope, ByoEdrError> {
    // CEF:Version|Vendor|Product|Version|SignatureID|Name|Severity|extension
    let parts: Vec<&str> = rest.splitn(8, '|').collect();
    if parts.len() < 7 {
        return Err(ByoEdrError::UnrecognizedSyslog);
    }
    let name = parts[5];
    let sev_num: u32 = parts[6].parse().unwrap_or(5);
    let severity = cef_severity(sev_num);
    let mut fields = BTreeMap::new();
    insert_str(&mut fields, "summary", name);
    insert_str(&mut fields, "vendor", parts.get(1).copied().unwrap_or(""));
    insert_str(&mut fields, "product", parts.get(2).copied().unwrap_or(""));
    if let Some(ext) = parts.get(7) {
        for kv in ext.split_whitespace() {
            if let Some((k, v)) = kv.split_once('=') {
                insert_str(&mut fields, k, v);
            }
        }
    }
    Ok(build_envelope(
        cfg,
        severity,
        EventCategory::Process,
        name.to_string(),
        fields,
    ))
}

fn build_envelope(
    cfg: &ByoEdrConfig,
    severity: Severity,
    category: EventCategory,
    _summary: String,
    fields: BTreeMap<String, Value>,
) -> Envelope {
    Envelope {
        schema_version: "1.0.0".into(),
        event_id: cfg
            .fixed_event_id
            .map(|b| b.to_vec())
            .unwrap_or_else(|| Ulid::new().to_bytes().to_vec()),
        observed_at: cfg
            .fixed_observed_at
            .or_else(|| Some(Timestamp::from(SystemTime::now()))),
        source: Some(Source {
            tenant_id: cfg.tenant_id.clone(),
            node_id: cfg.node_id.clone(),
            hostname: if cfg.hostname.is_empty() {
                cfg.node_id.clone()
            } else {
                cfg.hostname.clone()
            },
            agent_id: cfg.agent_id.clone(),
            agent_version: "0.1.0".into(),
            platform: Platform::Unspecified as i32,
            ..Default::default()
        }),
        severity: severity as i32,
        category: category as i32,
        ocsf: Some(OcsfMeta {
            class_uid: 2001,
            category_uid: 1,
            activity_id: 1,
        }),
        pii_sanitized: true,
        payload: Some(envelope::Payload::Raw(RawEvent {
            source_type: GENERIC_SOURCE_TYPE.into(),
            fields: Some(Struct { fields }),
        })),
        ..Default::default()
    }
}

fn insert_str(fields: &mut BTreeMap<String, Value>, key: &str, val: &str) {
    fields.insert(
        key.into(),
        Value {
            kind: Some(Kind::StringValue(val.to_string())),
        },
    );
}

fn map_severity(s: &str) -> Severity {
    match s.to_lowercase().as_str() {
        "critical" | "crit" => Severity::Critical,
        "high" => Severity::High,
        "medium" | "med" => Severity::Medium,
        "low" => Severity::Low,
        "info" | "informational" => Severity::Info,
        _ => Severity::Medium,
    }
}

fn map_category(c: &str) -> EventCategory {
    match c.to_lowercase().as_str() {
        "auth" | "authentication" => EventCategory::Auth,
        "network" => EventCategory::Network,
        "file" => EventCategory::File,
        "process" => EventCategory::Process,
        _ => EventCategory::Module,
    }
}

fn cef_severity(n: u32) -> Severity {
    match n {
        0..=3 => Severity::Low,
        4..=6 => Severity::Medium,
        7..=8 => Severity::High,
        _ => Severity::Critical,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use prost::Message;

    #[test]
    fn json_maps_to_raw_envelope() {
        let cfg = ByoEdrConfig::default();
        let line = r#"{"event_type":"alert","severity":"high","summary":"failed logon","user":"admin","category":"auth"}"#;
        let env = parse_json_line(line, &cfg).expect("parse");
        assert_eq!(env.schema_version, "1.0.0");
        assert_eq!(env.severity, Severity::High as i32);
        assert!(env.pii_sanitized);
        if let Some(envelope::Payload::Raw(raw)) = env.payload {
            assert_eq!(raw.source_type, GENERIC_SOURCE_TYPE);
        } else {
            panic!("expected raw payload");
        }
    }

    #[test]
    fn syslog_cef_parses() {
        let cfg = ByoEdrConfig::default();
        let line = "CEF:0|Vendor|EDR|1.0|100|Suspicious Process|8|src=10.0.0.1 dst=10.0.0.2";
        let env = parse_syslog_line(line, &cfg).expect("cef");
        assert_eq!(env.severity, Severity::High as i32);
        let wire = env.encode_to_vec();
        let back = Envelope::decode(wire.as_slice()).expect("roundtrip");
        assert_eq!(back.source.as_ref().unwrap().agent_id, "byo-edr-collector");
    }
}
