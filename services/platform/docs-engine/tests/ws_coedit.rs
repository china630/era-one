//! WebSocket co-edit integration tests (AC-O1) — JWT + OpLog fan-out.

use std::collections::HashMap;
use std::net::SocketAddr;
use std::sync::Arc;

use axum::body::{to_bytes, Body};
use axum::extract::{Path, Request};
use axum::http::StatusCode;
use axum::routing::{get, post};
use axum::{Json, Router};
use era_docs_engine::auth::Claims;
use era_docs_engine::canonical::to_canonical_json;
use era_docs_engine::model::EradDocument;
use era_docs_engine::persist::PgPersist;
use era_docs_engine::server::{self, AppState, SessionStore};
use futures_util::{SinkExt, StreamExt};
use jsonwebtoken::{encode, EncodingKey, Header};
use serde::Deserialize;
use serde_json::json;
use tokio::net::TcpListener;
use tokio::sync::Mutex;
use tokio_tungstenite::{connect_async, tungstenite::Message};

const JWT_SECRET: &[u8] = b"test-ws-secret";

fn ensure_drive_token() {
    std::env::set_var("ERA_DRIVE_SERVICE_TOKEN", "test-drive-service-token");
}

async fn recv_sync_envelope(ws: &mut (impl StreamExt<Item = Result<Message, tokio_tungstenite::tungstenite::Error>> + Unpin)) -> String {
    for _ in 0..8 {
        let text = recv_text_skip_presence(ws).await;
        if text.contains("insert_text") || (text.contains("\"op\"") && !text.contains("\"type\":\"presence\"")) {
            return text;
        }
    }
    panic!("no sync envelope within message budget");
}

async fn recv_ack(ws: &mut (impl StreamExt<Item = Result<Message, tokio_tungstenite::tungstenite::Error>> + Unpin)) {
    for _ in 0..8 {
        let text = recv_text_skip_presence(ws).await;
        if text == "ack" {
            return;
        }
    }
    panic!("no ack within message budget");
}

async fn recv_text_skip_presence(
    ws: &mut (impl StreamExt<Item = Result<Message, tokio_tungstenite::tungstenite::Error>> + Unpin),
) -> String {
    loop {
        let msg = tokio::time::timeout(std::time::Duration::from_secs(2), ws.next())
            .await
            .expect("ws recv timeout")
            .expect("ws closed")
            .expect("ws error");
        let text = match msg {
            Message::Text(t) => t.to_string(),
            other => panic!("expected text message, got {other:?}"),
        };
        if text.contains("\"type\":\"presence\"") || text.contains("\"type\": \"presence\"") {
            continue;
        }
        return text;
    }
}

#[derive(Deserialize)]
struct CreateResp {
    drive_object_id: String,
}

fn token(sub: &str, tenant: &str) -> String {
    let claims = Claims {
        sub: sub.into(),
        tenant_id: tenant.into(),
        email: format!("{sub}@x"),
        exp: 4102444800,
    };
    encode(
        &Header::default(),
        &claims,
        &EncodingKey::from_secret(JWT_SECRET),
    )
    .unwrap()
}

fn extract_json_blob(body: &[u8]) -> Option<Vec<u8>> {
    let start = body.iter().position(|&b| b == b'{')?;
    let end = body.iter().rposition(|&b| b == b'}')?;
    if end < start {
        return None;
    }
    Some(body[start..=end].to_vec())
}

async fn spawn_mock_drive() -> SocketAddr {
    let blobs: Arc<Mutex<HashMap<String, Vec<u8>>>> = Arc::new(Mutex::new(HashMap::new()));
    let app = Router::new()
        .route(
            "/api/v1/drive/objects",
            post({
                let blobs = blobs.clone();
                move |req: Request| {
                    let blobs = blobs.clone();
                    async move {
                        let body = to_bytes(req.into_body(), 2_000_000)
                            .await
                            .unwrap_or_default();
                        let json = extract_json_blob(&body).unwrap_or_else(|| {
                            to_canonical_json(&EradDocument::empty()).unwrap().into_bytes()
                        });
                        let id = "drive-obj-coedit-1".to_string();
                        blobs.lock().await.insert(id.clone(), json);
                        Json(serde_json::json!({ "id": id }))
                    }
                }
            }),
        )
        .route(
            "/api/v1/drive/objects/:id/versions",
            post({
                let blobs = blobs.clone();
                move |Path(id): Path<String>, req: Request| {
                    let blobs = blobs.clone();
                    async move {
                        let body = to_bytes(req.into_body(), 2_000_000)
                            .await
                            .unwrap_or_default();
                        let json = extract_json_blob(&body).unwrap_or_else(|| {
                            to_canonical_json(&EradDocument::empty()).unwrap().into_bytes()
                        });
                        blobs.lock().await.insert(id.clone(), json);
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
                        let json = blobs.lock().await.get(&id).cloned().unwrap_or_else(|| {
                            to_canonical_json(&EradDocument::empty()).unwrap().into_bytes()
                        });
                        axum::response::Response::builder()
                            .status(StatusCode::OK)
                            .header("content-type", "application/json")
                            .body(Body::from(json))
                            .unwrap()
                    }
                }
            }),
        );
    let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
    let addr = listener.local_addr().unwrap();
    tokio::spawn(async move {
        axum::serve(listener, app).await.unwrap();
    });
    addr
}

async fn spawn_docs_engine(drive_url: String, pg: Option<Arc<PgPersist>>) -> SocketAddr {
    let state = AppState {
        storage: era_docs_engine::drive_bind::platform_storage(drive_url.clone()),
        drive_url,
        license_ok: true,
        sessions: Arc::new(Mutex::new(SessionStore::default())),
        pg,
        hub: server::SyncHub::default(),
        presence: server::PresenceHub::default(),
        jwt_secret: JWT_SECRET.to_vec(),
        intent_secret: b"test-intent".to_vec(),
    };
    let app = server::router(state);
    let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
    let addr = listener.local_addr().unwrap();
    tokio::spawn(async move {
        axum::serve(listener, app).await.unwrap();
    });
    addr
}

#[tokio::test]
async fn ws_coedit_unauth_rejected() {
    ensure_drive_token();
    let drive_addr = spawn_mock_drive().await;
    let docs_addr = spawn_docs_engine(format!("http://{drive_addr}"), None).await;
    let ws_url = format!("ws://{docs_addr}/api/v1/docs/any/sync");
    let err = connect_async(&ws_url).await;
    assert!(err.is_err(), "unauth WS must fail");
}

#[tokio::test]
async fn ws_coedit_two_clients_merge() {
    ensure_drive_token();
    let drive_addr = spawn_mock_drive().await;
    let drive_url = format!("http://{drive_addr}");
    let docs_addr = spawn_docs_engine(drive_url, None).await;
    let base = format!("http://{docs_addr}");
    let tok = token("u1", "t-coedit");

    let client = reqwest::Client::new();
    let create: CreateResp = client
        .post(format!("{base}/api/v1/docs"))
        .header("Authorization", format!("Bearer {tok}"))
        .json(&json!({"name": "coedit.erad"}))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();
    let id = create.drive_object_id;

    let ws_url = format!("ws://{docs_addr}/api/v1/docs/{id}/sync?access_token={tok}");
    let (mut ws_a, _) = connect_async(&ws_url).await.unwrap();
    let (mut ws_b, _) = connect_async(&ws_url).await.unwrap();

    let block_id = {
        let doc: EradDocument = client
            .get(format!("{base}/api/v1/docs/{id}"))
            .header("Authorization", format!("Bearer {tok}"))
            .send()
            .await
            .unwrap()
            .json()
            .await
            .unwrap();
        doc.blocks[0].id.clone()
    };

    ws_a.send(Message::Text(
        json!({
            "type": "insert_text",
            "block_id": block_id,
            "offset": 0,
            "text": "Hello"
        })
        .to_string()
        .into(),
    ))
    .await
    .unwrap();
    recv_ack(&mut ws_a).await;

    let peer_text = recv_sync_envelope(&mut ws_b).await;
    assert!(
        peer_text.contains("Hello") && peer_text.contains("insert_text"),
        "peer live op missing Hello: {peer_text}"
    );

    ws_b.send(Message::Text(
        json!({
            "type": "insert_text",
            "block_id": block_id,
            "offset": 5,
            "text": " world"
        })
        .to_string()
        .into(),
    ))
    .await
    .unwrap();
    recv_ack(&mut ws_b).await;

    let peer_a_text = recv_sync_envelope(&mut ws_a).await;
    assert!(
        peer_a_text.contains(" world") || peer_a_text.contains("world"),
        "peer A missing B op: {peer_a_text}"
    );

    let merged: EradDocument = client
        .get(format!("{base}/api/v1/docs/{id}"))
        .header("Authorization", format!("Bearer {tok}"))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();
    let text = merged.plain_text();
    assert!(text.contains("Hello"), "got: {text}");
    assert!(text.contains("world"), "got: {text}");
}

/// Edit → snapshot (PutVersion) → new engine process → reopen same id → content matches.
#[tokio::test]
async fn edit_snapshot_reopen_same_id_content() {
    ensure_drive_token();
    let drive_addr = spawn_mock_drive().await;
    let drive_url = format!("http://{drive_addr}");
    let docs_addr = spawn_docs_engine(drive_url.clone(), None).await;
    let base = format!("http://{docs_addr}");
    let tok = token("u1", "t-reopen");
    let client = reqwest::Client::new();

    let create: CreateResp = client
        .post(format!("{base}/api/v1/docs"))
        .header("Authorization", format!("Bearer {tok}"))
        .json(&json!({"name": "reopen.erad"}))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();
    let id = create.drive_object_id.clone();

    let ws_url = format!("ws://{docs_addr}/api/v1/docs/{id}/sync?access_token={tok}");
    let (mut ws, _) = connect_async(&ws_url).await.unwrap();
    let block_id = {
        let doc: EradDocument = client
            .get(format!("{base}/api/v1/docs/{id}"))
            .header("Authorization", format!("Bearer {tok}"))
            .send()
            .await
            .unwrap()
            .json()
            .await
            .unwrap();
        doc.blocks[0].id.clone()
    };
    ws.send(Message::Text(
        json!({
            "type": "insert_text",
            "block_id": block_id,
            "offset": 0,
            "text": "snapshot-marker"
        })
        .to_string()
        .into(),
    ))
    .await
    .unwrap();
    recv_ack(&mut ws).await;

    let snap = client
        .post(format!("{base}/api/v1/docs/{id}/snapshot"))
        .header("Authorization", format!("Bearer {tok}"))
        .json(&json!({}))
        .send()
        .await
        .unwrap();
    assert_eq!(snap.status(), StatusCode::OK);

    // Simulate engine restart: fresh process, same Drive.
    let docs_addr2 = spawn_docs_engine(drive_url, None).await;
    let base2 = format!("http://{docs_addr2}");
    let reopened: EradDocument = client
        .get(format!("{base2}/api/v1/docs/{id}"))
        .header("Authorization", format!("Bearer {tok}"))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();
    // Same object id in URL; content must come from PutVersion blob.
    assert!(
        reopened.plain_text().contains("snapshot-marker"),
        "content after reopen same id must match PutVersion; got: {}",
        reopened.plain_text()
    );
}

#[tokio::test]
async fn ws_coedit_pg_reload_after_cache_clear() {
    ensure_drive_token();
    let dsn = match std::env::var("ERA_OFFICE_DATABASE_URL") {
        Ok(v) if !v.trim().is_empty() => v,
        _ => return,
    };
    let pg = Arc::new(PgPersist::from_env().await.unwrap().expect("pg pool"));
    let drive_addr = spawn_mock_drive().await;
    let drive_url = format!("http://{drive_addr}");
    let tok = token("u1", "t-pg-coedit");

    let docs_addr = spawn_docs_engine(drive_url.clone(), Some(pg.clone())).await;
    let base = format!("http://{docs_addr}");
    let client = reqwest::Client::new();

    let create: CreateResp = client
        .post(format!("{base}/api/v1/docs"))
        .header("Authorization", format!("Bearer {tok}"))
        .json(&json!({"name": "pg.erad"}))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();
    let id = create.drive_object_id;

    let ws_url = format!("ws://{docs_addr}/api/v1/docs/{id}/sync?access_token={tok}");
    let (mut ws, _) = connect_async(&ws_url).await.unwrap();
    let doc: EradDocument = client
        .get(format!("{base}/api/v1/docs/{id}"))
        .header("Authorization", format!("Bearer {tok}"))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();
    let block_id = doc.blocks[0].id.clone();

    ws.send(Message::Text(
        json!({
            "type": "insert_text",
            "block_id": block_id,
            "offset": 0,
            "text": "persisted"
        })
        .to_string()
        .into(),
    ))
    .await
    .unwrap();
    recv_ack(&mut ws).await;

    let docs_addr2 = spawn_docs_engine(drive_url, Some(pg)).await;
    let base2 = format!("http://{docs_addr2}");
    let reloaded: EradDocument = client
        .get(format!("{base2}/api/v1/docs/{id}"))
        .header("Authorization", format!("Bearer {tok}"))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();
    assert!(
        reloaded.plain_text().contains("persisted"),
        "got: {}",
        reloaded.plain_text()
    );
    let _ = dsn;
}

/// Peer carets are relayed as presence_caret (not DocOp).
#[tokio::test]
async fn ws_presence_caret_relay() {
    ensure_drive_token();
    let drive_addr = spawn_mock_drive().await;
    let drive_url = format!("http://{drive_addr}");
    let docs_addr = spawn_docs_engine(drive_url, None).await;
    let base = format!("http://{docs_addr}");
    let tok_a = token("u-caret-a", "t-caret");
    let tok_b = token("u-caret-b", "t-caret");

    let client = reqwest::Client::new();
    let create: CreateResp = client
        .post(format!("{base}/api/v1/docs"))
        .header("Authorization", format!("Bearer {tok_a}"))
        .json(&json!({"name": "caret.erad"}))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();
    let id = create.drive_object_id;

    let ws_a = format!("ws://{docs_addr}/api/v1/docs/{id}/sync?access_token={tok_a}");
    let ws_b = format!("ws://{docs_addr}/api/v1/docs/{id}/sync?access_token={tok_b}");
    let (mut sock_a, _) = connect_async(&ws_a).await.unwrap();
    let (mut sock_b, _) = connect_async(&ws_b).await.unwrap();

    // Drain join presence noise.
    for sock in [&mut sock_a, &mut sock_b] {
        let _ = tokio::time::timeout(std::time::Duration::from_millis(400), sock.next()).await;
    }

    sock_a
        .send(Message::Text(
            json!({
                "type": "presence_caret",
                "caret": { "block_id": "b1", "offset": 3, "color": "#c45" }
            })
            .to_string()
            .into(),
        ))
        .await
        .unwrap();

    let mut found = false;
    for _ in 0..16 {
        let msg = tokio::time::timeout(std::time::Duration::from_secs(2), sock_b.next())
            .await
            .expect("timeout waiting for caret")
            .expect("closed")
            .expect("err");
        let text = match msg {
            Message::Text(t) => t.to_string(),
            _ => continue,
        };
        if text.contains("presence_caret")
            && text.contains("u-caret-a")
            && text.contains("b1")
        {
            found = true;
            break;
        }
    }
    assert!(found, "peer B did not receive presence_caret relay");
}
