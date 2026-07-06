//! Golden-тест PII-редакции (ADR-0009, S1-7).

use era_agent::sample;
use era_agent::sanitize;
use std::fs;
use std::path::PathBuf;

const KEY: &str = "dev-tenant-key-for-tests-only";

fn testdata(name: &str) -> PathBuf {
    PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("testdata").join(name)
}

#[test]
fn golden_pii_redaction() {
    let raw = sample::process_envelope("C:/Windows/System32/cmd.exe");
    let clean = sanitize::sanitize(raw, KEY);
    let got = sanitize::process_fields_json(&clean).expect("process payload");

    let golden_path = testdata("process_sanitized.golden.json");
    if std::env::var("UPDATE_GOLDEN").is_ok() {
        fs::write(&golden_path, format!("{got}\n")).expect("write golden");
    }

    let want = fs::read_to_string(&golden_path).expect("read golden");
    assert_eq!(
        got.trim(),
        want.trim(),
        "PII golden mismatch; run with UPDATE_GOLDEN=1 при намеренном изменении"
    );

    assert!(!got.contains("SECRET123"));
    assert!(!got.contains("\"alice\""));
}
