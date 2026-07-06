//! Golden: BYO-EDR JSON/syslog → Envelope wire (ADR-0017 §3).

use std::path::Path;

use era_collectors::{parse_json_line, parse_syslog_line, ByoEdrConfig, GENERIC_SOURCE_TYPE};
use era_proto::envelope;
use prost::Message;
use prost_types::Timestamp;

fn cfg() -> ByoEdrConfig {
    let mut id = [0u8; 16];
    id.copy_from_slice(b"0123456789abcdef");
    ByoEdrConfig {
        tenant_id: "t1".into(),
        node_id: "n1".into(),
        hostname: "host1".into(),
        agent_id: "byo-edr-collector".into(),
        fixed_event_id: Some(id),
        fixed_observed_at: Some(Timestamp {
            seconds: 1_780_300_800,
            nanos: 0,
        }),
    }
}

#[test]
fn golden_json_feed() {
    let line = include_str!("testdata/byo_edr_feed.json");
    let env = parse_json_line(line.trim(), &cfg()).expect("parse json");
    let wire = env.encode_to_vec();
    let golden = Path::new("tests/testdata/byo_edr_envelope.golden.bin");
    if std::env::var("UPDATE_GOLDEN").as_deref() == Ok("1") {
        std::fs::write(golden, &wire).expect("write golden");
    }
    let want = std::fs::read(golden).unwrap_or_else(|e| {
        panic!("golden missing: {e}; run UPDATE_GOLDEN=1");
    });
    assert_eq!(
        wire.as_slice(),
        want.as_slice(),
        "golden mismatch; run UPDATE_GOLDEN=1"
    );
    if let Some(envelope::Payload::Raw(raw)) = env.payload {
        assert_eq!(raw.source_type, GENERIC_SOURCE_TYPE);
    } else {
        panic!("expected raw");
    }
}

#[test]
fn golden_syslog_cef() {
    let line = include_str!("testdata/byo_edr_syslog.cef");
    let env = parse_syslog_line(line.trim(), &cfg()).expect("parse cef");
    assert_eq!(env.source.as_ref().unwrap().node_id, "n1");
    assert!(env.pii_sanitized);
}
