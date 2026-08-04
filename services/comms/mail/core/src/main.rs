//! ERA Mail Server — SMTP/IMAP daemon entrypoint.

use anyhow::Result;
use era_mail_core::{remote_store, run, Config};
use tracing_subscriber::EnvFilter;

fn main() -> Result<()> {
    tracing_subscriber::fmt()
        .with_env_filter(EnvFilter::from_default_env().add_directive("era_mail_core=info".parse()?))
        .init();

    let cfg = Config::from_env();
    if std::env::var("ERA_MAIL_API_URL").map(|v| !v.trim().is_empty()).unwrap_or(false) {
        remote_store::init_blocking_http();
    }
    let store = remote_store::open_backend()?;
    tokio::runtime::Runtime::new()?.block_on(run(cfg, store))
}
