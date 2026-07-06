//! Enforcement detection → Envelope (ADR-0012 §3).

use crate::config::Config;
use crate::enforce::engine::Decision;
use crate::builder;
use crate::{Detection, Envelope, Severity};

pub fn enforcement_detection_envelope(
    cfg: &Config,
    decision: &Decision,
    image_path: &str,
) -> Option<Envelope> {
    let Decision::Deny {
        rule_id,
        engine,
        summary,
    } = decision
    else {
        return None;
    };
    let mut env = builder::process_envelope(
        cfg,
        "blocked",
        0,
        0,
        image_path,
        summary,
        "SYSTEM",
        false,
    );
    env.severity = Severity::High as i32;
    env.detection = Some(Detection {
        rule_id: rule_id.clone(),
        rule_name: summary.clone(),
        severity: Severity::High as i32,
        engine: engine.clone(),
        confidence: 1.0,
    });
    env.pii_sanitized = true;
    Some(env)
}
