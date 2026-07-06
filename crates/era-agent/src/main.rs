//! ERA XDR era-agent — installer entrypoint.
#![deny(unsafe_code)]

use anyhow::Context;
use era_agent_core::config::Config;
use era_agent_core::orchestrator;

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    tracing_subscriber::fmt().with_target(false).init();
    let cfg = Config::from_env().context("config")?;
    orchestrator::run(cfg).await
}
