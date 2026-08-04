use std::collections::HashMap;
use std::net::SocketAddr;
use std::sync::Arc;

use axum::extract::ws::{Message, WebSocket, WebSocketUpgrade};
use axum::extract::{Path, Query, State};
use axum::http::{HeaderMap, StatusCode};
use axum::response::IntoResponse;
use axum::routing::{get, post};
use axum::{Json, Router};
use futures_util::{SinkExt, StreamExt};
use serde::{Deserialize, Serialize};
use tokio::net::TcpListener;
use tokio::sync::{broadcast, mpsc, Mutex};
use tracing::info;
use uuid::Uuid;

use crate::canonical::to_canonical_json;
use crate::convert::{export_docx, export_odt, export_rtf, import_docx};
use crate::model::EradDocument;
use crate::persist::PgPersist;
use crate::sync::{apply_op, DocOp, OpLog};
use era_office_storage::StorageBackend;

const SYNC_ROOM_CAP: usize = 64;

/// Per-document fan-out of applied ops to live WebSocket peers (AC-O1).
#[derive(Clone, Default)]
pub struct SyncHub {
    rooms: Arc<Mutex<HashMap<String, broadcast::Sender<String>>>>,
}

impl SyncHub {
    pub async fn subscribe(&self, object_id: &str) -> broadcast::Receiver<String> {
        let mut rooms = self.rooms.lock().await;
        let tx = rooms.entry(object_id.to_string()).or_insert_with(|| {
            let (tx, _) = broadcast::channel(SYNC_ROOM_CAP);
            tx
        });
        tx.subscribe()
    }

    pub async fn publish(&self, object_id: &str, payload: String) {
        let rooms = self.rooms.lock().await;
        if let Some(tx) = rooms.get(object_id) {
            let _ = tx.send(payload);
        }
    }
}

/// Live peer roster per document (`conn_id` → `user_id`; roster = unique user ids).
#[derive(Clone, Default)]
pub struct PresenceHub {
    rooms: Arc<Mutex<HashMap<String, HashMap<String, String>>>>,
}

impl PresenceHub {
    fn peer_roster(peers: &HashMap<String, String>) -> Vec<String> {
        let mut ids: Vec<String> = peers.values().cloned().collect();
        ids.sort();
        ids.dedup();
        ids
    }

    pub async fn join(&self, doc_id: &str, conn_id: &str, user_id: &str) -> Vec<String> {
        let mut rooms = self.rooms.lock().await;
        let peers = rooms
            .entry(doc_id.to_string())
            .or_default()
            .entry(conn_id.to_string())
            .or_insert_with(|| user_id.to_string());
        *peers = user_id.to_string();
        Self::peer_roster(rooms.get(doc_id).unwrap())
    }

    pub async fn leave(&self, doc_id: &str, conn_id: &str) -> Vec<String> {
        let mut rooms = self.rooms.lock().await;
        if let Some(peers) = rooms.get_mut(doc_id) {
            peers.remove(conn_id);
            if peers.is_empty() {
                rooms.remove(doc_id);
                return Vec::new();
            }
            return Self::peer_roster(peers);
        }
        Vec::new()
    }
}

fn presence_payload(peers: &[String]) -> String {
    serde_json::json!({ "type": "presence", "peers": peers }).to_string()
}

/// Wire envelope broadcast to peers (UI also accepts bare DocOp JSON).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SyncEnvelope {
    pub op: DocOp,
    pub version: u64,
    pub author_id: String,
}

#[derive(Clone)]
pub struct AppState {
    pub drive_url: String,
    /// Authoritative blob store (Drive on Platform; LocalFs on Solo).
    pub storage: Arc<dyn StorageBackend>,
    pub license_ok: bool,
    pub sessions: Arc<Mutex<SessionStore>>,
    pub pg: Option<Arc<PgPersist>>,
    pub hub: SyncHub,
    pub presence: PresenceHub,
    pub jwt_secret: Vec<u8>,
    /// HMAC secret for Comms EditLink intents (`ERA_DOCS_INTENT_SECRET`).
    pub intent_secret: Vec<u8>,
}

#[derive(Default)]
pub struct SessionStore {
    pub logs: HashMap<String, OpLog>,
    pub docs: HashMap<String, EradDocument>,
    pub tenants: HashMap<String, String>,
}

#[derive(Serialize)]
pub struct Health {
    pub status: &'static str,
    pub service: &'static str,
    pub persistence: &'static str,
}

#[derive(Serialize)]
pub struct CreateDocResponse {
    pub drive_object_id: String,
}

#[derive(Deserialize)]
pub struct CreateDocRequest {
    pub tenant_id: Option<String>,
    pub user_id: Option<String>,
    pub name: Option<String>,
    pub folder_id: Option<String>,
}

pub fn router(state: AppState) -> Router {
    Router::new()
        .route("/healthz", get(healthz))
        .route("/api/v1/docs", post(create_doc))
        .route("/api/v1/docs/import", post(import_doc))
        .route("/api/v1/docs/:id", get(get_doc))
        .route("/api/v1/docs/:id/verify-intent", get(verify_intent_ep))
        .route("/api/v1/docs/:id/export/docx", post(export_doc))
        .route("/api/v1/docs/:id/export/rtf", post(export_rtf_doc))
        .route("/api/v1/docs/:id/export/odt", post(export_odt_doc))
        .route("/api/v1/docs/:id/snapshot", post(snapshot))
        .route("/api/v1/docs/:id/sync", get(sync_ws))
        .with_state(state)
}

async fn healthz(State(state): State<AppState>) -> Json<Health> {
    let persistence = if state.pg.is_some() {
        "postgres"
    } else {
        "memory"
    };
    Json(Health {
        status: "ok",
        service: "era-docs-engine",
        persistence,
    })
}

fn check_license(state: &AppState) -> Result<(), StatusCode> {
    if state.license_ok {
        Ok(())
    } else {
        Err(StatusCode::FORBIDDEN)
    }
}

fn require_principal(
    state: &AppState,
    headers: &HeaderMap,
) -> Result<crate::auth::Principal, StatusCode> {
    crate::auth::principal_from_headers(headers, &state.jwt_secret)
}

impl AppState {
    async fn init_session(
        &self,
        tenant_id: &str,
        object_id: &str,
        doc: EradDocument,
    ) -> Result<(), StatusCode> {
        {
            let mut store = self.sessions.lock().await;
            store.tenants.insert(object_id.to_string(), tenant_id.to_string());
            store.docs.insert(object_id.to_string(), doc);
            store.logs.insert(object_id.to_string(), OpLog::default());
        }
        if let Some(pg) = &self.pg {
            pg.ensure_session(tenant_id, object_id)
                .await
                .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
        }
        Ok(())
    }

    /// Load document; JWT `principal_tenant` must match session/object tenant.
    async fn document_for(
        &self,
        object_id: &str,
        user_id: &str,
        principal_tenant: &str,
    ) -> Result<EradDocument, StatusCode> {
        {
            let store = self.sessions.lock().await;
            if let Some(tid) = store.tenants.get(object_id) {
                if tid != principal_tenant {
                    return Err(StatusCode::FORBIDDEN);
                }
            }
            if let Some(doc) = store.docs.get(object_id) {
                return Ok(doc.clone());
            }
        }

        let tenant_id = {
            let store = self.sessions.lock().await;
            store.tenants.get(object_id).cloned()
        };
        let tenant_id = match tenant_id {
            Some(t) => t,
            None => {
                if let Some(pg) = &self.pg {
                    let t = pg
                        .tenant_for_object(object_id)
                        .await
                        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?
                        .ok_or(StatusCode::NOT_FOUND)?;
                    if t != principal_tenant {
                        return Err(StatusCode::FORBIDDEN);
                    }
                    t
                } else {
                    // AC-O3 reopen: no session/PG — trust JWT tenant; Drive enforces ACL.
                    principal_tenant.to_string()
                }
            }
        };
        if tenant_id != principal_tenant {
            return Err(StatusCode::FORBIDDEN);
        }

        let json = self
            .storage
            .get_json(&tenant_id, user_id, object_id)
            .await
            .map_err(|_| StatusCode::NOT_FOUND)?;
        let base: EradDocument =
            serde_json::from_str(&json).map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;

        let doc = if let Some(pg) = &self.pg {
            let log = pg
                .load_log(&tenant_id, object_id)
                .await
                .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
            let merged = log.replay(base);
            let mut store = self.sessions.lock().await;
            store.tenants.insert(object_id.to_string(), tenant_id);
            store.docs.insert(object_id.to_string(), merged.clone());
            store.logs.insert(object_id.to_string(), log);
            merged
        } else {
            let mut store = self.sessions.lock().await;
            store.tenants.insert(object_id.to_string(), tenant_id);
            store.docs.insert(object_id.to_string(), base.clone());
            store.logs.entry(object_id.to_string()).or_default();
            base
        };
        Ok(doc)
    }

    /// Apply op with insert-OT against the last applied peer op (WS hot path).
    async fn apply_sync_op(
        &self,
        object_id: &str,
        op: DocOp,
    ) -> Result<(u64, DocOp), StatusCode> {
        let (tenant_id, version, transformed) = {
            let mut store = self.sessions.lock().await;
            let transformed = if let Some(log) = store.logs.get(object_id) {
                let mut t = op;
                if let Some(prev) = log.ops.last() {
                    t = t.transform_against(prev);
                }
                t
            } else {
                op
            };
            if let Some(doc) = store.docs.get_mut(object_id) {
                apply_op(doc, &transformed);
            }
            let version = if let Some(log) = store.logs.get_mut(object_id) {
                log.append(transformed.clone());
                log.version
            } else {
                return Err(StatusCode::NOT_FOUND);
            };
            (
                store.tenants.get(object_id).cloned(),
                version,
                transformed,
            )
        };
        if let (Some(pg), Some(tenant_id)) = (&self.pg, tenant_id) {
            let log = {
                let store = self.sessions.lock().await;
                store.logs.get(object_id).cloned()
            };
            if let Some(log) = log {
                pg.save_log(&tenant_id, object_id, &log)
                    .await
                    .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
            }
        }
        Ok((version, transformed))
    }
}

async fn create_doc(
    State(state): State<AppState>,
    headers: HeaderMap,
    Json(req): Json<CreateDocRequest>,
) -> Result<Json<CreateDocResponse>, StatusCode> {
    check_license(&state)?;
    let p = require_principal(&state, &headers)?;
    let name = req.name.unwrap_or_else(|| "document.erad".to_string());
    let doc = EradDocument::empty();
    let json = to_canonical_json(&doc).map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    let id = state
        .storage
        .put(&p.tenant_id, &p.user_id, &name, json.as_bytes(), None)
        .await
        .map_err(drive_put_status)?
        .id;
    state.init_session(&p.tenant_id, &id, doc).await?;
    Ok(Json(CreateDocResponse {
        drive_object_id: id,
    }))
}

#[derive(Deserialize)]
struct ImportBody {
    #[allow(dead_code)]
    tenant_id: Option<String>,
    #[allow(dead_code)]
    user_id: Option<String>,
    /// Original filename (e.g. `memo.docx`); used to build a unique Drive `.erad` name.
    name: Option<String>,
    docx_base64: String,
}

fn unique_import_erad_name(original: Option<&str>) -> String {
    let raw = original.unwrap_or("import.docx").trim();
    let stem = std::path::Path::new(raw)
        .file_stem()
        .and_then(|s| s.to_str())
        .filter(|s| !s.is_empty())
        .unwrap_or("import");
    // Drive forbids duplicate names in a folder — never reuse a fixed "import.erad".
    format!(
        "{}-{}.erad",
        stem.chars().take(64).collect::<String>(),
        &uuid::Uuid::new_v4().to_string()[..8]
    )
}

fn drive_put_status(err: anyhow::Error) -> StatusCode {
    let msg = format!("{err:#}");
    if msg.contains("409") || msg.to_ascii_lowercase().contains("duplicate") {
        StatusCode::CONFLICT
    } else {
        StatusCode::BAD_GATEWAY
    }
}

async fn import_doc(
    State(state): State<AppState>,
    headers: HeaderMap,
    Json(body): Json<ImportBody>,
) -> Result<Json<CreateDocResponse>, StatusCode> {
    check_license(&state)?;
    let p = require_principal(&state, &headers)?;
    use base64::Engine;
    let data = base64::engine::general_purpose::STANDARD
        .decode(body.docx_base64.as_bytes())
        .map_err(|_| StatusCode::BAD_REQUEST)?;
    let doc = import_docx(&data).map_err(|_| StatusCode::BAD_REQUEST)?;
    let name = unique_import_erad_name(body.name.as_deref());
    let json = to_canonical_json(&doc).map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    let id = state
        .storage
        .put(&p.tenant_id, &p.user_id, &name, json.as_bytes(), None)
        .await
        .map_err(drive_put_status)?
        .id;
    state.init_session(&p.tenant_id, &id, doc).await?;
    Ok(Json(CreateDocResponse {
        drive_object_id: id,
    }))
}

#[derive(Deserialize)]
struct IntentQuery {
    intent_exp: Option<String>,
    intent_sig: Option<String>,
}

async fn verify_intent_ep(
    State(state): State<AppState>,
    headers: HeaderMap,
    Path(id): Path<String>,
    Query(q): Query<IntentQuery>,
) -> Result<StatusCode, StatusCode> {
    check_license(&state)?;
    let _p = require_principal(&state, &headers)?;
    let (Some(exp), Some(sig)) = (q.intent_exp.as_deref(), q.intent_sig.as_deref()) else {
        return Err(StatusCode::UNAUTHORIZED);
    };
    if !crate::intent::verify_intent(&state.intent_secret, &id, exp, sig) {
        return Err(StatusCode::UNAUTHORIZED);
    }
    Ok(StatusCode::NO_CONTENT)
}

async fn get_doc(
    State(state): State<AppState>,
    headers: HeaderMap,
    Path(id): Path<String>,
    Query(q): Query<IntentQuery>,
) -> Result<Json<EradDocument>, StatusCode> {
    check_license(&state)?;
    let p = require_principal(&state, &headers)?;
    // When Comms deep-link intent params are present, verify HMAC before open (AC-O8).
    if q.intent_exp.is_some() || q.intent_sig.is_some() {
        let (Some(exp), Some(sig)) = (q.intent_exp.as_deref(), q.intent_sig.as_deref()) else {
            return Err(StatusCode::UNAUTHORIZED);
        };
        if !crate::intent::verify_intent(&state.intent_secret, &id, exp, sig) {
            return Err(StatusCode::UNAUTHORIZED);
        }
    }
    let doc = state.document_for(&id, &p.user_id, &p.tenant_id).await?;
    Ok(Json(doc))
}

async fn export_doc(
    State(state): State<AppState>,
    headers: HeaderMap,
    Path(id): Path<String>,
) -> Result<(StatusCode, [(axum::http::header::HeaderName, &'static str); 1], Vec<u8>), StatusCode>
{
    check_license(&state)?;
    let p = require_principal(&state, &headers)?;
    let doc = state.document_for(&id, &p.user_id, &p.tenant_id).await?;
    let bytes = export_docx(&doc).map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    Ok((
        StatusCode::OK,
        [(
            axum::http::header::CONTENT_TYPE,
            "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
        )],
        bytes,
    ))
}

async fn export_rtf_doc(
    State(state): State<AppState>,
    headers: HeaderMap,
    Path(id): Path<String>,
) -> Result<(StatusCode, [(axum::http::header::HeaderName, &'static str); 1], Vec<u8>), StatusCode>
{
    check_license(&state)?;
    let p = require_principal(&state, &headers)?;
    let doc = state.document_for(&id, &p.user_id, &p.tenant_id).await?;
    let bytes = export_rtf(&doc);
    Ok((
        StatusCode::OK,
        [(axum::http::header::CONTENT_TYPE, "application/rtf")],
        bytes,
    ))
}

async fn export_odt_doc(
    State(state): State<AppState>,
    headers: HeaderMap,
    Path(id): Path<String>,
) -> Result<(StatusCode, [(axum::http::header::HeaderName, &'static str); 1], Vec<u8>), StatusCode>
{
    check_license(&state)?;
    let p = require_principal(&state, &headers)?;
    let doc = state.document_for(&id, &p.user_id, &p.tenant_id).await?;
    let bytes = export_odt(&doc).map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    Ok((
        StatusCode::OK,
        [(
            axum::http::header::CONTENT_TYPE,
            "application/vnd.oasis.opendocument.text",
        )],
        bytes,
    ))
}

async fn snapshot(
    State(state): State<AppState>,
    headers: HeaderMap,
    Path(id): Path<String>,
    Json(_req): Json<CreateDocRequest>,
) -> Result<StatusCode, StatusCode> {
    check_license(&state)?;
    let p = require_principal(&state, &headers)?;
    let doc = state.document_for(&id, &p.user_id, &p.tenant_id).await?;
    let json = to_canonical_json(&doc).map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    state
        .storage
        .put(
            &p.tenant_id,
            &p.user_id,
            &format!("{id}.erad"),
            json.as_bytes(),
            Some(&id),
        )
        .await
        .map_err(|_| StatusCode::BAD_GATEWAY)?;
    Ok(StatusCode::OK)
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
    let query = q
        .get("access_token")
        .map(|t| format!("access_token={t}"));
    let principal = match crate::auth::principal_from_ws(
        &headers,
        query.as_deref(),
        &state.jwt_secret,
    ) {
        Ok(p) => p,
        Err(code) => return code.into_response(),
    };
    // Tenant isolation: reject if session exists for another tenant.
    {
        let store = state.sessions.lock().await;
        if let Some(tid) = store.tenants.get(&id) {
            if tid != &principal.tenant_id {
                return StatusCode::FORBIDDEN.into_response();
            }
        }
    }
    let conn_id = Uuid::new_v4().to_string();
    let user_id = principal.user_id.clone();
    ws.on_upgrade(move |socket| handle_sync(socket, state, id, conn_id, user_id))
}

async fn handle_sync(
    socket: WebSocket,
    state: AppState,
    id: String,
    conn_id: String,
    user_id: String,
) {
    let peers = state.presence.join(&id, &conn_id, &user_id).await;
    let presence_msg = presence_payload(&peers);
    state.hub.publish(&id, presence_msg.clone()).await;

    let mut room_rx = state.hub.subscribe(&id).await;
    let (mut ws_sink, mut ws_stream) = socket.split();
    let (out_tx, mut out_rx) = mpsc::unbounded_channel::<Message>();

    let _ = out_tx.send(Message::Text(presence_msg.into()));

    let write_task = tokio::spawn(async move {
        while let Some(msg) = out_rx.recv().await {
            if ws_sink.send(msg).await.is_err() {
                break;
            }
        }
    });

    let out_fan = out_tx.clone();
    let my_conn = conn_id.clone();
    let fan_task = tokio::spawn(async move {
        loop {
            match room_rx.recv().await {
                Ok(payload) => {
                    if let Ok(env) = serde_json::from_str::<SyncEnvelope>(&payload) {
                        if env.author_id == my_conn {
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

    while let Some(Ok(msg)) = ws_stream.next().await {
        let Message::Text(text) = msg else {
            continue;
        };
        if let Ok(v) = serde_json::from_str::<serde_json::Value>(&text) {
            if let Some(t) = v.get("type").and_then(|t| t.as_str()) {
                if t == "presence_heartbeat" || t == "presence_ping" {
                    continue;
                }
                // Shared suggesting revisions (accept/reject fan-out).
                if t == "revision_event" {
                    let mut out = v;
                    if out.get("from").is_none() {
                        out["from"] = serde_json::json!(user_id);
                    }
                    if let Ok(payload) = serde_json::to_string(&out) {
                        state.hub.publish(&id, payload).await;
                    }
                    continue;
                }
                // Relay peer carets / presence_caret envelopes (not DocOp).
                if t == "presence_caret"
                    || (t == "presence" && v.get("caret").is_some())
                {
                    let mut out = v;
                    if out.get("from").is_none() {
                        out["from"] = serde_json::json!(user_id);
                    }
                    if out.get("user_id").is_none() {
                        out["user_id"] = serde_json::json!(user_id);
                    }
                    out["type"] = serde_json::json!("presence_caret");
                    if let Ok(payload) = serde_json::to_string(&out) {
                        state.hub.publish(&id, payload).await;
                    }
                    continue;
                }
            }
        }
        let Ok(op) = serde_json::from_str::<DocOp>(&text) else {
            continue;
        };
        let Ok((version, applied)) = state.apply_sync_op(&id, op).await else {
            continue;
        };
        let _ = out_tx.send(Message::Text("ack".into()));
        let envelope = SyncEnvelope {
            op: applied,
            version,
            author_id: conn_id.clone(),
        };
        if let Ok(payload) = serde_json::to_string(&envelope) {
            state.hub.publish(&id, payload).await;
        }
    }

    let peers = state.presence.leave(&id, &conn_id).await;
    state
        .hub
        .publish(&id, presence_payload(&peers))
        .await;

    drop(out_tx);
    let _ = write_task.await;
    fan_task.abort();
}

pub async fn serve(addr: SocketAddr, state: AppState) -> anyhow::Result<()> {
    let app = router(state);
    let listener = TcpListener::bind(addr).await?;
    info!("era-docs-engine listening {}", addr);
    axum::serve(listener, app).await?;
    Ok(())
}

/// License gate for office-documents module.
///
/// Fail-closed when production-like (`ERA_LICENSE_STRICT` / `ERA_PRODUCTION` /
/// `ERA_ENV=production` / `ERA_ENV_PRODUCTION`) — ignores `ERA_OFFICE_DEV`.
/// Lab: set `ERA_OFFICE_DEV=1` **without** those, or set
/// `ERA_LICENSE_OFFICE_DOCUMENTS=1`.
pub fn license_from_env() -> bool {
    if production_like() {
        return std::env::var("ERA_LICENSE_OFFICE_DOCUMENTS")
            .map(|v| v == "1" || v == "true")
            .unwrap_or(false);
    }
    if env_truthy("ERA_OFFICE_DEV") {
        return true;
    }
    std::env::var("ERA_LICENSE_OFFICE_DOCUMENTS")
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

    #[test]
    fn import_erad_names_are_unique_and_not_fixed() {
        let a = unique_import_erad_name(Some("memo.docx"));
        let b = unique_import_erad_name(Some("memo.docx"));
        assert!(a.starts_with("memo-") && a.ends_with(".erad"), "{a}");
        assert!(b.starts_with("memo-") && b.ends_with(".erad"), "{b}");
        assert_ne!(a, b, "colliding Drive names caused Import 502 in TE");
        assert_ne!(unique_import_erad_name(None), "import.erad");
    }

    fn test_state(license_ok: bool) -> AppState {
        AppState {
            drive_url: "http://127.0.0.1:1".into(),
            storage: platform_storage("http://127.0.0.1:1"),
            license_ok,
            sessions: Arc::new(Mutex::new(SessionStore::default())),
            pg: None,
            hub: SyncHub::default(),
            presence: PresenceHub::default(),
            jwt_secret: b"test-secret".to_vec(),
            intent_secret: b"intent-secret".to_vec(),
        }
    }

    fn mint_jwt(sub: &str, tenant: &str, exp: i64) -> String {
        let claims = crate::auth::Claims {
            sub: sub.into(),
            tenant_id: tenant.into(),
            email: format!("{sub}@x"),
            exp,
        };
        jsonwebtoken::encode(
            &jsonwebtoken::Header::default(),
            &claims,
            &jsonwebtoken::EncodingKey::from_secret(b"test-secret"),
        )
        .unwrap()
    }

    async fn spawn_app(state: AppState) -> SocketAddr {
        let app = router(state);
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let addr = listener.local_addr().unwrap();
        tokio::spawn(async move {
            axum::serve(listener, app).await.unwrap();
        });
        addr
    }

    #[test]
    fn license_denied_without_module_in_strict() {
        std::env::set_var("ERA_PRODUCTION", "1");
        std::env::remove_var("ERA_OFFICE_DEV");
        std::env::remove_var("ERA_LICENSE_OFFICE_DOCUMENTS");
        assert!(!license_from_env());
        std::env::remove_var("ERA_PRODUCTION");
    }

    #[test]
    fn license_denied_in_production_even_with_office_dev() {
        std::env::set_var("ERA_PRODUCTION", "1");
        std::env::set_var("ERA_OFFICE_DEV", "1");
        std::env::remove_var("ERA_LICENSE_OFFICE_DOCUMENTS");
        assert!(!license_from_env());
        std::env::remove_var("ERA_PRODUCTION");
        std::env::remove_var("ERA_OFFICE_DEV");
    }

    #[test]
    fn license_denied_when_era_env_production_ignores_office_dev() {
        std::env::remove_var("ERA_PRODUCTION");
        std::env::remove_var("ERA_LICENSE_STRICT");
        std::env::remove_var("ERA_ENV_PRODUCTION");
        std::env::set_var("ERA_ENV", "production");
        std::env::set_var("ERA_OFFICE_DEV", "1");
        std::env::remove_var("ERA_LICENSE_OFFICE_DOCUMENTS");
        assert!(!license_from_env());
        std::env::remove_var("ERA_ENV");
        std::env::remove_var("ERA_OFFICE_DEV");
    }

    #[tokio::test]
    async fn create_without_jwt_is_401() {
        let addr = spawn_app(test_state(true)).await;
        let resp = reqwest::Client::new()
            .post(format!("http://{addr}/api/v1/docs"))
            .json(&serde_json::json!({"tenant_id":"t","user_id":"u"}))
            .send()
            .await
            .unwrap();
        assert_eq!(resp.status(), StatusCode::UNAUTHORIZED);
    }

    #[tokio::test]
    async fn create_with_expired_jwt_is_401() {
        let addr = spawn_app(test_state(true)).await;
        let tok = mint_jwt("u1", "t1", 1_600_000_000);
        let resp = reqwest::Client::new()
            .post(format!("http://{addr}/api/v1/docs"))
            .header("Authorization", format!("Bearer {tok}"))
            .json(&serde_json::json!({"name":"x.erad"}))
            .send()
            .await
            .unwrap();
        assert_eq!(resp.status(), StatusCode::UNAUTHORIZED);
    }

    #[tokio::test]
    async fn create_without_license_is_403() {
        let addr = spawn_app(test_state(false)).await;
        let tok = mint_jwt("u1", "t1", 4102444800);
        let resp = reqwest::Client::new()
            .post(format!("http://{addr}/api/v1/docs"))
            .header("Authorization", format!("Bearer {tok}"))
            .json(&serde_json::json!({"name":"x.erad"}))
            .send()
            .await
            .unwrap();
        assert_eq!(resp.status(), StatusCode::FORBIDDEN);
    }

    #[tokio::test]
    async fn get_doc_rejects_cross_tenant() {
        let state = test_state(true);
        {
            let mut store = state.sessions.lock().await;
            store.tenants.insert("obj-b".into(), "tenant-b".into());
            store.docs.insert("obj-b".into(), EradDocument::empty());
            store.logs.insert("obj-b".into(), OpLog::default());
        }
        let addr = spawn_app(state).await;
        let tok_a = mint_jwt("u-a", "tenant-a", 4102444800);
        let resp = reqwest::Client::new()
            .get(format!("http://{addr}/api/v1/docs/obj-b"))
            .header("Authorization", format!("Bearer {tok_a}"))
            .send()
            .await
            .unwrap();
        assert_eq!(resp.status(), StatusCode::FORBIDDEN);
    }

    #[tokio::test]
    async fn verify_intent_ok_and_bad_sig() {
        let addr = spawn_app(test_state(true)).await;
        let tok = mint_jwt("u1", "t1", 4102444800);
        let exp = 4102444800i64;
        let sig = crate::intent::sign_intent(b"intent-secret", "oid-1", exp);
        let ok = reqwest::Client::new()
            .get(format!(
                "http://{addr}/api/v1/docs/oid-1/verify-intent?intent_exp={exp}&intent_sig={sig}"
            ))
            .header("Authorization", format!("Bearer {tok}"))
            .send()
            .await
            .unwrap();
        assert_eq!(ok.status(), StatusCode::NO_CONTENT);

        let bad = reqwest::Client::new()
            .get(format!(
                "http://{addr}/api/v1/docs/oid-1/verify-intent?intent_exp={exp}&intent_sig=deadbeef"
            ))
            .header("Authorization", format!("Bearer {tok}"))
            .send()
            .await
            .unwrap();
        assert_eq!(bad.status(), StatusCode::UNAUTHORIZED);
    }
}
