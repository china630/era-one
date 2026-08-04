use std::collections::HashMap;
use std::net::SocketAddr;
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::Arc;

use axum::body::{to_bytes, Body};
use axum::extract::{Path, Request};
use axum::http::StatusCode;
use axum::routing::{get, post};
use axum::{Json, Router};
use era_tables_engine::auth::Claims;
use era_tables_engine::model::EratSheet;
use era_tables_engine::server::{self, AppState, PresenceHub, SessionStore, SyncHub};
use futures_util::{SinkExt, StreamExt};
use jsonwebtoken::{encode, EncodingKey, Header};
use serde::Deserialize;
use serde_json::json;
use tokio::net::TcpListener;
use tokio::sync::Mutex;
use tokio_tungstenite::{connect_async, tungstenite::Message};

const JWT_SECRET: &[u8] = b"tables-ws-secret";

fn ensure_drive_token() {
    std::env::set_var("ERA_DRIVE_SERVICE_TOKEN", "test-drive-service-token");
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

async fn recv_ack(
    ws: &mut (impl StreamExt<Item = Result<Message, tokio_tungstenite::tungstenite::Error>> + Unpin),
) {
    for _ in 0..8 {
        if recv_text_skip_presence(ws).await == "ack" {
            return;
        }
    }
    panic!("no ack within message budget");
}

#[derive(Deserialize)]
struct CreateResp {
    drive_object_id: String,
}

fn token() -> String {
    token_for("u1")
}

fn token_for(sub: &str) -> String {
    let claims = Claims {
        sub: sub.into(),
        tenant_id: "t1".into(),
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

/// Pull JSON object from multipart upload body (Drive put_erat).
fn extract_json_blob(body: &[u8]) -> Option<Vec<u8>> {
    let start = body.iter().position(|&b| b == b'{')?;
    let end = body.iter().rposition(|&b| b == b'}')?;
    if end < start {
        return None;
    }
    Some(body[start..=end].to_vec())
}

async fn spawn_mock_drive(upload_count: Option<Arc<AtomicUsize>>) -> SocketAddr {
    let count = upload_count.unwrap_or_else(|| Arc::new(AtomicUsize::new(0)));
    let blobs: Arc<Mutex<HashMap<String, Vec<u8>>>> = Arc::new(Mutex::new(HashMap::new()));
    let app = Router::new()
        .route(
            "/api/v1/drive/objects",
            post({
                let count = count.clone();
                let blobs = blobs.clone();
                move |req: Request| {
                    let count = count.clone();
                    let blobs = blobs.clone();
                    async move {
                        let body = to_bytes(req.into_body(), 2_000_000)
                            .await
                            .unwrap_or_default();
                        let json = extract_json_blob(&body).unwrap_or_else(|| {
                            serde_json::to_vec(&EratSheet::empty()).unwrap()
                        });
                        let id = "drive-sheet-1".to_string();
                        blobs.lock().await.insert(id.clone(), json);
                        count.fetch_add(1, Ordering::SeqCst);
                        Json(json!({ "id": id }))
                    }
                }
            }),
        )
        .route(
            "/api/v1/drive/objects/:id/versions",
            post({
                let count = count.clone();
                let blobs = blobs.clone();
                move |Path(id): Path<String>, req: Request| {
                    let count = count.clone();
                    let blobs = blobs.clone();
                    async move {
                        let body = to_bytes(req.into_body(), 2_000_000)
                            .await
                            .unwrap_or_default();
                        let json = extract_json_blob(&body).unwrap_or_else(|| {
                            serde_json::to_vec(&EratSheet::empty()).unwrap()
                        });
                        blobs.lock().await.insert(id.clone(), json);
                        count.fetch_add(1, Ordering::SeqCst);
                        Json(json!({ "id": id }))
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
                        let bytes = blobs.lock().await.get(&id).cloned().unwrap_or_else(|| {
                            serde_json::to_vec(&EratSheet::empty()).unwrap()
                        });
                        axum::response::Response::builder()
                            .status(StatusCode::OK)
                            .header("content-type", "application/json")
                            .body(Body::from(bytes))
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

#[tokio::test]
async fn ws_sheet_two_clients_merge() {
    ensure_drive_token();
    let drive = spawn_mock_drive(None).await;
    let state = AppState {
        drive_url: format!("http://{drive}"),
        storage: era_tables_engine::drive_bind::platform_storage(format!("http://{drive}")),
        license_ok: true,
        sessions: Arc::new(Mutex::new(SessionStore::default())),
        hub: SyncHub::default(),
        presence: PresenceHub::default(),
        jwt_secret: JWT_SECRET.to_vec(),
    };
    let app = server::router(state);
    let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
    let addr = listener.local_addr().unwrap();
    tokio::spawn(async move {
        axum::serve(listener, app).await.unwrap();
    });
    let base = format!("http://{addr}");
    let tok = token();
    let client = reqwest::Client::new();
    let create: CreateResp = client
        .post(format!("{base}/api/v1/tables"))
        .header("Authorization", format!("Bearer {tok}"))
        .json(&json!({"name":"coedit.erat"}))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();
    let id = create.drive_object_id;
    let ws_url = format!("ws://{addr}/api/v1/tables/{id}/sync?access_token={tok}");
    let (mut a, _) = connect_async(&ws_url).await.unwrap();
    let (mut b, _) = connect_async(&ws_url).await.unwrap();
    a.send(Message::Text(
        json!({"type":"set_cell","addr":"A1","value":"1"})
            .to_string()
            .into(),
    ))
    .await
    .unwrap();
    recv_ack(&mut a).await;
    let peer_txt = recv_text_skip_presence(&mut b).await;
    assert!(
        peer_txt.contains("A1") || peer_txt.contains("set_cell"),
        "peer fan-out missing: {peer_txt}"
    );
    b.send(Message::Text(
        json!({"type":"set_cell","addr":"B1","value":"2"})
            .to_string()
            .into(),
    ))
    .await
    .unwrap();
    recv_ack(&mut b).await;
    let sheet: EratSheet = client
        .get(format!("{base}/api/v1/tables/{id}"))
        .header("Authorization", format!("Bearer {tok}"))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();
    assert_eq!(sheet.cells["A1"].value, "1");
    assert_eq!(sheet.cells["B1"].value, "2");
}

/// After WS apply_op, engine schedules debounced put_erat (AC-T1/T5).
#[tokio::test]
async fn ws_edit_flushes_to_drive() {
    ensure_drive_token();
    std::env::set_var("ERA_TABLES_FLUSH_DEBOUNCE_MS", "80");
    let uploads = Arc::new(AtomicUsize::new(0));
    let drive = spawn_mock_drive(Some(uploads.clone())).await;
    let state = AppState {
        drive_url: format!("http://{drive}"),
        storage: era_tables_engine::drive_bind::platform_storage(format!("http://{drive}")),
        license_ok: true,
        sessions: Arc::new(Mutex::new(SessionStore::default())),
        hub: SyncHub::default(),
        presence: PresenceHub::default(),
        jwt_secret: JWT_SECRET.to_vec(),
    };
    let app = server::router(state);
    let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
    let addr = listener.local_addr().unwrap();
    tokio::spawn(async move {
        axum::serve(listener, app).await.unwrap();
    });
    let base = format!("http://{addr}");
    let tok = token();
    let client = reqwest::Client::new();
    let create: CreateResp = client
        .post(format!("{base}/api/v1/tables"))
        .header("Authorization", format!("Bearer {tok}"))
        .json(&json!({"name":"flush.erat"}))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();
    assert_eq!(uploads.load(Ordering::SeqCst), 1, "create must put_erat");
    let id = create.drive_object_id;
    let ws_url = format!("ws://{addr}/api/v1/tables/{id}/sync?access_token={tok}");
    let (mut a, _) = connect_async(&ws_url).await.unwrap();
    let _ = tokio::time::timeout(std::time::Duration::from_millis(300), a.next()).await;
    a.send(Message::Text(
        json!({"type":"set_cell","addr":"C1","value":"9"})
            .to_string()
            .into(),
    ))
    .await
    .unwrap();
    let mut got_ack = false;
    for _ in 0..4 {
        let msg = tokio::time::timeout(std::time::Duration::from_secs(2), a.next())
            .await
            .unwrap()
            .unwrap()
            .unwrap();
        if msg == Message::Text("ack".into()) {
            got_ack = true;
            break;
        }
    }
    assert!(got_ack);
    tokio::time::sleep(std::time::Duration::from_millis(250)).await;
    assert!(
        uploads.load(Ordering::SeqCst) >= 2,
        "expected debounced put_erat after WS edit, uploads={}",
        uploads.load(Ordering::SeqCst)
    );
}

/// Edit → flush (PutVersion) → clear engine session → reopen same id → content matches.
#[tokio::test]
async fn ws_edit_flush_reopen_same_id_content() {
    ensure_drive_token();
    std::env::set_var("ERA_TABLES_FLUSH_DEBOUNCE_MS", "80");
    let uploads = Arc::new(AtomicUsize::new(0));
    let drive = spawn_mock_drive(Some(uploads.clone())).await;
    let sessions = Arc::new(Mutex::new(SessionStore::default()));
    let state = AppState {
        drive_url: format!("http://{drive}"),
        storage: era_tables_engine::drive_bind::platform_storage(format!("http://{drive}")),
        license_ok: true,
        sessions: sessions.clone(),
        hub: SyncHub::default(),
        presence: PresenceHub::default(),
        jwt_secret: JWT_SECRET.to_vec(),
    };
    let app = server::router(state);
    let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
    let addr = listener.local_addr().unwrap();
    tokio::spawn(async move {
        axum::serve(listener, app).await.unwrap();
    });
    let base = format!("http://{addr}");
    let tok = token();
    let client = reqwest::Client::new();
    let create: CreateResp = client
        .post(format!("{base}/api/v1/tables"))
        .header("Authorization", format!("Bearer {tok}"))
        .json(&json!({"name":"reopen.erat"}))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();
    let id = create.drive_object_id.clone();
    let ws_url = format!("ws://{addr}/api/v1/tables/{id}/sync?access_token={tok}");
    let (mut a, _) = connect_async(&ws_url).await.unwrap();
    let _ = tokio::time::timeout(std::time::Duration::from_millis(300), a.next()).await;
    a.send(Message::Text(
        json!({"type":"set_cell","addr":"D1","value":"reopen-marker"})
            .to_string()
            .into(),
    ))
    .await
    .unwrap();
    for _ in 0..4 {
        let msg = tokio::time::timeout(std::time::Duration::from_secs(2), a.next())
            .await
            .unwrap()
            .unwrap()
            .unwrap();
        if msg == Message::Text("ack".into()) {
            break;
        }
    }
    tokio::time::sleep(std::time::Duration::from_millis(300)).await;
    assert!(
        uploads.load(Ordering::SeqCst) >= 2,
        "flush must PutVersion after edit"
    );

    // Simulate engine restart: drop in-memory sessions.
    {
        let mut store = sessions.lock().await;
        *store = SessionStore::default();
    }

    let sheet: EratSheet = client
        .get(format!("{base}/api/v1/tables/{id}"))
        .header("Authorization", format!("Bearer {tok}"))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();
    assert_eq!(sheet.drive_object_id, id, "stable Drive object id");
    assert_eq!(
        sheet.cells.get("D1").map(|c| c.value.as_str()),
        Some("reopen-marker"),
        "content after reopen must match flushed PutVersion"
    );
}

/// Peer active-cell highlights are relayed as presence_cell (not SheetOp).
#[tokio::test]
async fn ws_presence_cell_relay() {
    ensure_drive_token();
    let drive = spawn_mock_drive(None).await;
    let state = AppState {
        drive_url: format!("http://{drive}"),
        storage: era_tables_engine::drive_bind::platform_storage(format!("http://{drive}")),
        license_ok: true,
        sessions: Arc::new(Mutex::new(SessionStore::default())),
        hub: SyncHub::default(),
        presence: PresenceHub::default(),
        jwt_secret: JWT_SECRET.to_vec(),
    };
    let app = server::router(state);
    let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
    let addr = listener.local_addr().unwrap();
    tokio::spawn(async move {
        axum::serve(listener, app).await.unwrap();
    });
    let base = format!("http://{addr}");
    let tok_a = token_for("u-cell-a");
    let tok_b = token_for("u-cell-b");
    let client = reqwest::Client::new();
    let create: CreateResp = client
        .post(format!("{base}/api/v1/tables"))
        .header("Authorization", format!("Bearer {tok_a}"))
        .json(&json!({"name": "presence.erat"}))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();
    let id = create.drive_object_id;
    let ws_a = format!("ws://{addr}/api/v1/tables/{id}/sync?access_token={tok_a}");
    let ws_b = format!("ws://{addr}/api/v1/tables/{id}/sync?access_token={tok_b}");
    let (mut sock_a, _) = connect_async(&ws_a).await.unwrap();
    let (mut sock_b, _) = connect_async(&ws_b).await.unwrap();
    for sock in [&mut sock_a, &mut sock_b] {
        let _ = tokio::time::timeout(std::time::Duration::from_millis(400), sock.next()).await;
    }
    sock_a
        .send(Message::Text(
            json!({
                "type": "presence_cell",
                "addr": "C3",
                "color": "#c45"
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
            .expect("timeout waiting for presence_cell")
            .expect("closed")
            .expect("err");
        let text = match msg {
            Message::Text(t) => t.to_string(),
            _ => continue,
        };
        if text.contains("presence_cell") && text.contains("u-cell-a") && text.contains("C3") {
            found = true;
            break;
        }
    }
    assert!(found, "peer B did not receive presence_cell relay");
}
