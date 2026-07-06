//! Golden: plugin NDJSON → Envelope wire (стабильность маппинга).

use era_agent_core::config::Config;
use era_agent_core::plugin::{inventory_record_to_envelope, parse_ndjson_line};
use prost_types::Timestamp;
use std::fs;
use std::path::PathBuf;

fn testdata(name: &str) -> PathBuf {
    PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join("testdata")
        .join(name)
}

#[test]
fn golden_plugin_ndjson_to_envelope() {
    let line = fs::read_to_string(testdata("inventory_ndjson_line.json"))
        .expect("read ndjson fixture")
        .trim()
        .to_string();
    let rec = parse_ndjson_line(&line).expect("parse");
    let mut env = inventory_record_to_envelope(&Config::dev_defaults(), &rec);
    env.event_id = b"0123456789abcdef".to_vec();
    env.observed_at = Some(Timestamp {
        seconds: 1_700_000_000,
        nanos: 0,
    });
    let got = hex::encode(prost::Message::encode_to_vec(&env));

    let golden_path = testdata("inventory_envelope_wire.golden.hex");
    if std::env::var("UPDATE_GOLDEN").is_ok() {
        fs::write(&golden_path, format!("{got}\n")).expect("write golden");
    }
    let want = fs::read_to_string(&golden_path)
        .expect("golden missing; run UPDATE_GOLDEN=1")
        .trim()
        .to_string();
    assert_eq!(got, want, "envelope wire golden mismatch");
}
