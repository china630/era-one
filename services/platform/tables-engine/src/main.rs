use std::net::SocketAddr;
use std::sync::Arc;

use era_tables_engine::drive_bind::platform_storage;
use era_tables_engine::server::{self, AppState, PresenceHub, SessionStore, SyncHub};
use tokio::sync::Mutex;
use tracing_subscriber::EnvFilter;

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    tracing_subscriber::fmt()
        .with_env_filter(EnvFilter::from_default_env())
        .init();
    let addr: SocketAddr = std::env::var("ERA_TABLES_HTTP_ADDR")
        .unwrap_or_else(|_| ":8143".into())
        .parse()
        .unwrap_or_else(|_| SocketAddr::from(([0, 0, 0, 0], 8143)));
    let drive_url = std::env::var("ERA_DRIVE_API_URL")
        .unwrap_or_else(|_| "http://127.0.0.1:8175".into());
    let state = AppState {
        drive_url: drive_url.clone(),
        storage: platform_storage(drive_url),
        license_ok: server::license_from_env(),
        sessions: Arc::new(Mutex::new(SessionStore::default())),
        hub: SyncHub::default(),
        presence: PresenceHub::default(),
        jwt_secret: era_tables_engine::auth::jwt_secret_from_env(),
    };
    server::serve(addr, state).await
}
