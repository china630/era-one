use std::collections::HashMap;
use std::net::SocketAddr;
use std::sync::Arc;

use era_presentations_engine::server::{self, AppState};
use tokio::sync::Mutex;
use tracing_subscriber::EnvFilter;

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    tracing_subscriber::fmt()
        .with_env_filter(EnvFilter::from_default_env())
        .init();
    let addr: SocketAddr = std::env::var("ERA_PRESENTATIONS_HTTP_ADDR")
        .unwrap_or_else(|_| ":8144".into())
        .parse()
        .unwrap_or_else(|_| SocketAddr::from(([0, 0, 0, 0], 8144)));
    let state = AppState {
        drive_url: std::env::var("ERA_DRIVE_API_URL")
            .unwrap_or_else(|_| "http://127.0.0.1:8175".into()),
        license_ok: server::license_from_env(),
        decks: Arc::new(Mutex::new(HashMap::new())),
        jwt_secret: era_presentations_engine::auth::jwt_secret_from_env(),
    };
    server::serve(addr, state).await
}
