//! Production capture backend: platform-native sources, без dev-stub.

#[cfg(not(any(target_os = "linux", target_os = "windows", target_os = "macos")))]
use crate::config::Config;
#[cfg(not(any(target_os = "linux", target_os = "windows", target_os = "macos")))]
use crate::Envelope;

#[cfg(target_os = "linux")]
pub use super::linux_audit::LinuxAuditCapture as PlatformCapture;

#[cfg(target_os = "windows")]
pub use super::windows_events::WindowsEventsCapture as PlatformCapture;

#[cfg(target_os = "macos")]
pub use super::macos_unified::MacosUnifiedCapture as PlatformCapture;

#[cfg(not(any(target_os = "linux", target_os = "windows", target_os = "macos")))]
pub struct PlatformCapture {
    _cfg: Config,
}

#[cfg(not(any(target_os = "linux", target_os = "windows", target_os = "macos")))]
impl PlatformCapture {
    pub fn new(cfg: &Config) -> Self {
        Self { _cfg: cfg.clone() }
    }
    pub fn poll(&mut self) -> Vec<Envelope> {
        Vec::new()
    }
}

pub fn is_production_mode() -> bool {
    std::env::var("ERA_PRODUCTION")
        .map(|v| v == "1" || v.eq_ignore_ascii_case("true"))
        .unwrap_or(false)
}
