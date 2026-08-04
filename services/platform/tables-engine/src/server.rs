use std::collections::HashMap;
use std::net::SocketAddr;
use std::sync::Arc;
use std::time::Duration;

use axum::extract::ws::{Message, WebSocket, WebSocketUpgrade};
use axum::extract::{Path, State};
use axum::http::{HeaderMap, StatusCode};
use axum::response::IntoResponse;
use axum::routing::{get, post};
use axum::{Json, Router};
use futures_util::{SinkExt, StreamExt};
use serde::{Deserialize, Serialize};
use tokio::net::TcpListener;
use tokio::sync::{broadcast, mpsc, Mutex};

use crate::auth;
use crate::calc::recalc;
use crate::convert::{export_xlsx, import_xlsx};
use crate::convert_ods::{export_ods, import_ods};
use crate::model::{Cell, EratSheet};
use crate::sync::{apply_op, blocked_by_protect, OpLog, SheetOp};
use era_office_storage::StorageBackend;

const SYNC_ROOM_CAP: usize = 64;

/// Debounce window before flushing WS edits to Drive (AC-T1/T5).
/// Override with `ERA_TABLES_FLUSH_DEBOUNCE_MS` (tests use a short value).
fn drive_flush_debounce() -> Duration {
    std::env::var("ERA_TABLES_FLUSH_DEBOUNCE_MS")
        .ok()
        .and_then(|v| v.parse::<u64>().ok())
        .map(Duration::from_millis)
        .unwrap_or(Duration::from_secs(2))
}

#[derive(Clone, Default)]
pub struct SyncHub {
    rooms: Arc<Mutex<HashMap<String, broadcast::Sender<String>>>>,
}

impl SyncHub {
    pub async fn subscribe(&self, id: &str) -> broadcast::Receiver<String> {
        let mut rooms = self.rooms.lock().await;
        let tx = rooms.entry(id.to_string()).or_insert_with(|| {
            let (tx, _) = broadcast::channel(SYNC_ROOM_CAP);
            tx
        });
        tx.subscribe()
    }

    pub async fn publish(&self, id: &str, payload: String) {
        let rooms = self.rooms.lock().await;
        if let Some(tx) = rooms.get(id) {
            let _ = tx.send(payload);
        }
    }
}

#[derive(Clone, Serialize, Deserialize)]
pub struct SyncEnvelope {
    pub op: SheetOp,
    pub version: u64,
    pub author_id: String,
    /// Post-recalc cell snapshot so peers refresh formula results (TE-T03).
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub cells: Option<std::collections::BTreeMap<String, Cell>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub active_sheet: Option<usize>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub sheet_names: Option<Vec<String>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub protected: Option<bool>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub freeze_rows: Option<u32>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub freeze_cols: Option<u32>,
}

#[derive(Clone, Default)]
pub struct PresenceHub {
    rooms: Arc<Mutex<HashMap<String, HashMap<String, String>>>>,
}

impl PresenceHub {
    pub async fn join(&self, sheet_id: &str, conn_id: &str, user_id: &str) -> Vec<String> {
        let mut rooms = self.rooms.lock().await;
        let room = rooms.entry(sheet_id.to_string()).or_default();
        room.insert(conn_id.to_string(), user_id.to_string());
        let mut peers: Vec<String> = room.values().cloned().collect();
        peers.sort();
        peers.dedup();
        peers
    }

    pub async fn leave(&self, sheet_id: &str, conn_id: &str) -> Vec<String> {
        let mut rooms = self.rooms.lock().await;
        let peers = if let Some(room) = rooms.get_mut(sheet_id) {
            room.remove(conn_id);
            let mut peers: Vec<String> = room.values().cloned().collect();
            peers.sort();
            peers.dedup();
            if room.is_empty() {
                rooms.remove(sheet_id);
            }
            peers
        } else {
            vec![]
        };
        peers
    }
}

#[derive(Clone)]
pub struct AppState {
    pub drive_url: String,
    /// Authoritative blob store (Drive on Platform; LocalFs on Solo).
    pub storage: Arc<dyn StorageBackend>,
    pub license_ok: bool,
    pub sessions: Arc<Mutex<SessionStore>>,
    pub hub: SyncHub,
    pub presence: PresenceHub,
    pub jwt_secret: Vec<u8>,
}

#[derive(Default)]
pub struct SessionStore {
    pub sheets: HashMap<String, EratSheet>,
    pub logs: HashMap<String, OpLog>,
    pub tenants: HashMap<String, String>,
    /// Monotonic generation per sheet; used to coalesce Drive flushes.
    pub flush_gens: HashMap<String, u64>,
}

#[derive(Serialize)]
pub struct Health {
    pub status: &'static str,
    pub service: &'static str,
}

#[derive(Deserialize)]
pub struct CreateReq {
    pub tenant_id: Option<String>,
    pub user_id: Option<String>,
    pub name: Option<String>,
}

#[derive(Serialize)]
pub struct CreateResp {
    pub drive_object_id: String,
}

#[derive(Deserialize)]
pub struct ImportReq {
    pub tenant_id: Option<String>,
    pub user_id: Option<String>,
    pub name: Option<String>,
    pub xlsx_base64: String,
}

#[derive(Deserialize)]
pub struct ImportOdsReq {
    pub tenant_id: Option<String>,
    pub user_id: Option<String>,
    pub name: Option<String>,
    pub ods_base64: String,
}

pub fn router(state: AppState) -> Router {
    Router::new()
        .route("/healthz", get(healthz))
        .route("/api/v1/tables", post(create_table))
        .route("/api/v1/tables/import", post(import_table))
        .route("/api/v1/tables/import-ods", post(import_table_ods))
        .route("/api/v1/tables/:id", get(get_table))
        .route("/api/v1/tables/:id/export/xlsx", post(export_table))
        .route("/api/v1/tables/:id/export/ods", post(export_table_ods))
        .route("/api/v1/tables/:id/sync", get(sync_ws))
        .with_state(state)
}

async fn healthz() -> Json<Health> {
    Json(Health {
        status: "ok",
        service: "era-tables-engine",
    })
}

fn require_principal(state: &AppState, headers: &HeaderMap) -> Result<auth::Principal, StatusCode> {
    auth::principal_from_headers(headers, &state.jwt_secret)
}

async fn create_table(
    State(state): State<AppState>,
    headers: HeaderMap,
    Json(req): Json<CreateReq>,
) -> Result<Json<CreateResp>, StatusCode> {
    if !state.license_ok {
        return Err(StatusCode::FORBIDDEN);
    }
    let p = require_principal(&state, &headers)?;
    let name = req.name.unwrap_or_else(|| "sheet.erat".into());
    let mut sheet = EratSheet::empty();
    sheet.name = name.clone();
    recalc(&mut sheet);
    let json = serde_json::to_vec(&sheet).map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    let id = state
        .storage
        .put(&p.tenant_id, &p.user_id, &name, &json, None)
        .await
        .map_err(|_| StatusCode::BAD_GATEWAY)?
        .id;
    sheet.drive_object_id = id.clone();
    sheet.tenant_id = p.tenant_id.clone();
    {
        let mut store = state.sessions.lock().await;
        store.tenants.insert(id.clone(), p.tenant_id);
        store.sheets.insert(id.clone(), sheet);
        store.logs.insert(id.clone(), OpLog::default());
    }
    Ok(Json(CreateResp {
        drive_object_id: id,
    }))
}

async fn import_table(
    State(state): State<AppState>,
    headers: HeaderMap,
    Json(req): Json<ImportReq>,
) -> Result<Json<CreateResp>, StatusCode> {
    if !state.license_ok {
        return Err(StatusCode::FORBIDDEN);
    }
    let p = require_principal(&state, &headers)?;
    use base64::Engine;
    let data = base64::engine::general_purpose::STANDARD
        .decode(req.xlsx_base64.as_bytes())
        .map_err(|_| StatusCode::BAD_REQUEST)?;
    let mut sheet = import_xlsx(&data).map_err(|_| StatusCode::BAD_REQUEST)?;
    recalc(&mut sheet);
    let name = req
        .name
        .unwrap_or_else(|| "import.erat".into());
    sheet.name = name.clone();
    let json = serde_json::to_vec(&sheet).map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    let id = state
        .storage
        .put(&p.tenant_id, &p.user_id, &name, &json, None)
        .await
        .map_err(|_| StatusCode::BAD_GATEWAY)?
        .id;
    sheet.drive_object_id = id.clone();
    sheet.tenant_id = p.tenant_id.clone();
    {
        let mut store = state.sessions.lock().await;
        store.tenants.insert(id.clone(), p.tenant_id);
        store.sheets.insert(id.clone(), sheet);
        store.logs.insert(id.clone(), OpLog::default());
    }
    Ok(Json(CreateResp {
        drive_object_id: id,
    }))
}

async fn import_table_ods(
    State(state): State<AppState>,
    headers: HeaderMap,
    Json(req): Json<ImportOdsReq>,
) -> Result<Json<CreateResp>, StatusCode> {
    if !state.license_ok {
        return Err(StatusCode::FORBIDDEN);
    }
    let p = require_principal(&state, &headers)?;
    use base64::Engine;
    let data = base64::engine::general_purpose::STANDARD
        .decode(req.ods_base64.as_bytes())
        .map_err(|_| StatusCode::BAD_REQUEST)?;
    let mut sheet = import_ods(&data).map_err(|_| StatusCode::BAD_REQUEST)?;
    recalc(&mut sheet);
    let name = req.name.unwrap_or_else(|| "import.erat".into());
    sheet.name = name.clone();
    let json = serde_json::to_vec(&sheet).map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    let id = state
        .storage
        .put(&p.tenant_id, &p.user_id, &name, &json, None)
        .await
        .map_err(|_| StatusCode::BAD_GATEWAY)?
        .id;
    sheet.drive_object_id = id.clone();
    sheet.tenant_id = p.tenant_id.clone();
    {
        let mut store = state.sessions.lock().await;
        store.tenants.insert(id.clone(), p.tenant_id);
        store.sheets.insert(id.clone(), sheet);
        store.logs.insert(id.clone(), OpLog::default());
    }
    Ok(Json(CreateResp {
        drive_object_id: id,
    }))
}

async fn export_table(
    State(state): State<AppState>,
    headers: HeaderMap,
    Path(id): Path<String>,
) -> Result<impl IntoResponse, StatusCode> {
    if !state.license_ok {
        return Err(StatusCode::FORBIDDEN);
    }
    let p = require_principal(&state, &headers)?;
    let sheet = {
        let store = state.sessions.lock().await;
        if let Some(tid) = store.tenants.get(&id) {
            if tid != &p.tenant_id {
                return Err(StatusCode::FORBIDDEN);
            }
        }
        store.sheets.get(&id).cloned()
    };
    let sheet = match sheet {
        Some(s) => s,
        None => {
            let json = state
                .storage
                .get_json(&p.tenant_id, &p.user_id, &id)
                .await
                .map_err(|_| StatusCode::NOT_FOUND)?;
            let mut s: EratSheet =
                serde_json::from_str(&json).map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
            recalc(&mut s);
            s
        }
    };
    let bytes = export_xlsx(&sheet).map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    Ok((
        [
            (
                axum::http::header::CONTENT_TYPE,
                "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
            ),
            (
                axum::http::header::CONTENT_DISPOSITION,
                "attachment; filename=\"export.xlsx\"",
            ),
        ],
        bytes,
    ))
}

async fn export_table_ods(
    State(state): State<AppState>,
    headers: HeaderMap,
    Path(id): Path<String>,
) -> Result<impl IntoResponse, StatusCode> {
    if !state.license_ok {
        return Err(StatusCode::FORBIDDEN);
    }
    let p = require_principal(&state, &headers)?;
    let sheet = {
        let store = state.sessions.lock().await;
        if let Some(tid) = store.tenants.get(&id) {
            if tid != &p.tenant_id {
                return Err(StatusCode::FORBIDDEN);
            }
        }
        store.sheets.get(&id).cloned()
    };
    let sheet = match sheet {
        Some(s) => s,
        None => {
            let json = state
                .storage
                .get_json(&p.tenant_id, &p.user_id, &id)
                .await
                .map_err(|_| StatusCode::NOT_FOUND)?;
            let mut s: EratSheet =
                serde_json::from_str(&json).map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
            recalc(&mut s);
            s
        }
    };
    let bytes = export_ods(&sheet).map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    Ok((
        [
            (
                axum::http::header::CONTENT_TYPE,
                "application/vnd.oasis.opendocument.spreadsheet",
            ),
            (
                axum::http::header::CONTENT_DISPOSITION,
                "attachment; filename=\"export.ods\"",
            ),
        ],
        bytes,
    ))
}

async fn get_table(
    State(state): State<AppState>,
    headers: HeaderMap,
    Path(id): Path<String>,
) -> Result<Json<EratSheet>, StatusCode> {
    if !state.license_ok {
        return Err(StatusCode::FORBIDDEN);
    }
    let p = require_principal(&state, &headers)?;
    {
        let mut store = state.sessions.lock().await;
        if let Some(sheet) = store.sheets.get_mut(&id) {
            sheet.normalize_tabs();
        }
        if let Some(tid) = store.tenants.get(&id) {
            if tid != &p.tenant_id {
                return Err(StatusCode::FORBIDDEN);
            }
        }
        if let Some(sheet) = store.sheets.get(&id) {
            return Ok(Json(sheet.clone()));
        }
    }
    // AC-T1 reopen: load from Drive when session miss
    let json = state
        .storage
        .get_json(&p.tenant_id, &p.user_id, &id)
        .await
        .map_err(|_| StatusCode::NOT_FOUND)?;
    let mut sheet: EratSheet =
        serde_json::from_str(&json).map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    sheet.drive_object_id = id.clone();
    sheet.tenant_id = p.tenant_id.clone();
    sheet.normalize_tabs();
    recalc(&mut sheet);
    sheet.flush_active_to_tab();
    {
        let mut store = state.sessions.lock().await;
        store.tenants.insert(id.clone(), p.tenant_id);
        store.sheets.insert(id.clone(), sheet.clone());
        store.logs.entry(id).or_default();
    }
    Ok(Json(sheet))
}

async fn sync_ws(
    ws: WebSocketUpgrade,
    State(state): State<AppState>,
    headers: HeaderMap,
    Path(id): Path<String>,
    axum::extract::Query(q): axum::extract::Query<std::collections::HashMap<String, String>>,
) -> impl IntoResponse {
    if !state.license_ok {
        return StatusCode::FORBIDDEN.into_response();
    }
    let query = q.get("access_token").map(|t| format!("access_token={t}"));
    let principal = match auth::principal_from_ws(&headers, query.as_deref(), &state.jwt_secret) {
        Ok(p) => p,
        Err(code) => return code.into_response(),
    };
    {
        let store = state.sessions.lock().await;
        if let Some(tid) = store.tenants.get(&id) {
            if tid != &principal.tenant_id {
                return StatusCode::FORBIDDEN.into_response();
            }
        }
    }
    let user_id = principal.user_id.clone();
    let tenant_id = principal.tenant_id.clone();
    let conn_id = uuid::Uuid::new_v4().to_string();
    ws.on_upgrade(move |socket| handle_sync(socket, state, id, conn_id, user_id, tenant_id))
}

/// Schedule a debounced `put_erat` of the in-memory session sheet (AC-T5).
/// Multiple ops within [`DRIVE_FLUSH_DEBOUNCE`] coalesce to one Drive write.
fn schedule_drive_flush(state: AppState, id: String, tenant_id: String, user_id: String) {
    tokio::spawn(async move {
        let gen = {
            let mut store = state.sessions.lock().await;
            let e = store.flush_gens.entry(id.clone()).or_insert(0);
            *e = e.saturating_add(1);
            *e
        };
        tokio::time::sleep(drive_flush_debounce()).await;
        let sheet = {
            let store = state.sessions.lock().await;
            if store.flush_gens.get(&id).copied() != Some(gen) {
                return; // newer edit scheduled a later flush
            }
            store.sheets.get(&id).cloned()
        };
        let Some(sheet) = sheet else {
            return;
        };
        let name = if sheet.name.is_empty() {
            format!("{id}.erat")
        } else {
            sheet.name.clone()
        };
        let json = match serde_json::to_vec(&sheet) {
            Ok(v) => v,
            Err(e) => {
                tracing::warn!(sheet_id = %id, error = %e, "drive flush serialize failed");
                return;
            }
        };
        if let Err(e) = state
            .storage
            .put(&tenant_id, &user_id, &name, &json, Some(&id))
            .await
        {
            tracing::warn!(sheet_id = %id, error = %e, "drive flush put failed");
        }
    });
}

async fn handle_sync(
    socket: WebSocket,
    state: AppState,
    id: String,
    conn_id: String,
    user_id: String,
    tenant_id: String,
) {
    let mut room_rx = state.hub.subscribe(&id).await;
    let (mut sink, mut stream) = socket.split();
    let (out_tx, mut out_rx) = mpsc::unbounded_channel::<Message>();
    let write = tokio::spawn(async move {
        while let Some(msg) = out_rx.recv().await {
            if sink.send(msg).await.is_err() {
                break;
            }
        }
    });
    let out_fan = out_tx.clone();
    let my = conn_id.clone();
    let fan = tokio::spawn(async move {
        loop {
            match room_rx.recv().await {
                Ok(payload) => {
                    if let Ok(env) = serde_json::from_str::<SyncEnvelope>(&payload) {
                        if env.author_id == my {
                            continue;
                        }
                    }
                    if out_fan.send(Message::Text(payload.into())).is_err() {
                        break;
                    }
                }
                Err(broadcast::error::RecvError::Closed) => break,
                Err(broadcast::error::RecvError::Lagged(_)) => continue,
            }
        }
    });

    let peers = state.presence.join(&id, &conn_id, &user_id).await;
    if let Ok(roster) = serde_json::to_string(&serde_json::json!({
        "type": "presence",
        "peers": peers,
    })) {
        let _ = out_tx.send(Message::Text(roster.clone().into()));
        state.hub.publish(&id, roster).await;
    }

    while let Some(Ok(msg)) = stream.next().await {
        let Message::Text(text) = msg else { continue };
        if let Ok(v) = serde_json::from_str::<serde_json::Value>(&text) {
            if let Some(t) = v.get("type").and_then(|t| t.as_str()) {
                if t == "presence_heartbeat" || t == "presence_ping" {
                    continue;
                }
                if t == "presence_cell" {
                    let mut out = v;
                    if out.get("from").is_none() {
                        out["from"] = serde_json::json!(user_id);
                    }
                    if out.get("user_id").is_none() {
                        out["user_id"] = serde_json::json!(user_id);
                    }
                    if let Ok(payload) = serde_json::to_string(&out) {
                        state.hub.publish(&id, payload).await;
                    }
                    continue;
                }
            }
        }
        let Ok(op) = serde_json::from_str::<SheetOp>(&text) else {
            continue;
        };
        let applied = {
            let mut store = state.sessions.lock().await;
            let Some(sheet) = store.sheets.get_mut(&id) else {
                continue;
            };
            if blocked_by_protect(sheet, &op) {
                let _ = out_tx.send(Message::Text(
                    r#"{"type":"error","code":"protected","message":"Sheet is protected"}"#.into(),
                ));
                None
            } else {
                apply_op(sheet, &op);
                let cells = sheet.cells.clone();
                let active_sheet = sheet.active_sheet;
                let sheet_names: Vec<String> = sheet.sheets.iter().map(|t| t.name.clone()).collect();
                let protected = sheet.is_active_protected();
                let (freeze_rows, freeze_cols) = sheet.active_freeze();
                let Some(log) = store.logs.get_mut(&id) else {
                    continue;
                };
                log.append(op.clone());
                Some((
                    log.version,
                    cells,
                    active_sheet,
                    sheet_names,
                    protected,
                    freeze_rows,
                    freeze_cols,
                    op,
                ))
            }
        };
        let Some((version, cells, active_sheet, sheet_names, protected, freeze_rows, freeze_cols, op)) =
            applied
        else {
            continue;
        };
        let _ = out_tx.send(Message::Text("ack".into()));
        if let Ok(payload) = serde_json::to_string(&SyncEnvelope {
            op,
            version,
            author_id: conn_id.clone(),
            cells: Some(cells),
            active_sheet: Some(active_sheet),
            sheet_names: Some(sheet_names),
            protected: Some(protected),
            freeze_rows: Some(freeze_rows),
            freeze_cols: Some(freeze_cols),
        }) {
            state.hub.publish(&id, payload).await;
        }
        schedule_drive_flush(
            state.clone(),
            id.clone(),
            tenant_id.clone(),
            user_id.clone(),
        );
    }
    let peers = state.presence.leave(&id, &conn_id).await;
    if let Ok(roster) = serde_json::to_string(&serde_json::json!({
        "type": "presence",
        "peers": peers,
    })) {
        state.hub.publish(&id, roster).await;
    }
    drop(out_tx);
    let _ = write.await;
    fan.abort();
}

pub async fn serve(addr: SocketAddr, state: AppState) -> anyhow::Result<()> {
    let listener = TcpListener::bind(addr).await?;
    axum::serve(listener, router(state)).await?;
    Ok(())
}

pub fn license_from_env() -> bool {
    if production_like() {
        return std::env::var("ERA_LICENSE_OFFICE_TABLES")
            .map(|v| v == "1" || v == "true")
            .unwrap_or(false);
    }
    if env_truthy("ERA_OFFICE_DEV") {
        return true;
    }
    std::env::var("ERA_LICENSE_OFFICE_TABLES")
        .map(|v| v == "1" || v == "true")
        .unwrap_or(false)
}

fn production_like() -> bool {
    env_truthy("ERA_LICENSE_STRICT")
        || env_truthy("ERA_PRODUCTION")
        || env_truthy("ERA_ENV_PRODUCTION")
        || std::env::var("ERA_ENV")
            .map(|v| v.eq_ignore_ascii_case("production"))
            .unwrap_or(false)
}

fn env_truthy(k: &str) -> bool {
    matches!(
        std::env::var(k).ok().as_deref(),
        Some("1") | Some("true") | Some("yes")
    )
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::drive_bind::platform_storage;

    fn mint(sub: &str, tenant: &str, exp: i64) -> String {
        let claims = auth::Claims {
            sub: sub.into(),
            tenant_id: tenant.into(),
            email: format!("{sub}@x"),
            exp,
        };
        jsonwebtoken::encode(
            &jsonwebtoken::Header::default(),
            &claims,
            &jsonwebtoken::EncodingKey::from_secret(b"s"),
        )
        .unwrap()
    }

    async fn spawn(state: AppState) -> std::net::SocketAddr {
        let app = router(state);
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let addr = listener.local_addr().unwrap();
        tokio::spawn(async move {
            axum::serve(listener, app).await.unwrap();
        });
        addr
    }

    #[test]
    fn license_denied_when_era_env_production_ignores_office_dev() {
        std::env::remove_var("ERA_PRODUCTION");
        std::env::remove_var("ERA_LICENSE_STRICT");
        std::env::set_var("ERA_ENV", "production");
        std::env::set_var("ERA_OFFICE_DEV", "1");
        std::env::remove_var("ERA_LICENSE_OFFICE_TABLES");
        assert!(!license_from_env());
        std::env::remove_var("ERA_ENV");
        std::env::remove_var("ERA_OFFICE_DEV");
    }

    #[tokio::test]
    async fn tables_create_without_jwt_401() {
        let addr = spawn(AppState {
            drive_url: "http://127.0.0.1:1".into(),
            storage: platform_storage("http://127.0.0.1:1"),
            license_ok: true,
            sessions: Arc::new(Mutex::new(SessionStore::default())),
            hub: SyncHub::default(),
            presence: PresenceHub::default(),
            jwt_secret: b"s".to_vec(),
        })
        .await;
        let resp = reqwest::Client::new()
            .post(format!("http://{addr}/api/v1/tables"))
            .json(&serde_json::json!({}))
            .send()
            .await
            .unwrap();
        assert_eq!(resp.status(), StatusCode::UNAUTHORIZED);
    }

    #[tokio::test]
    async fn tables_create_with_expired_jwt_401() {
        let addr = spawn(AppState {
            drive_url: "http://127.0.0.1:1".into(),
            storage: platform_storage("http://127.0.0.1:1"),
            license_ok: true,
            sessions: Arc::new(Mutex::new(SessionStore::default())),
            hub: SyncHub::default(),
            presence: PresenceHub::default(),
            jwt_secret: b"s".to_vec(),
        })
        .await;
        let tok = mint("u", "t", 1_600_000_000);
        let resp = reqwest::Client::new()
            .post(format!("http://{addr}/api/v1/tables"))
            .header("Authorization", format!("Bearer {tok}"))
            .json(&serde_json::json!({}))
            .send()
            .await
            .unwrap();
        assert_eq!(resp.status(), StatusCode::UNAUTHORIZED);
    }

    #[tokio::test]
    async fn tables_create_without_license_403() {
        let addr = spawn(AppState {
            drive_url: "http://127.0.0.1:1".into(),
            storage: platform_storage("http://127.0.0.1:1"),
            license_ok: false,
            sessions: Arc::new(Mutex::new(SessionStore::default())),
            hub: SyncHub::default(),
            presence: PresenceHub::default(),
            jwt_secret: b"s".to_vec(),
        })
        .await;
        let tok = mint("u", "t", 4102444800);
        let resp = reqwest::Client::new()
            .post(format!("http://{addr}/api/v1/tables"))
            .header("Authorization", format!("Bearer {tok}"))
            .json(&serde_json::json!({}))
            .send()
            .await
            .unwrap();
        assert_eq!(resp.status(), StatusCode::FORBIDDEN);
    }

    #[tokio::test]
    async fn get_table_rejects_cross_tenant() {
        let state = AppState {
            drive_url: "http://127.0.0.1:1".into(),
            storage: platform_storage("http://127.0.0.1:1"),
            license_ok: true,
            sessions: Arc::new(Mutex::new(SessionStore::default())),
            hub: SyncHub::default(),
            presence: PresenceHub::default(),
            jwt_secret: b"s".to_vec(),
        };
        {
            let mut store = state.sessions.lock().await;
            store.tenants.insert("sheet-b".into(), "tenant-b".into());
            store.sheets.insert("sheet-b".into(), EratSheet::empty());
            store.logs.insert("sheet-b".into(), OpLog::default());
        }
        let addr = spawn(state).await;
        let tok_a = mint("u-a", "tenant-a", 4102444800);
        let resp = reqwest::Client::new()
            .get(format!("http://{addr}/api/v1/tables/sheet-b"))
            .header("Authorization", format!("Bearer {tok_a}"))
            .send()
            .await
            .unwrap();
        assert_eq!(resp.status(), StatusCode::FORBIDDEN);
    }
}
