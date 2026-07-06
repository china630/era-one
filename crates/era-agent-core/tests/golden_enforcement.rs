//! Golden: фикс. policy + бинари → allow/deny (ADR-0012).

use era_agent_core::enforce::{
    Decision, EnforceEngine, EnforcementPolicy, ExecRequest,
};
use std::fs;
use std::path::PathBuf;

fn testdata(name: &str) -> PathBuf {
    PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join("src/enforce/testdata")
        .join(name)
}

#[derive(serde::Deserialize)]
struct GoldenRow {
    image_path: String,
    decision: String,
    rule_id: Option<String>,
}

#[test]
fn golden_allow_deny_exec() {
    let policy_json = fs::read_to_string(testdata("enforcement_policy.json")).unwrap();
    let policy = EnforcementPolicy::parse_json(&policy_json).unwrap();
    let eng = EnforceEngine::new(policy);

    let rows: Vec<GoldenRow> =
        serde_json::from_str(&fs::read_to_string(testdata("allow_deny.golden.json")).unwrap())
            .unwrap();

    let mut got = Vec::new();
    for row in &rows {
        let d = eng.evaluate_exec(&ExecRequest {
            image_path: row.image_path.clone(),
            command_line: String::new(),
            hash_sha256: String::new(),
            signer: String::new(),
            parent_path: String::new(),
        });
        let (decision, rule_id) = match &d {
            Decision::Allow => ("allow".to_string(), None),
            Decision::Deny { rule_id, .. } => ("deny".to_string(), Some(rule_id.clone())),
        };
        got.push(serde_json::json!({
            "image_path": row.image_path,
            "decision": decision,
            "rule_id": rule_id,
        }));
        assert_eq!(decision, row.decision, "{}", row.image_path);
        if row.decision == "deny" {
            assert_eq!(rule_id.as_deref(), row.rule_id.as_deref());
        }
    }

    if std::env::var("UPDATE_GOLDEN").is_ok() {
        let b = serde_json::to_string_pretty(&got).unwrap();
        fs::write(testdata("allow_deny_out.golden.json"), b).unwrap();
    }
}
