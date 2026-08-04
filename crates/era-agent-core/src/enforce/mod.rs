//! Enforcement policy engine (ADR-0012): offline allow/deny, monitor/enforce, fail-open.

mod check;
mod engine;
mod envelope;
mod kernel;
mod loader;
mod policy;
mod user_land;

pub use check::check_process_envelope;
pub use engine::{BlockResult, Decision, DeviceRequest, EnforceEngine, ExecRequest};
pub use envelope::enforcement_detection_envelope;
pub use kernel::{probe_kernel_hook, KernelHookStatus};
pub use loader::fetch_policy;
pub use policy::{
    AppRule, DeviceRule, EnforcementMode, EnforcementPolicy, FailMode, RuleAction,
    VirtualPatchRule,
};
pub use user_land::{enforce_live_enabled, live_env_lock, try_user_land_block};
