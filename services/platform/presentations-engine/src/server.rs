use std::collections::HashMap;
use std::net::SocketAddr;
use std::sync::Arc;

use axum::extract::{Path, State};
use axum::http::{HeaderMap, StatusCode};
use axum::response::IntoResponse;
use axum::routing::{get, post};
use axum::{Json, Router};
use serde::{Deserialize, Serialize};
use tokio::net::TcpListener;
use tokio::sync::Mutex;

use crate::auth;
use crate::convert::{export_pptx, import_pptx};
use crate::convert_odp::export_odp;
use crate::drive_bind::DriveClient;
use crate::model::ErapDeck;
use era_office_richtext::{transform_frame_op, FrameKey, FrameOp};

#[derive(Clone)]
pub struct AppState {
    pub drive_url: String,
    pub license_ok: bool,
    pub decks: Arc<Mutex<HashMap<String, ErapDeck>>>,
    pub jwt_secret: Vec<u8>,
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
    pub pptx_base64: String,
}

#[derive(Serialize)]
struct Health {
    status: &'static str,
    service: &'static str,
}

pub fn router(state: AppState) -> Router {
    Router::new()
        .route(
            "/healthz",
            get(|| async {
                Json(Health {
                    status: "ok",
                    service: "era-presentations-engine",
                })
            }),
        )
        .route("/api/v1/presentations", post(create))
        .route("/api/v1/presentations/import", post(import_deck))
        .route("/api/v1/presentations/:id", get(get_deck).put(put_deck))
        .route(
            "/api/v1/presentations/:id/frame-op",
            post(apply_deck_frame_op),
        )
        .route("/api/v1/presentations/:id/export/pptx", post(export_deck))
        .route("/api/v1/presentations/:id/export/odp", post(export_deck_odp))
        .with_state(state)
}

#[derive(Deserialize)]
pub struct FrameOpReq {
    pub slide_id: String,
    pub frame: FrameKey,
    pub op: FrameOp,
    /// Client's last known deck version; mismatch → 409 + snapshot.
    #[serde(default)]
    pub base_version: Option<u64>,
}

#[derive(Serialize)]
pub struct FrameOpResp {
    pub version: u64,
    pub deck: ErapDeck,
}

fn require_principal(state: &AppState, headers: &HeaderMap) -> Result<auth::Principal, StatusCode> {
    auth::principal_from_headers(headers, &state.jwt_secret)
}

async fn create(
    State(state): State<AppState>,
    headers: HeaderMap,
    Json(req): Json<CreateReq>,
) -> Result<Json<CreateResp>, StatusCode> {
    if !state.license_ok {
        return Err(StatusCode::FORBIDDEN);
    }
    let p = require_principal(&state, &headers)?;
    let name = req.name.unwrap_or_else(|| "deck.erap".into());
    let mut deck = ErapDeck::empty();
    deck.name = name.clone();
    deck.tenant_id = p.tenant_id.clone();
    let client = DriveClient::new(&state.drive_url);
    let id = client
        .put_erap(&p.tenant_id, &p.user_id, &name, &deck, None)
        .await
        .map_err(|_| StatusCode::BAD_GATEWAY)?;
    deck.drive_object_id = id.clone();
    state.decks.lock().await.insert(id.clone(), deck);
    Ok(Json(CreateResp {
        drive_object_id: id,
    }))
}

async fn import_deck(
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
        .decode(req.pptx_base64.as_bytes())
        .map_err(|_| StatusCode::BAD_REQUEST)?;
    let mut deck = import_pptx(&data).map_err(|_| StatusCode::BAD_REQUEST)?;
    let name = req.name.unwrap_or_else(|| "import.erap".into());
    deck.name = name.clone();
    deck.tenant_id = p.tenant_id.clone();
    let client = DriveClient::new(&state.drive_url);
    let id = client
        .put_erap(&p.tenant_id, &p.user_id, &name, &deck, None)
        .await
        .map_err(|_| StatusCode::BAD_GATEWAY)?;
    deck.drive_object_id = id.clone();
    state.decks.lock().await.insert(id.clone(), deck);
    Ok(Json(CreateResp {
        drive_object_id: id,
    }))
}

async fn apply_deck_frame_op(
    State(state): State<AppState>,
    headers: HeaderMap,
    Path(id): Path<String>,
    Json(req): Json<FrameOpReq>,
) -> Result<Json<FrameOpResp>, StatusCode> {
    if !state.license_ok {
        return Err(StatusCode::FORBIDDEN);
    }
    let p = require_principal(&state, &headers)?;
    let mut store = state.decks.lock().await;
    let deck = store.get_mut(&id).ok_or(StatusCode::NOT_FOUND)?;
    if deck.tenant_id.is_empty() || deck.tenant_id != p.tenant_id {
        return Err(StatusCode::FORBIDDEN);
    }
    if let Some(bv) = req.base_version {
        if bv != deck.version {
            return Err(StatusCode::CONFLICT);
        }
    }
    let slide = deck
        .slides
        .iter_mut()
        .find(|s| s.id == req.slide_id)
        .ok_or(StatusCode::BAD_REQUEST)?;
    // Transform against nothing in-memory (single-writer lab); apply directly.
    let _ = transform_frame_op(&req.op, &req.op);
    slide.apply_op(req.frame, &req.op);
    deck.version = deck.version.saturating_add(1);
    let out = deck.clone();
    drop(store);
    // Persist best-effort (same as put).
    let client = DriveClient::new(&state.drive_url);
    let name = if out.name.is_empty() {
        format!("{id}.erap")
    } else {
        out.name.clone()
    };
    let _ = client
        .put_erap(&p.tenant_id, &p.user_id, &name, &out, Some(&id))
        .await;
    Ok(Json(FrameOpResp {
        version: out.version,
        deck: out,
    }))
}

async fn put_deck(
    State(state): State<AppState>,
    headers: HeaderMap,
    Path(id): Path<String>,
    Json(mut deck): Json<ErapDeck>,
) -> Result<Json<ErapDeck>, StatusCode> {
    if !state.license_ok {
        return Err(StatusCode::FORBIDDEN);
    }
    let p = require_principal(&state, &headers)?;
    {
        let store = state.decks.lock().await;
        if let Some(existing) = store.get(&id) {
            if existing.tenant_id.is_empty() || existing.tenant_id != p.tenant_id {
                return Err(StatusCode::FORBIDDEN);
            }
        }
    }
    if deck.slides.is_empty() {
        return Err(StatusCode::BAD_REQUEST);
    }
    deck.drive_object_id = id.clone();
    deck.tenant_id = p.tenant_id.clone();
    deck.version = deck.version.saturating_add(1);
    let name = if deck.name.is_empty() {
        format!("{id}.erap")
    } else {
        deck.name.clone()
    };
    let client = DriveClient::new(&state.drive_url);
    client
        .put_erap(&p.tenant_id, &p.user_id, &name, &deck, Some(&id))
        .await
        .map_err(|_| StatusCode::BAD_GATEWAY)?;
    state.decks.lock().await.insert(id, deck.clone());
    Ok(Json(deck))
}

async fn load_deck_for_export(
    state: &AppState,
    headers: &HeaderMap,
    id: &str,
) -> Result<ErapDeck, StatusCode> {
    if !state.license_ok {
        return Err(StatusCode::FORBIDDEN);
    }
    let p = require_principal(state, headers)?;
    let deck = {
        let store = state.decks.lock().await;
        if let Some(d) = store.get(id) {
            if d.tenant_id.is_empty() || d.tenant_id != p.tenant_id {
                return Err(StatusCode::FORBIDDEN);
            }
            Some(d.clone())
        } else {
            None
        }
    };
    match deck {
        Some(d) => Ok(d),
        None => {
            let client = DriveClient::new(&state.drive_url);
            let bytes = client
                .get_erap_json(&p.tenant_id, &p.user_id, id)
                .await
                .map_err(|_| StatusCode::NOT_FOUND)?;
            serde_json::from_slice(&bytes).map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)
        }
    }
}

async fn export_deck(
    State(state): State<AppState>,
    headers: HeaderMap,
    Path(id): Path<String>,
) -> Result<impl IntoResponse, StatusCode> {
    let deck = load_deck_for_export(&state, &headers, &id).await?;
    let bytes = export_pptx(&deck).map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    Ok((
        [
            (
                axum::http::header::CONTENT_TYPE,
                "application/vnd.openxmlformats-officedocument.presentationml.presentation",
            ),
            (
                axum::http::header::CONTENT_DISPOSITION,
                "attachment; filename=\"export.pptx\"",
            ),
        ],
        bytes,
    ))
}

async fn export_deck_odp(
    State(state): State<AppState>,
    headers: HeaderMap,
    Path(id): Path<String>,
) -> Result<impl IntoResponse, StatusCode> {
    let deck = load_deck_for_export(&state, &headers, &id).await?;
    let bytes = export_odp(&deck).map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    Ok((
        [
            (
                axum::http::header::CONTENT_TYPE,
                "application/vnd.oasis.opendocument.presentation",
            ),
            (
                axum::http::header::CONTENT_DISPOSITION,
                "attachment; filename=\"export.odp\"",
            ),
        ],
        bytes,
    ))
}

async fn get_deck(
    State(state): State<AppState>,
    headers: HeaderMap,
    Path(id): Path<String>,
) -> Result<Json<ErapDeck>, StatusCode> {
    if !state.license_ok {
        return Err(StatusCode::FORBIDDEN);
    }
    let p = require_principal(&state, &headers)?;
    {
        let store = state.decks.lock().await;
        if let Some(deck) = store.get(&id) {
            // Always require non-empty tenant bind matching JWT (no empty-tenant skip).
            if deck.tenant_id.is_empty() || deck.tenant_id != p.tenant_id {
                return Err(StatusCode::FORBIDDEN);
            }
            return Ok(Json(deck.clone()));
        }
    }
    let client = DriveClient::new(&state.drive_url);
    let bytes = client
        .get_erap_json(&p.tenant_id, &p.user_id, &id)
        .await
        .map_err(|_| StatusCode::NOT_FOUND)?;
    let mut deck: ErapDeck =
        serde_json::from_slice(&bytes).map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    deck.drive_object_id = id.clone();
    deck.tenant_id = p.tenant_id.clone();
    state.decks.lock().await.insert(id, deck.clone());
    Ok(Json(deck))
}

pub async fn serve(addr: SocketAddr, state: AppState) -> anyhow::Result<()> {
    axum::serve(TcpListener::bind(addr).await?, router(state)).await?;
    Ok(())
}

pub fn license_from_env() -> bool {
    if production_like() {
        return std::env::var("ERA_LICENSE_OFFICE_PRESENTATIONS")
            .map(|v| v == "1" || v == "true")
            .unwrap_or(false);
    }
    if env_truthy("ERA_OFFICE_DEV") {
        return true;
    }
    std::env::var("ERA_LICENSE_OFFICE_PRESENTATIONS")
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

    #[test]
    fn license_denied_without_module_in_strict() {
        std::env::set_var("ERA_PRODUCTION", "1");
        std::env::remove_var("ERA_OFFICE_DEV");
        std::env::remove_var("ERA_LICENSE_OFFICE_PRESENTATIONS");
        assert!(!license_from_env());
        std::env::remove_var("ERA_PRODUCTION");
    }

    #[test]
    fn license_denied_when_era_env_production_ignores_office_dev() {
        std::env::remove_var("ERA_PRODUCTION");
        std::env::remove_var("ERA_LICENSE_STRICT");
        std::env::set_var("ERA_ENV", "production");
        std::env::set_var("ERA_OFFICE_DEV", "1");
        std::env::remove_var("ERA_LICENSE_OFFICE_PRESENTATIONS");
        assert!(!license_from_env());
        std::env::remove_var("ERA_ENV");
        std::env::remove_var("ERA_OFFICE_DEV");
    }

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

    async fn spawn(state: AppState) -> SocketAddr {
        let app = router(state);
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let addr = listener.local_addr().unwrap();
        tokio::spawn(async move {
            axum::serve(listener, app).await.unwrap();
        });
        addr
    }

    #[tokio::test]
    async fn get_deck_rejects_cross_tenant() {
        let state = AppState {
            drive_url: "http://127.0.0.1:1".into(),
            license_ok: true,
            decks: Arc::new(Mutex::new(HashMap::new())),
            jwt_secret: b"s".to_vec(),
        };
        {
            let mut deck = ErapDeck::empty();
            deck.tenant_id = "tenant-b".into();
            deck.name = "b.erap".into();
            state.decks.lock().await.insert("deck-b".into(), deck);
        }
        let addr = spawn(state).await;
        let tok_a = mint("u-a", "tenant-a", 4102444800);
        let resp = reqwest::Client::new()
            .get(format!("http://{addr}/api/v1/presentations/deck-b"))
            .header("Authorization", format!("Bearer {tok_a}"))
            .send()
            .await
            .unwrap();
        assert_eq!(resp.status(), StatusCode::FORBIDDEN);
    }

    #[tokio::test]
    async fn get_deck_rejects_empty_tenant() {
        let state = AppState {
            drive_url: "http://127.0.0.1:1".into(),
            license_ok: true,
            decks: Arc::new(Mutex::new(HashMap::new())),
            jwt_secret: b"s".to_vec(),
        };
        {
            let mut deck = ErapDeck::empty();
            deck.tenant_id = String::new();
            state.decks.lock().await.insert("deck-empty".into(), deck);
        }
        let addr = spawn(state).await;
        let tok = mint("u", "tenant-a", 4102444800);
        let resp = reqwest::Client::new()
            .get(format!("http://{addr}/api/v1/presentations/deck-empty"))
            .header("Authorization", format!("Bearer {tok}"))
            .send()
            .await
            .unwrap();
        assert_eq!(resp.status(), StatusCode::FORBIDDEN);
    }

    #[tokio::test]
    async fn presentations_create_without_jwt_401() {
        let state = AppState {
            drive_url: "http://127.0.0.1:1".into(),
            license_ok: true,
            decks: Arc::new(Mutex::new(HashMap::new())),
            jwt_secret: b"s".to_vec(),
        };
        let app = router(state);
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let addr = listener.local_addr().unwrap();
        tokio::spawn(async move {
            axum::serve(listener, app).await.unwrap();
        });
        let resp = reqwest::Client::new()
            .post(format!("http://{addr}/api/v1/presentations"))
            .json(&serde_json::json!({}))
            .send()
            .await
            .unwrap();
        assert_eq!(resp.status(), StatusCode::UNAUTHORIZED);
    }

    #[tokio::test]
    async fn presentations_get_without_license_403() {
        let state = AppState {
            drive_url: "http://127.0.0.1:1".into(),
            license_ok: false,
            decks: Arc::new(Mutex::new(HashMap::new())),
            jwt_secret: b"s".to_vec(),
        };
        let claims = auth::Claims {
            sub: "u".into(),
            tenant_id: "t".into(),
            email: "u@x".into(),
            exp: 4102444800,
        };
        let tok = jsonwebtoken::encode(
            &jsonwebtoken::Header::default(),
            &claims,
            &jsonwebtoken::EncodingKey::from_secret(b"s"),
        )
        .unwrap();
        let app = router(state);
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let addr = listener.local_addr().unwrap();
        tokio::spawn(async move {
            axum::serve(listener, app).await.unwrap();
        });
        let resp = reqwest::Client::new()
            .get(format!("http://{addr}/api/v1/presentations/any"))
            .header("Authorization", format!("Bearer {tok}"))
            .send()
            .await
            .unwrap();
        assert_eq!(resp.status(), StatusCode::FORBIDDEN);
    }

    #[tokio::test]
    async fn presentations_create_without_license_403() {
        let state = AppState {
            drive_url: "http://127.0.0.1:1".into(),
            license_ok: false,
            decks: Arc::new(Mutex::new(HashMap::new())),
            jwt_secret: b"s".to_vec(),
        };
        let claims = auth::Claims {
            sub: "u".into(),
            tenant_id: "t".into(),
            email: "u@x".into(),
            exp: 4102444800,
        };
        let tok = jsonwebtoken::encode(
            &jsonwebtoken::Header::default(),
            &claims,
            &jsonwebtoken::EncodingKey::from_secret(b"s"),
        )
        .unwrap();
        let app = router(state);
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let addr = listener.local_addr().unwrap();
        tokio::spawn(async move {
            axum::serve(listener, app).await.unwrap();
        });
        let resp = reqwest::Client::new()
            .post(format!("http://{addr}/api/v1/presentations"))
            .header("Authorization", format!("Bearer {tok}"))
            .json(&serde_json::json!({}))
            .send()
            .await
            .unwrap();
        assert_eq!(resp.status(), StatusCode::FORBIDDEN);
    }

    #[tokio::test]
    async fn put_deck_persists_to_drive() {
        use axum::body::to_bytes;
        use axum::extract::{Path, Request};
        use axum::routing::{get, post};
        use std::sync::atomic::{AtomicUsize, Ordering};

        std::env::set_var("ERA_DRIVE_SERVICE_TOKEN", "test-drive-service-token");
        let uploads = Arc::new(AtomicUsize::new(0));
        let blobs: Arc<Mutex<HashMap<String, Vec<u8>>>> = Arc::new(Mutex::new(HashMap::new()));
        let drive_app = Router::new()
            .route(
                "/api/v1/drive/objects/:id/versions",
                post({
                    let uploads = uploads.clone();
                    let blobs = blobs.clone();
                    move |Path(id): Path<String>, req: Request| {
                        let uploads = uploads.clone();
                        let blobs = blobs.clone();
                        async move {
                            let body = to_bytes(req.into_body(), 2_000_000)
                                .await
                                .unwrap_or_default();
                            let start = body.iter().position(|&b| b == b'{').unwrap_or(0);
                            let end = body.iter().rposition(|&b| b == b'}').unwrap_or(0);
                            let json = if end >= start {
                                body[start..=end].to_vec()
                            } else {
                                serde_json::to_vec(&ErapDeck::empty()).unwrap()
                            };
                            blobs.lock().await.insert(id.clone(), json);
                            uploads.fetch_add(1, Ordering::SeqCst);
                            Json(serde_json::json!({ "id": id }))
                        }
                    }
                }),
            )
            .route(
                "/api/v1/drive/objects/:id",
                get({
                    let blobs = blobs.clone();
                    move |Path(id): Path<String>| {
                        let blobs = blobs.clone();
                        async move {
                            let bytes = blobs.lock().await.get(&id).cloned().unwrap_or_default();
                            (
                                StatusCode::OK,
                                [(
                                    axum::http::header::CONTENT_TYPE,
                                    "application/json",
                                )],
                                bytes,
                            )
                        }
                    }
                }),
            );
        let drive_listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let drive_addr = drive_listener.local_addr().unwrap();
        tokio::spawn(async move {
            axum::serve(drive_listener, drive_app).await.unwrap();
        });

        let decks = Arc::new(Mutex::new(HashMap::new()));
        {
            let mut deck = ErapDeck::empty();
            deck.tenant_id = "tenant-a".into();
            deck.name = "save.erap".into();
            decks.lock().await.insert("deck-put-1".into(), deck);
        }
        let state = AppState {
            drive_url: format!("http://{drive_addr}"),
            license_ok: true,
            decks: decks.clone(),
            jwt_secret: b"s".to_vec(),
        };
        let addr = spawn(state).await;
        let tok = mint("u-a", "tenant-a", 4102444800);
        let body = serde_json::json!({
            "name": "save.erap",
            "format": "erap",
            "slides": [{"id":"s1","title":"T","body":"reopen-marker","layout":"title_body"}]
        });
        let resp = reqwest::Client::new()
            .put(format!("http://{addr}/api/v1/presentations/deck-put-1"))
            .header("Authorization", format!("Bearer {tok}"))
            .json(&body)
            .send()
            .await
            .unwrap();
        assert_eq!(resp.status(), StatusCode::OK);
        assert!(
            uploads.load(Ordering::SeqCst) >= 1,
            "put_deck must PutVersion via put_erap"
        );

        // Restart: clear in-memory decks, reopen same id from Drive.
        decks.lock().await.clear();
        let reopened: ErapDeck = reqwest::Client::new()
            .get(format!("http://{addr}/api/v1/presentations/deck-put-1"))
            .header("Authorization", format!("Bearer {tok}"))
            .send()
            .await
            .unwrap()
            .json()
            .await
            .unwrap();
        assert_eq!(reopened.drive_object_id, "deck-put-1");
        assert_eq!(
            reopened.slides[0].body(),
            "reopen-marker",
            "content after reopen must match PutVersion"
        );
    }

    #[tokio::test]
    async fn presentations_create_with_expired_jwt_401() {
        let state = AppState {
            drive_url: "http://127.0.0.1:1".into(),
            license_ok: true,
            decks: Arc::new(Mutex::new(HashMap::new())),
            jwt_secret: b"s".to_vec(),
        };
        let claims = auth::Claims {
            sub: "u".into(),
            tenant_id: "t".into(),
            email: "u@x".into(),
            exp: 1_600_000_000,
        };
        let tok = jsonwebtoken::encode(
            &jsonwebtoken::Header::default(),
            &claims,
            &jsonwebtoken::EncodingKey::from_secret(b"s"),
        )
        .unwrap();
        let app = router(state);
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let addr = listener.local_addr().unwrap();
        tokio::spawn(async move {
            axum::serve(listener, app).await.unwrap();
        });
        let resp = reqwest::Client::new()
            .post(format!("http://{addr}/api/v1/presentations"))
            .header("Authorization", format!("Bearer {tok}"))
            .json(&serde_json::json!({}))
            .send()
            .await
            .unwrap();
        assert_eq!(resp.status(), StatusCode::UNAUTHORIZED);
    }
}
