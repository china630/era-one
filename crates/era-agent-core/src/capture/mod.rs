//! Capture: stub + sysinfo + multi-domain stubs (F2-2) + production (GA-1).

mod domain_stub;
mod production;
mod stub;
mod sysinfo_cap;

pub use production::is_production_mode;

mod linux_audit;
mod windows_events;
mod macos_unified;
mod evtx_parser;

use crate::builder;
use crate::config::Config;
use crate::Envelope;

pub fn stub_event(cfg: &Config) -> Envelope {
    builder::process_envelope(
        cfg,
        "create",
        4242,
        1000,
        "C:/Windows/System32/cmd.exe",
        "cmd.exe --token=SECRET123 --user alice",
        "alice",
        false,
    )
}

pub enum CaptureBackend {
    Stub(stub::StubCapture),
    Sysinfo(sysinfo_cap::SysinfoCapture),
    Domain(domain_stub::DomainCapture),
    Production(production::PlatformCapture),
}

impl CaptureBackend {
    pub fn new(cfg: &Config) -> Self {
        if production::is_production_mode() {
            if std::env::var("ERA_DOMAIN_STUB").is_ok() || std::env::var("ERA_CAPTURE_STUB").is_ok() {
                tracing::warn!(
                    "ERA_PRODUCTION=1: игнорируем ERA_CAPTURE_STUB/ERA_DOMAIN_STUB (fail-closed prod)"
                );
            }
            return CaptureBackend::Production(production::PlatformCapture::new(cfg));
        }
        if std::env::var("ERA_DOMAIN_STUB").is_ok() {
            return CaptureBackend::Domain(domain_stub::DomainCapture::new(cfg.clone()));
        }
        if std::env::var("ERA_CAPTURE_STUB").is_ok() {
            return CaptureBackend::Stub(stub::StubCapture::new());
        }
        CaptureBackend::Sysinfo(sysinfo_cap::SysinfoCapture::new(cfg))
    }

    pub fn poll(&mut self) -> Vec<Envelope> {
        match self {
            CaptureBackend::Stub(s) => s.poll(),
            CaptureBackend::Sysinfo(s) => s.poll(),
            CaptureBackend::Domain(d) => d.poll(),
            CaptureBackend::Production(p) => p.poll(),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::config::Config;

    #[test]
    fn production_ignores_stub_env() {
        std::env::set_var("ERA_PRODUCTION", "1");
        std::env::set_var("ERA_CAPTURE_STUB", "1");
        let cfg = Config::dev_defaults();
        let backend = CaptureBackend::new(&cfg);
        assert!(matches!(backend, CaptureBackend::Production(_)));
        std::env::remove_var("ERA_PRODUCTION");
        std::env::remove_var("ERA_CAPTURE_STUB");
    }
}
