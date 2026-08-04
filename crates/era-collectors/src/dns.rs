//! DNS Trace collector — best-effort DnsEvent emitter (Resolve Phase 2).
//!
//! Windows: ETW DNS Client hook is lab/stub (no WHQL). Linux: stub samples.
//! Core NDR tunnel heuristics remain in detection-engine.
//!
//! Emitter always labels `Source.environment = "mode=stub"` until a live hook ships.
//! Query/answers are redacted before `pii_sanitized=true` (ADR-0009).

use era_proto::{
    envelope, DnsEvent, Envelope, EventCategory, OcsfMeta, Platform, Severity, Source,
};
use prost_types::Timestamp;
use std::time::SystemTime;
use ulid::Ulid;

/// Emitter mode metadata (lab stub until WHQL/ETW live path).
pub const DNS_EMITTER_MODE: &str = "mode=stub";

/// Emitter config (tenant/node from agent).
#[derive(Debug, Clone)]
pub struct DnsTraceConfig {
    pub tenant_id: String,
    pub node_id: String,
    pub hostname: String,
    pub agent_id: String,
    pub fixed_event_id: Option<[u8; 16]>,
    pub fixed_observed_at: Option<Timestamp>,
}

impl Default for DnsTraceConfig {
    fn default() -> Self {
        Self {
            tenant_id: "default".into(),
            node_id: "dns-trace-node".into(),
            hostname: "endpoint".into(),
            agent_id: "era-agent".into(),
            fixed_event_id: None,
            fixed_observed_at: None,
        }
    }
}

/// One observed DNS query (from ETW/stub).
#[derive(Debug, Clone)]
pub struct DnsObservation {
    pub query: String,
    pub query_type: String,
    pub answers: Vec<String>,
    pub pid: u64,
}

fn redact_hostname(q: &str) -> String {
    if q.is_empty() {
        return String::new();
    }
    // Keep TLD hint for NDR heuristics without leaking FQDN labels.
    let parts: Vec<&str> = q.trim_end_matches('.').split('.').collect();
    if parts.len() <= 1 {
        return "[redacted:dns]".into();
    }
    format!("[redacted:dns].{}", parts.last().unwrap())
}

/// Build DnsEvent Envelope for ingest.
pub fn emit_dns_event(obs: &DnsObservation, cfg: &DnsTraceConfig) -> Envelope {
    let event_id = cfg
        .fixed_event_id
        .unwrap_or_else(|| Ulid::new().to_bytes())
        .to_vec();
    let observed_at = cfg
        .fixed_observed_at
        .clone()
        .or_else(|| Some(Timestamp::from(SystemTime::now())));
    let platform = if cfg!(target_os = "windows") {
        Platform::Windows as i32
    } else {
        Platform::Linux as i32
    };
    let query = redact_hostname(&obs.query);
    let answers: Vec<String> = obs
        .answers
        .iter()
        .map(|a| redact_hostname(a))
        .collect();
    Envelope {
        schema_version: "1.0.0".into(),
        event_id,
        observed_at,
        source: Some(Source {
            tenant_id: cfg.tenant_id.clone(),
            environment: DNS_EMITTER_MODE.into(),
            node_id: cfg.node_id.clone(),
            hostname: cfg.hostname.clone(),
            agent_id: cfg.agent_id.clone(),
            platform,
            ..Default::default()
        }),
        severity: Severity::Low as i32,
        category: EventCategory::Dns as i32,
        ocsf: Some(OcsfMeta {
            class_uid: 4003,
            category_uid: 4,
            activity_id: 1,
        }),
        pii_sanitized: true,
        payload: Some(envelope::Payload::Dns(DnsEvent {
            query,
            query_type: obs.query_type.clone(),
            answers,
            pid: obs.pid,
        })),
        ..Default::default()
    }
}

/// Linux/Windows lab stub: synthesize a sample observation (no kernel hook).
pub fn stub_sample_observation() -> DnsObservation {
    DnsObservation {
        query: "lab.malware.test".into(),
        query_type: "A".into(),
        answers: vec![],
        pid: 0,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use prost::Message;

    #[test]
    fn emit_dns_golden_shape() {
        let mut cfg = DnsTraceConfig::default();
        cfg.fixed_event_id = Some(*b"0123456789abcdef");
        cfg.fixed_observed_at = Some(Timestamp {
            seconds: 1_700_000_000,
            nanos: 0,
        });
        let env = emit_dns_event(&stub_sample_observation(), &cfg);
        assert_eq!(env.category, EventCategory::Dns as i32);
        assert!(env.pii_sanitized);
        assert_eq!(
            env.source.as_ref().unwrap().environment,
            DNS_EMITTER_MODE
        );
        match env.payload.as_ref().unwrap() {
            envelope::Payload::Dns(d) => {
                assert_eq!(d.query, "[redacted:dns].test");
                assert!(!d.query.contains("malware"));
                assert_eq!(d.query_type, "A");
            }
            _ => panic!("expected Dns payload"),
        }
        let _ = env.encode_to_vec();
    }

    #[test]
    fn stub_mode_is_explicit() {
        assert_eq!(DNS_EMITTER_MODE, "mode=stub");
        let env = emit_dns_event(&stub_sample_observation(), &DnsTraceConfig::default());
        assert_eq!(env.source.as_ref().unwrap().environment, "mode=stub");
    }
}
