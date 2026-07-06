//! ERA One — agent orchestrator core (ADR-0019).
#![deny(unsafe_code)]

pub mod enforce;
pub mod budget_guard;
pub mod buffer;
pub mod builder;
pub mod capture;
pub mod config;
pub mod license_gate;
pub mod orchestrator;
pub mod ota;
pub mod plugin;
pub mod register;
pub mod sanitize;
pub mod scheduler;
pub mod sender;
pub mod tamper;

pub use era_proto::era::v1::*;

/// Версия агента для Envelope (совпадает с installer `era-agent`).
pub const AGENT_VERSION: &str = env!("CARGO_PKG_VERSION");

/// Тестовые конструкторы (unit/golden-тесты).
pub mod sample {
    use super::builder;
    use super::config::Config;

    pub fn process_envelope(image: &str) -> super::Envelope {
        builder::process_envelope(
            &Config::dev_defaults(),
            "create",
            4242,
            1000,
            image,
            &format!("{image} --token=SECRET123 --user alice"),
            "alice",
            false,
        )
    }
}
