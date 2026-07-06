//! Enforcement policy engine (ADR-0012): offline allow/deny, monitor/enforce, fail-open.

mod check;
mod engine;
mod envelope;
mod loader;
mod policy;

pub use check::check_process_envelope;
pub use engine::{Decision, EnforceEngine, ExecRequest, DeviceRequest};
pub use envelope::enforcement_detection_envelope;
pub use loader::fetch_policy;
pub use policy::{
    AppRule, DeviceRule, EnforcementMode, EnforcementPolicy, FailMode, RuleAction,
    VirtualPatchRule,
};
