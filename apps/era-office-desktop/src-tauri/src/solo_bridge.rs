//! Local loopback BFF for Solo Office product SPAs (docs / tables / presentations / projects).
//! Serves static assets + minimal `/api/v1/*` + WS sync for docs/tables.
//! Targets: Browser ❌ · Solo ✅ · Corporate ❌

use std::net::SocketAddr;
use std::path::{Path, PathBuf};
use std::sync::{Arc, Mutex};

use axum::body::Body;
use axum::extract::ws::{Message, WebSocket, WebSocketUpgrade};
use axum::extract::{Path as AxumPath, State};
use axum::http::{header, HeaderMap, StatusCode};
use axum::response::{Html, IntoResponse, Response};
use axum::routing::{delete, get, post};
use axum::{Json, Router};
use era_docs_core::convert::export_docx;
use era_docs_core::model::EradDocument;
use era_docs_core::sync::{apply_op, DocOp};
use era_office_richtext::{FrameKey, FrameOp};
use era_pres_core::convert::export_pptx;
use era_pres_core::convert_odp::export_odp;
use era_pres_core::model::ErapDeck;
use era_projects_core::ProjectTask;
use era_tables_core::calc::recalc;
use era_tables_core::convert::export_xlsx;
use era_tables_core::convert_ods::{export_ods, import_ods};
use era_tables_core::sync::{apply_op as apply_sheet_op, SheetOp};
use futures_util::{SinkExt, StreamExt};
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use tower_http::services::ServeDir;

use crate::solo::SoloState;
use crate::solo_pres::SoloPresState;
use crate::solo_projects::SoloProjectsState;
use crate::solo_tables::SoloTablesState;

pub const SOLO_DOC_ID: &str = "solo";

#[derive(Clone, Debug)]
pub struct AssetRoots {
    pub docs_web: PathBuf,
    pub tables_web: PathBuf,
    pub presentations_web: PathBuf,
    pub projects_web: PathBuf,
    pub office_assets: PathBuf,
    pub boot_js: PathBuf,
    pub skin_css: PathBuf,
    pub drive_web: Option<PathBuf>,
    pub office_ai_web: Option<PathBuf>,
}

#[derive(Clone)]
pub struct BridgeState {
    pub solo: Arc<Mutex<SoloState>>,
    pub tables: Arc<Mutex<SoloTablesState>>,
    pub pres: Arc<Mutex<SoloPresState>>,
    pub projects: Arc<Mutex<SoloProjectsState>>,
    pub docs_web: PathBuf,
    pub tables_web: PathBuf,
    pub presentations_web: PathBuf,
    pub projects_web: PathBuf,
    pub office_assets: PathBuf,
    pub boot_js: PathBuf,
    pub skin_css: PathBuf,
}

#[derive(Clone)]
pub struct BridgeHandle {
    pub base_url: String,
    pub state: BridgeState,
}

#[derive(Serialize)]
struct CreateDocResponse {
    drive_object_id: String,
}

#[derive(Deserialize)]
struct CreateDocRequest {
    #[serde(default)]
    name: Option<String>,
}

#[derive(Deserialize)]
struct SnapshotBody {
    #[serde(default)]
    document: Option<EradDocument>,
}

#[derive(Deserialize)]
struct CreateNamedRequest {
    #[serde(default)]
    name: Option<String>,
}

#[derive(Deserialize)]
struct ImportXlsxReq {
    #[serde(default)]
    name: Option<String>,
    xlsx_base64: String,
}

#[derive(Deserialize)]
struct ImportOdsReq {
    #[serde(default)]
    name: Option<String>,
    ods_base64: String,
}

pub fn default_projects_board() -> SoloProjectsState {
    SoloProjectsState::default()
}

/// Resolve ui paths: (a) next to exe/assets, (b) CARGO_MANIFEST_DIR/assets, (c) repo ui/.
pub fn resolve_asset_roots() -> AssetRoots {
    let manifest = PathBuf::from(env!("CARGO_MANIFEST_DIR"));
    let exe_assets = std::env::current_exe()
        .ok()
        .and_then(|p| p.parent().map(|d| d.join("assets")));
    let bundled = manifest.join("assets");

    let pick_root = |name: &str| -> PathBuf {
        if let Some(ref a) = exe_assets {
            let p = a.join(name);
            if p.is_dir() {
                return p;
            }
        }
        let p = bundled.join(name);
        if p.is_dir() {
            return p;
        }
        let repo_ui = manifest
            .join("../../../ui")
            .canonicalize()
            .unwrap_or_else(|_| manifest.join("../../../ui"));
        match name {
            "docs-web" => repo_ui.join("docs/web"),
            "tables-web" => repo_ui.join("tables/web"),
            "presentations-web" => repo_ui.join("presentations/web"),
            "projects-web" => repo_ui.join("projects/web"),
            "office-assets" => repo_ui.join("office-shell/web"),
            "drive-web" => repo_ui.join("drive/web"),
            "office-ai-web" => repo_ui.join("office-ai/web"),
            _ => repo_ui.join(name),
        }
    };

    let pick_file = |name: &str| -> PathBuf {
        if let Some(ref a) = exe_assets {
            let p = a.join(name);
            if p.is_file() {
                return p;
            }
        }
        let p = bundled.join(name);
        if p.is_file() {
            return p;
        }
        manifest.join("assets").join(name)
    };

    let drive = pick_root("drive-web");
    let office_ai = pick_root("office-ai-web");
    AssetRoots {
        docs_web: pick_root("docs-web"),
        tables_web: pick_root("tables-web"),
        presentations_web: pick_root("presentations-web"),
        projects_web: pick_root("projects-web"),
        office_assets: pick_root("office-assets"),
        boot_js: pick_file("solo-docs-boot.js"),
        skin_css: pick_file("solo-docs-skin.css"),
        drive_web: if drive.is_dir() { Some(drive) } else { None },
        office_ai_web: if office_ai.is_dir() {
            Some(office_ai)
        } else {
            None
        },
    }
}

pub fn router(state: BridgeState) -> Router {
    let docs_static = ServeDir::new(&state.docs_web);
    let tables_static = ServeDir::new(&state.tables_web);
    let presentations_static = ServeDir::new(&state.presentations_web);
    let projects_static = ServeDir::new(&state.projects_web);
    let office_static = ServeDir::new(&state.office_assets);

    Router::new()
        .route(
            "/healthz",
            get(|| async { Json(json!({"status":"ok","service":"era-solo-bridge"})) }),
        )
        .route("/", get(hub_index))
        .route("/solo-docs-boot.js", get(serve_boot_js))
        .route("/solo-docs-skin.css", get(serve_skin_css))
        // Product SPAs
        .route("/docs", get(docs_index))
        .route("/docs/", get(docs_index))
        .route("/docs/:id", get(docs_spa_or_file))
        .route("/tables", get(tables_index))
        .route("/tables/", get(tables_index))
        .route("/tables/:id", get(tables_spa_or_file))
        .route("/presentations", get(presentations_index))
        .route("/presentations/", get(presentations_index))
        .route("/presentations/:id", get(presentations_spa_or_file))
        .route("/projects", get(projects_index))
        .route("/projects/", get(projects_index))
        .route("/projects/:id", get(projects_spa_or_file))
        .nest_service("/docs-static", docs_static)
        .nest_service("/tables-static", tables_static)
        .nest_service("/presentations-static", presentations_static)
        .nest_service("/projects-static", projects_static)
        .nest_service("/office-assets", office_static)
        // Docs API
        .route("/api/v1/docs", post(create_doc))
        .route("/api/v1/docs/import", post(import_doc))
        .route("/api/v1/docs/:id", get(get_doc))
        .route("/api/v1/docs/:id/verify-intent", get(verify_intent))
        .route("/api/v1/docs/:id/snapshot", post(snapshot))
        .route("/api/v1/docs/:id/export/docx", post(export_docx_ep))
        .route("/api/v1/docs/:id/export/rtf", post(export_rtf_ep))
        .route("/api/v1/docs/:id/export/odt", post(export_odt_ep))
        .route("/api/v1/docs/:id/sync", get(sync_ws))
        // Tables API
        .route("/api/v1/tables", post(create_table))
        .route("/api/v1/tables/import", post(import_table_xlsx))
        .route("/api/v1/tables/import-ods", post(import_table_ods))
        .route("/api/v1/tables/:id", get(get_table))
        .route("/api/v1/tables/:id/export/xlsx", post(export_table_xlsx))
        .route("/api/v1/tables/:id/export/ods", post(export_table_ods))
        .route("/api/v1/tables/:id/sync", get(tables_sync_ws))
        // Presentations API
        .route("/api/v1/presentations", post(create_deck))
        .route("/api/v1/presentations/import", post(import_deck_pptx))
        .route(
            "/api/v1/presentations/:id",
            get(get_deck).put(put_deck),
        )
        .route(
            "/api/v1/presentations/:id/frame-op",
            post(apply_frame_op),
        )
        .route(
            "/api/v1/presentations/:id/export/pptx",
            post(export_pptx_ep),
        )
        .route(
            "/api/v1/presentations/:id/export/odp",
            post(export_odp_ep),
        )
        // Projects API
        .route(
            "/api/v1/projects/board",
            get(projects_board_get)
                .put(projects_board_put)
                .post(projects_board_put),
        )
        .route(
            "/api/v1/projects/tasks",
            get(projects_tasks_list).post(projects_tasks_create),
        )
        .route("/api/v1/projects/tasks/:id", delete(projects_task_delete))
        .route("/api/v1/projects", post(projects_create))
        .route(
            "/api/v1/projects/:id",
            get(projects_get).put(projects_board_put_id),
        )
        .route(
            "/api/v1/projects/:id/tasks",
            get(projects_tasks_list_id).post(projects_tasks_create_id),
        )
        .route(
            "/api/v1/projects/:id/tasks/:tid",
            delete(projects_task_delete_id),
        )
        // Drive stubs
        .route("/api/v1/drive/objects/:id/meta", get(drive_meta))
        .route(
            "/api/v1/drive/objects/:id",
            axum::routing::patch(drive_rename),
        )
        .route("/api/v1/drive/objects/:id/versions", get(drive_versions))
        .route("/api/v1/drive/folders/_root/children", get(drive_children))
        .route("/api/v1/solo/file/open", post(solo_file_open))
        .route("/api/v1/solo/file/save", post(solo_file_save))
        .route("/api/v1/solo/file/save-as", post(solo_file_save_as))
        .route("/login", get(login_stub))
        .with_state(state)
}

/// Bind `127.0.0.1:0` and spawn the server. Returns base URL `http://127.0.0.1:port`.
pub async fn start(state: BridgeState) -> anyhow::Result<BridgeHandle> {
    let listener = tokio::net::TcpListener::bind(SocketAddr::from(([127, 0, 0, 1], 0))).await?;
    let addr = listener.local_addr()?;
    let base_url = format!("http://{addr}");
    let app = router(state.clone());
    tokio::spawn(async move {
        if let Err(e) = axum::serve(listener, app).await {
            eprintln!("solo_bridge stopped: {e}");
        }
    });
    Ok(BridgeHandle { base_url, state })
}

pub fn docs_href(base: &str) -> String {
    format!("{}/docs/{}", base.trim_end_matches('/'), SOLO_DOC_ID)
}

pub fn tables_href(base: &str) -> String {
    format!("{}/tables/{}", base.trim_end_matches('/'), SOLO_DOC_ID)
}

pub fn presentations_href(base: &str) -> String {
    format!(
        "{}/presentations/{}",
        base.trim_end_matches('/'),
        SOLO_DOC_ID
    )
}

pub fn projects_href(base: &str) -> String {
    format!("{}/projects/{}", base.trim_end_matches('/'), SOLO_DOC_ID)
}

pub fn hub_href(base: &str) -> String {
    format!("{}/", base.trim_end_matches('/'))
}

pub fn sku_href(base: &str, sku: crate::sku::Sku) -> String {
    match sku {
        crate::sku::Sku::Suite => hub_href(base),
        crate::sku::Sku::Docs => docs_href(base),
        crate::sku::Sku::Tables => tables_href(base),
        crate::sku::Sku::Presentations => presentations_href(base),
        crate::sku::Sku::Projects => projects_href(base),
    }
}

fn inject_boot(mut html: String, static_prefix: &str) -> String {
    if !html.contains("solo-docs-skin.css") {
        html = html.replacen(
            "</head>",
            "  <link rel=\"stylesheet\" href=\"/solo-docs-skin.css\"/>\n</head>",
            1,
        );
    }
    let app_tag = format!("<script src=\"/{static_prefix}/app.js\"></script>");
    let boot_before_app =
        format!("<script src=\"/solo-docs-boot.js\"></script>\n  {app_tag}");
    if !html.contains("solo-docs-boot.js") {
        if html.contains(&app_tag) {
            html = html.replacen(&app_tag, &boot_before_app, 1);
        } else {
            html = html.replacen(
                "<script src=\"/office-assets/icons.js\"></script>",
                "<script src=\"/solo-docs-boot.js\"></script>\n  <script src=\"/office-assets/icons.js\"></script>",
                1,
            );
        }
    }
    let prefix = format!("/{static_prefix}");
    html = html.replace("src=\"app.js\"", &format!("src=\"{prefix}/app.js\""));
    html = html.replace("src=\"later.js\"", &format!("src=\"{prefix}/later.js\""));
    html = html.replace(
        "src=\"era_plus.js\"",
        &format!("src=\"{prefix}/era_plus.js\""),
    );
    html
}

async fn hub_index() -> Html<&'static str> {
    Html(
        r#"<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1"/>
  <title>ERA Office · Solo</title>
  <link rel="stylesheet" href="/solo-docs-skin.css"/>
  <style>
    body{font-family:Segoe UI,system-ui,sans-serif;margin:0;background:#f4f6f8;color:#1a1d21}
    main{max-width:36rem;margin:3rem auto;padding:0 1.25rem}
    h1{font-size:1.5rem;margin:0 0 .35rem}
    p{color:#5c6770;margin:0 0 1.25rem}
    ul{list-style:none;padding:0;margin:0;display:grid;gap:.55rem}
    a{display:block;padding:.85rem 1rem;background:#fff;border:1px solid #d8dee6;border-radius:8px;
      text-decoration:none;color:inherit;font-weight:600}
    a:hover{border-color:#2f6fed;background:#eef3fc}
    a span{display:block;font-weight:400;font-size:.85rem;color:#5c6770;margin-top:.2rem}
  </style>
</head>
<body class="era-solo">
  <main>
    <h1>ERA Office · Solo</h1>
    <p>Local products on this device — no tenant server.</p>
    <ul>
      <li><a href="/docs/solo">Documents<span>Word-like editor · .erad / docx</span></a></li>
      <li><a href="/tables/solo">Tables<span>Spreadsheet · .erat / xlsx / ods</span></a></li>
      <li><a href="/presentations/solo">Presentations<span>Slides · .erap</span></a></li>
      <li><a href="/projects/solo">Projects<span>Board &amp; tasks</span></a></li>
    </ul>
  </main>
</body>
</html>"#,
    )
}

async fn docs_index(State(state): State<BridgeState>) -> impl IntoResponse {
    serve_product_html(&state.docs_web, "docs-static")
}

async fn tables_index(State(state): State<BridgeState>) -> impl IntoResponse {
    serve_product_html(&state.tables_web, "tables-static")
}

async fn presentations_index(State(state): State<BridgeState>) -> impl IntoResponse {
    serve_product_html(&state.presentations_web, "presentations-static")
}

async fn projects_index(State(state): State<BridgeState>) -> impl IntoResponse {
    serve_product_html(&state.projects_web, "projects-static")
}

async fn docs_spa_or_file(
    State(state): State<BridgeState>,
    AxumPath(id): AxumPath<String>,
) -> Response {
    spa_or_file(&state.docs_web, "docs-static", &id)
}

async fn tables_spa_or_file(
    State(state): State<BridgeState>,
    AxumPath(id): AxumPath<String>,
) -> Response {
    spa_or_file(&state.tables_web, "tables-static", &id)
}

async fn presentations_spa_or_file(
    State(state): State<BridgeState>,
    AxumPath(id): AxumPath<String>,
) -> Response {
    spa_or_file(&state.presentations_web, "presentations-static", &id)
}

async fn projects_spa_or_file(
    State(state): State<BridgeState>,
    AxumPath(id): AxumPath<String>,
) -> Response {
    spa_or_file(&state.projects_web, "projects-static", &id)
}

fn spa_or_file(web_root: &Path, static_prefix: &str, id: &str) -> Response {
    if id.contains('.') {
        let path = web_root.join(id);
        if path.is_file() {
            return file_response(&path);
        }
    }
    match serve_product_html(web_root, static_prefix) {
        Ok(html) => html.into_response(),
        Err(e) => (StatusCode::INTERNAL_SERVER_ERROR, e).into_response(),
    }
}

fn serve_product_html(web_root: &Path, static_prefix: &str) -> Result<Html<String>, String> {
    let path = web_root.join("index.html");
    let raw = std::fs::read_to_string(&path).map_err(|e| format!("read index.html: {e}"))?;
    Ok(Html(inject_boot(raw, static_prefix)))
}

fn file_response(path: &Path) -> Response {
    match std::fs::read(path) {
        Ok(bytes) => {
            let mime = mime_guess(path);
            Response::builder()
                .status(StatusCode::OK)
                .header(header::CONTENT_TYPE, mime)
                .body(Body::from(bytes))
                .unwrap_or_else(|_| StatusCode::INTERNAL_SERVER_ERROR.into_response())
        }
        Err(_) => StatusCode::NOT_FOUND.into_response(),
    }
}

fn mime_guess(path: &Path) -> &'static str {
    match path
        .extension()
        .and_then(|e| e.to_str())
        .unwrap_or("")
        .to_ascii_lowercase()
        .as_str()
    {
        "js" => "application/javascript; charset=utf-8",
        "css" => "text/css; charset=utf-8",
        "html" => "text/html; charset=utf-8",
        "svg" => "image/svg+xml",
        "png" => "image/png",
        "json" => "application/json",
        _ => "application/octet-stream",
    }
}

async fn serve_boot_js(State(state): State<BridgeState>) -> Response {
    file_response(&state.boot_js)
}

async fn serve_skin_css(State(state): State<BridgeState>) -> Response {
    file_response(&state.skin_css)
}

async fn login_stub() -> Html<&'static str> {
    Html(
        "<!doctype html><title>Solo</title><p>Solo mode — no login. <a href=\"/\">Open hub</a></p>",
    )
}

// ── Docs ────────────────────────────────────────────────────────────────────

async fn create_doc(
    State(state): State<BridgeState>,
    Json(req): Json<CreateDocRequest>,
) -> Result<Json<CreateDocResponse>, StatusCode> {
    let mut g = state
        .solo
        .lock()
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    g.new_doc();
    if let Some(name) = req.name {
        g.title = name;
    }
    Ok(Json(CreateDocResponse {
        drive_object_id: SOLO_DOC_ID.into(),
    }))
}

async fn get_doc(
    State(state): State<BridgeState>,
    AxumPath(_id): AxumPath<String>,
) -> Result<Json<EradDocument>, StatusCode> {
    let g = state
        .solo
        .lock()
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    Ok(Json(g.doc.clone()))
}

async fn verify_intent() -> StatusCode {
    StatusCode::OK
}

async fn snapshot(
    State(state): State<BridgeState>,
    AxumPath(_id): AxumPath<String>,
    body: Option<Json<SnapshotBody>>,
) -> Result<StatusCode, StatusCode> {
    let mut g = state
        .solo
        .lock()
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    if let Some(Json(b)) = body {
        if let Some(doc) = b.document {
            g.doc = doc;
            g.dirty = true;
        }
    }
    if g.path.is_some() {
        g.save().map_err(|_| StatusCode::FORBIDDEN)?;
    }
    Ok(StatusCode::OK)
}

async fn import_doc(
    State(state): State<BridgeState>,
    headers: HeaderMap,
    body: Body,
) -> Result<Json<CreateDocResponse>, StatusCode> {
    let bytes = axum::body::to_bytes(body, 32 * 1024 * 1024)
        .await
        .map_err(|_| StatusCode::BAD_REQUEST)?;
    let docx = extract_docx_bytes(&headers, &bytes)?;
    let mut g = state
        .solo
        .lock()
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    g.import_docx_bytes(&docx)
        .map_err(|_| StatusCode::BAD_REQUEST)?;
    Ok(Json(CreateDocResponse {
        drive_object_id: SOLO_DOC_ID.into(),
    }))
}

fn extract_docx_bytes(headers: &HeaderMap, bytes: &[u8]) -> Result<Vec<u8>, StatusCode> {
    let ct = headers
        .get(header::CONTENT_TYPE)
        .and_then(|v| v.to_str().ok())
        .unwrap_or("");
    if ct.contains("multipart/") {
        if let Some(i) = find_zip_start(bytes) {
            return Ok(bytes[i..].to_vec());
        }
        return Err(StatusCode::BAD_REQUEST);
    }
    Ok(bytes.to_vec())
}

fn find_zip_start(bytes: &[u8]) -> Option<usize> {
    bytes.windows(4).position(|w| w == [0x50, 0x4b, 0x03, 0x04])
}

async fn export_docx_ep(
    State(state): State<BridgeState>,
    AxumPath(_id): AxumPath<String>,
) -> Result<Response, StatusCode> {
    let g = state
        .solo
        .lock()
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    let bytes = export_docx(&g.doc).map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    Ok(Response::builder()
        .status(StatusCode::OK)
        .header(
            header::CONTENT_TYPE,
            "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
        )
        .header(
            header::CONTENT_DISPOSITION,
            "attachment; filename=\"export.docx\"",
        )
        .body(Body::from(bytes))
        .unwrap())
}

async fn export_rtf_ep(
    State(state): State<BridgeState>,
    AxumPath(_id): AxumPath<String>,
) -> Result<Response, StatusCode> {
    let g = state
        .solo
        .lock()
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    let bytes = era_docs_core::convert::export_rtf(&g.doc);
    Ok(Response::builder()
        .status(StatusCode::OK)
        .header(header::CONTENT_TYPE, "application/rtf")
        .body(Body::from(bytes))
        .unwrap())
}

async fn export_odt_ep(
    State(state): State<BridgeState>,
    AxumPath(_id): AxumPath<String>,
) -> Result<Response, StatusCode> {
    let g = state
        .solo
        .lock()
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    let bytes =
        era_docs_core::convert::export_odt(&g.doc).map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    Ok(Response::builder()
        .status(StatusCode::OK)
        .header(
            header::CONTENT_TYPE,
            "application/vnd.oasis.opendocument.text",
        )
        .body(Body::from(bytes))
        .unwrap())
}

async fn drive_meta(State(state): State<BridgeState>) -> Json<Value> {
    let name = state
        .solo
        .lock()
        .map(|s| s.title.clone())
        .unwrap_or_else(|_| "Untitled.erad".into());
    Json(json!({ "id": SOLO_DOC_ID, "name": name, "kind": "file" }))
}

async fn drive_rename(State(state): State<BridgeState>, Json(body): Json<Value>) -> StatusCode {
    if let Some(name) = body.get("name").and_then(|v| v.as_str()) {
        if let Ok(mut g) = state.solo.lock() {
            g.title = name.to_string();
            g.dirty = true;
        }
    }
    StatusCode::OK
}

async fn drive_versions() -> Json<Value> {
    Json(json!({ "versions": [] }))
}

async fn drive_children() -> Json<Value> {
    Json(json!({ "children": [] }))
}

async fn sync_ws(
    ws: WebSocketUpgrade,
    State(state): State<BridgeState>,
    AxumPath(_id): AxumPath<String>,
) -> impl IntoResponse {
    ws.on_upgrade(move |socket| handle_sync(socket, state))
}

async fn handle_sync(socket: WebSocket, state: BridgeState) {
    let (mut sink, mut stream) = socket.split();
    let _ = sink
        .send(Message::Text(
            json!({"type":"presence","peers":[{"user_id":"solo-user","you":true}]}).to_string(),
        ))
        .await;
    while let Some(Ok(msg)) = stream.next().await {
        let Message::Text(text) = msg else {
            continue;
        };
        if let Ok(op) = serde_json::from_str::<DocOp>(&text) {
            if let Ok(mut g) = state.solo.lock() {
                apply_op(&mut g.doc, &op);
                g.dirty = true;
            }
            let _ = sink.send(Message::Text("ack".into())).await;
        }
    }
}

async fn solo_file_open(
    State(state): State<BridgeState>,
    axum::extract::Query(q): axum::extract::Query<SoloFileQuery>,
) -> Result<Json<CreateDocResponse>, StatusCode> {
    let product = q.product.as_deref().unwrap_or("docs");
    let path = {
        let product = product.to_string();
        tokio::task::spawn_blocking(move || pick_open_for(&product))
            .await
            .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?
            .ok_or(StatusCode::BAD_REQUEST)?
    };
    match product {
        "tables" => {
            let mut g = state
                .tables
                .lock()
                .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
            g.open_path(&path).map_err(|_| StatusCode::BAD_REQUEST)?;
        }
        "presentations" => {
            let mut g = state
                .pres
                .lock()
                .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
            g.open_path(&path).map_err(|_| StatusCode::BAD_REQUEST)?;
        }
        "projects" => {
            let mut g = state
                .projects
                .lock()
                .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
            g.open_path(&path).map_err(|_| StatusCode::BAD_REQUEST)?;
        }
        _ => {
            let mut g = state
                .solo
                .lock()
                .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
            g.open_path(&path).map_err(|_| StatusCode::BAD_REQUEST)?;
        }
    }
    Ok(Json(CreateDocResponse {
        drive_object_id: SOLO_DOC_ID.into(),
    }))
}

async fn solo_file_save(
    State(state): State<BridgeState>,
    axum::extract::Query(q): axum::extract::Query<SoloFileQuery>,
) -> Result<StatusCode, StatusCode> {
    let product = q.product.as_deref().unwrap_or("docs");
    let needs_as = match product {
        "tables" => state
            .tables
            .lock()
            .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?
            .path
            .is_none(),
        "presentations" => state
            .pres
            .lock()
            .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?
            .path
            .is_none(),
        "projects" => state
            .projects
            .lock()
            .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?
            .path
            .is_none(),
        _ => state
            .solo
            .lock()
            .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?
            .path
            .is_none(),
    };
    if needs_as {
        let _ = solo_file_save_as(State(state), axum::extract::Query(q)).await?;
        return Ok(StatusCode::OK);
    }
    match product {
        "tables" => {
            state
                .tables
                .lock()
                .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?
                .save()
                .map_err(|_| StatusCode::FORBIDDEN)?;
        }
        "presentations" => {
            state
                .pres
                .lock()
                .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?
                .save()
                .map_err(|_| StatusCode::FORBIDDEN)?;
        }
        "projects" => {
            state
                .projects
                .lock()
                .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?
                .save()
                .map_err(|_| StatusCode::FORBIDDEN)?;
        }
        _ => {
            state
                .solo
                .lock()
                .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?
                .save()
                .map_err(|_| StatusCode::FORBIDDEN)?;
        }
    }
    Ok(StatusCode::OK)
}

async fn solo_file_save_as(
    State(state): State<BridgeState>,
    axum::extract::Query(q): axum::extract::Query<SoloFileQuery>,
) -> Result<Json<Value>, StatusCode> {
    let product = q.product.as_deref().unwrap_or("docs").to_string();
    let path = {
        let product = product.clone();
        tokio::task::spawn_blocking(move || pick_save_for(&product))
            .await
            .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?
            .ok_or(StatusCode::BAD_REQUEST)?
    };
    match product.as_str() {
        "tables" => {
            state
                .tables
                .lock()
                .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?
                .save_to(&path)
                .map_err(|_| StatusCode::FORBIDDEN)?;
        }
        "presentations" => {
            state
                .pres
                .lock()
                .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?
                .save_to(&path)
                .map_err(|_| StatusCode::FORBIDDEN)?;
        }
        "projects" => {
            state
                .projects
                .lock()
                .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?
                .save_to(&path)
                .map_err(|_| StatusCode::FORBIDDEN)?;
        }
        _ => {
            state
                .solo
                .lock()
                .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?
                .save_to(&path)
                .map_err(|_| StatusCode::FORBIDDEN)?;
        }
    }
    Ok(Json(json!({ "path": path.display().to_string() })))
}

#[derive(Deserialize, Default)]
struct SoloFileQuery {
    #[serde(default)]
    product: Option<String>,
}

fn pick_open_for(product: &str) -> Option<PathBuf> {
    let mut d = rfd::FileDialog::new();
    match product {
        "tables" => {
            d = d
                .add_filter("ERA Tables", &["erat", "xlsx", "ods"])
                .add_filter("All", &["*"]);
        }
        "presentations" => {
            d = d
                .add_filter("ERA Presentations", &["erap", "pptx"])
                .add_filter("All", &["*"]);
        }
        "projects" => {
            d = d
                .add_filter("ERA Projects", &["eraj"])
                .add_filter("All", &["*"]);
        }
        _ => {
            d = d
                .add_filter("ERA Documents", &["erad", "docx"])
                .add_filter("All", &["*"]);
        }
    }
    d.pick_file()
}

fn pick_save_for(product: &str) -> Option<PathBuf> {
    let mut d = rfd::FileDialog::new();
    match product {
        "tables" => {
            d = d
                .add_filter("ERA Table", &["erat"])
                .set_file_name("workbook.erat");
        }
        "presentations" => {
            d = d
                .add_filter("ERA Presentation", &["erap"])
                .set_file_name("deck.erap");
        }
        "projects" => {
            d = d
                .add_filter("ERA Project", &["eraj"])
                .set_file_name("board.eraj");
        }
        _ => {
            d = d
                .add_filter("ERA Document", &["erad"])
                .set_file_name("document.erad");
        }
    }
    d.save_file()
}

// ── Tables ──────────────────────────────────────────────────────────────────

async fn create_table(
    State(state): State<BridgeState>,
    Json(req): Json<CreateNamedRequest>,
) -> Result<Json<CreateDocResponse>, StatusCode> {
    let mut g = state
        .tables
        .lock()
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    g.new_sheet();
    if let Some(name) = req.name {
        g.sheet.name = name;
    }
    g.sheet.drive_object_id = SOLO_DOC_ID.into();
    Ok(Json(CreateDocResponse {
        drive_object_id: SOLO_DOC_ID.into(),
    }))
}

async fn get_table(
    State(state): State<BridgeState>,
    AxumPath(id): AxumPath<String>,
) -> Result<Json<Value>, StatusCode> {
    let mut g = state
        .tables
        .lock()
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    g.sheet.drive_object_id = id;
    g.sheet.normalize_tabs();
    recalc(&mut g.sheet);
    serde_json::to_value(&g.sheet)
        .map(Json)
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)
}

async fn import_table_xlsx(
    State(state): State<BridgeState>,
    Json(req): Json<ImportXlsxReq>,
) -> Result<Json<CreateDocResponse>, StatusCode> {
    use base64::Engine;
    let data = base64::engine::general_purpose::STANDARD
        .decode(req.xlsx_base64.as_bytes())
        .map_err(|_| StatusCode::BAD_REQUEST)?;
    let mut g = state
        .tables
        .lock()
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    g.import_xlsx_bytes(&data)
        .map_err(|_| StatusCode::BAD_REQUEST)?;
    if let Some(name) = req.name {
        g.sheet.name = name;
    }
    g.sheet.drive_object_id = SOLO_DOC_ID.into();
    Ok(Json(CreateDocResponse {
        drive_object_id: SOLO_DOC_ID.into(),
    }))
}

async fn import_table_ods(
    State(state): State<BridgeState>,
    Json(req): Json<ImportOdsReq>,
) -> Result<Json<CreateDocResponse>, StatusCode> {
    use base64::Engine;
    let data = base64::engine::general_purpose::STANDARD
        .decode(req.ods_base64.as_bytes())
        .map_err(|_| StatusCode::BAD_REQUEST)?;
    let mut sheet = import_ods(&data).map_err(|_| StatusCode::BAD_REQUEST)?;
    sheet.normalize_tabs();
    recalc(&mut sheet);
    let mut g = state
        .tables
        .lock()
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    g.sheet = sheet;
    g.path = None;
    g.dirty = true;
    if let Some(name) = req.name {
        g.sheet.name = name;
    }
    g.sheet.drive_object_id = SOLO_DOC_ID.into();
    Ok(Json(CreateDocResponse {
        drive_object_id: SOLO_DOC_ID.into(),
    }))
}

async fn export_table_xlsx(
    State(state): State<BridgeState>,
    AxumPath(_id): AxumPath<String>,
) -> Result<Response, StatusCode> {
    let g = state
        .tables
        .lock()
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    let bytes = export_xlsx(&g.sheet).map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    Ok(Response::builder()
        .status(StatusCode::OK)
        .header(
            header::CONTENT_TYPE,
            "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        )
        .header(
            header::CONTENT_DISPOSITION,
            "attachment; filename=\"export.xlsx\"",
        )
        .body(Body::from(bytes))
        .unwrap())
}

async fn export_table_ods(
    State(state): State<BridgeState>,
    AxumPath(_id): AxumPath<String>,
) -> Result<Response, StatusCode> {
    let g = state
        .tables
        .lock()
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    let bytes = export_ods(&g.sheet).map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    Ok(Response::builder()
        .status(StatusCode::OK)
        .header(
            header::CONTENT_TYPE,
            "application/vnd.oasis.opendocument.spreadsheet",
        )
        .header(
            header::CONTENT_DISPOSITION,
            "attachment; filename=\"export.ods\"",
        )
        .body(Body::from(bytes))
        .unwrap())
}

async fn tables_sync_ws(
    ws: WebSocketUpgrade,
    State(state): State<BridgeState>,
    AxumPath(_id): AxumPath<String>,
) -> impl IntoResponse {
    ws.on_upgrade(move |socket| handle_tables_sync(socket, state))
}

async fn handle_tables_sync(socket: WebSocket, state: BridgeState) {
    let (mut sink, mut stream) = socket.split();
    let _ = sink
        .send(Message::Text(
            json!({"type":"presence","peers":[{"user_id":"solo-user","you":true}]}).to_string(),
        ))
        .await;
    while let Some(Ok(msg)) = stream.next().await {
        let Message::Text(text) = msg else {
            continue;
        };
        // Accept bare SheetOp or {op: SheetOp}
        let op = match serde_json::from_str::<SheetOp>(&text) {
            Ok(op) => Some(op),
            Err(_) => serde_json::from_str::<Value>(&text)
                .ok()
                .and_then(|v| v.get("op").cloned())
                .and_then(|v| serde_json::from_value::<SheetOp>(v).ok()),
        };
        if let Some(op) = op {
            if let Ok(mut g) = state.tables.lock() {
                apply_sheet_op(&mut g.sheet, &op);
                recalc(&mut g.sheet);
                g.dirty = true;
            }
            let _ = sink.send(Message::Text("ack".into())).await;
        }
    }
}

// ── Presentations ───────────────────────────────────────────────────────────

#[derive(Deserialize)]
struct FrameOpReq {
    slide_id: String,
    frame: FrameKey,
    op: FrameOp,
    #[serde(default)]
    base_version: Option<u64>,
}

#[derive(Serialize)]
struct FrameOpResp {
    version: u64,
    deck: ErapDeck,
}

#[derive(Deserialize)]
struct ImportPptxReq {
    #[serde(default)]
    name: Option<String>,
    pptx_base64: String,
}

fn deck_value(deck: &ErapDeck) -> Result<Value, StatusCode> {
    serde_json::to_value(deck).map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)
}

async fn create_deck(
    State(state): State<BridgeState>,
    Json(req): Json<CreateNamedRequest>,
) -> Result<Json<CreateDocResponse>, StatusCode> {
    let mut g = state
        .pres
        .lock()
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    g.new_deck();
    if let Some(name) = req.name {
        g.deck.name = name;
    } else {
        g.deck.name = format!("Untitled-{}.erap", chrono_ish());
    }
    g.deck.drive_object_id = SOLO_DOC_ID.into();
    g.deck.id = SOLO_DOC_ID.into();
    g.deck.tenant_id = "solo".into();
    Ok(Json(CreateDocResponse {
        drive_object_id: SOLO_DOC_ID.into(),
    }))
}

fn chrono_ish() -> u64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_secs())
        .unwrap_or(0)
}

async fn get_deck(
    State(state): State<BridgeState>,
    AxumPath(id): AxumPath<String>,
) -> Result<Json<Value>, StatusCode> {
    let mut g = state
        .pres
        .lock()
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    g.deck.drive_object_id = id.clone();
    if g.deck.id.is_empty() {
        g.deck.id = id;
    }
    if g.deck.slides.is_empty() {
        g.deck = ErapDeck::empty();
        g.deck.drive_object_id = SOLO_DOC_ID.into();
    }
    Ok(Json(deck_value(&g.deck)?))
}

async fn put_deck(
    State(state): State<BridgeState>,
    AxumPath(id): AxumPath<String>,
    Json(body): Json<Value>,
) -> Result<Json<Value>, StatusCode> {
    let mut deck: ErapDeck =
        serde_json::from_value(body).map_err(|_| StatusCode::BAD_REQUEST)?;
    if deck.slides.is_empty() {
        return Err(StatusCode::BAD_REQUEST);
    }
    deck.drive_object_id = id.clone();
    deck.id = id;
    deck.tenant_id = "solo".into();
    deck.version = deck.version.saturating_add(1);
    let mut g = state
        .pres
        .lock()
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    g.set_deck(deck);
    Ok(Json(deck_value(&g.deck)?))
}

async fn apply_frame_op(
    State(state): State<BridgeState>,
    AxumPath(_id): AxumPath<String>,
    Json(req): Json<FrameOpReq>,
) -> Result<Json<FrameOpResp>, StatusCode> {
    let mut g = state
        .pres
        .lock()
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    if let Some(bv) = req.base_version {
        if bv != g.deck.version {
            return Err(StatusCode::CONFLICT);
        }
    }
    let slide = g
        .deck
        .slides
        .iter_mut()
        .find(|s| s.id == req.slide_id)
        .ok_or(StatusCode::BAD_REQUEST)?;
    slide.apply_op(req.frame, &req.op);
    g.deck.version = g.deck.version.saturating_add(1);
    g.dirty = true;
    Ok(Json(FrameOpResp {
        version: g.deck.version,
        deck: g.deck.clone(),
    }))
}

async fn export_pptx_ep(
    State(state): State<BridgeState>,
    AxumPath(_id): AxumPath<String>,
) -> Result<Response, StatusCode> {
    let bytes = {
        let g = state
            .pres
            .lock()
            .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
        export_pptx(&g.deck).map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?
    };
    Ok(Response::builder()
        .status(StatusCode::OK)
        .header(
            header::CONTENT_TYPE,
            "application/vnd.openxmlformats-officedocument.presentationml.presentation",
        )
        .header(
            header::CONTENT_DISPOSITION,
            "attachment; filename=\"deck.pptx\"",
        )
        .body(Body::from(bytes))
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?)
}

async fn export_odp_ep(
    State(state): State<BridgeState>,
    AxumPath(_id): AxumPath<String>,
) -> Result<Response, StatusCode> {
    let bytes = {
        let g = state
            .pres
            .lock()
            .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
        export_odp(&g.deck).map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?
    };
    Ok(Response::builder()
        .status(StatusCode::OK)
        .header(
            header::CONTENT_TYPE,
            "application/vnd.oasis.opendocument.presentation",
        )
        .header(
            header::CONTENT_DISPOSITION,
            "attachment; filename=\"deck.odp\"",
        )
        .body(Body::from(bytes))
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?)
}

async fn import_deck_pptx(
    State(state): State<BridgeState>,
    Json(req): Json<ImportPptxReq>,
) -> Result<Json<CreateDocResponse>, StatusCode> {
    use base64::Engine;
    let data = base64::engine::general_purpose::STANDARD
        .decode(req.pptx_base64.as_bytes())
        .map_err(|_| StatusCode::BAD_REQUEST)?;
    let mut g = state
        .pres
        .lock()
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    g.import_pptx_bytes(&data, req.name)
        .map_err(|_| StatusCode::BAD_REQUEST)?;
    g.deck.drive_object_id = SOLO_DOC_ID.into();
    g.deck.id = SOLO_DOC_ID.into();
    Ok(Json(CreateDocResponse {
        drive_object_id: SOLO_DOC_ID.into(),
    }))
}

// ── Projects ────────────────────────────────────────────────────────────────

fn task_to_value(t: &ProjectTask) -> Value {
    serde_json::to_value(t).unwrap_or(json!({}))
}

fn value_to_task(body: Value) -> Result<ProjectTask, StatusCode> {
    serde_json::from_value(body).map_err(|_| StatusCode::BAD_REQUEST)
}

async fn projects_board_get(State(state): State<BridgeState>) -> Result<Json<Value>, StatusCode> {
    let g = state
        .projects
        .lock()
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    Ok(Json(json!({ "name": g.board.name })))
}

async fn projects_board_put(
    State(state): State<BridgeState>,
    Json(body): Json<Value>,
) -> Result<Json<Value>, StatusCode> {
    let mut g = state
        .projects
        .lock()
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    if let Some(name) = body.get("name").and_then(|v| v.as_str()) {
        g.set_name(name);
    }
    Ok(Json(json!({ "name": g.board.name })))
}

async fn projects_board_put_id(
    State(state): State<BridgeState>,
    AxumPath(id): AxumPath<String>,
    Json(body): Json<Value>,
) -> Result<Json<Value>, StatusCode> {
    let mut g = state
        .projects
        .lock()
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    g.board.id = id.clone();
    if let Some(name) = body.get("name").and_then(|v| v.as_str()) {
        g.set_name(name);
    }
    Ok(Json(json!({
        "name": g.board.name,
        "drive_object_id": id
    })))
}

async fn projects_get(
    State(state): State<BridgeState>,
    AxumPath(id): AxumPath<String>,
) -> Result<Json<Value>, StatusCode> {
    let mut g = state
        .projects
        .lock()
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    g.board.id = id.clone();
    Ok(Json(json!({
        "name": g.board.name,
        "drive_object_id": id
    })))
}

async fn projects_create(
    State(state): State<BridgeState>,
    Json(req): Json<CreateNamedRequest>,
) -> Result<Json<CreateDocResponse>, StatusCode> {
    let id = SOLO_DOC_ID.to_string();
    let name = req.name.unwrap_or_else(|| "Untitled.eraj".into());
    let mut g = state
        .projects
        .lock()
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    g.new_board();
    g.board.id = id.clone();
    g.set_name(&name);
    Ok(Json(CreateDocResponse {
        drive_object_id: id,
    }))
}

async fn projects_tasks_list(State(state): State<BridgeState>) -> Result<Json<Value>, StatusCode> {
    let g = state
        .projects
        .lock()
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    let tasks: Vec<Value> = g.board.tasks.iter().map(task_to_value).collect();
    Ok(Json(json!(tasks)))
}

async fn projects_tasks_list_id(
    State(state): State<BridgeState>,
    AxumPath(_id): AxumPath<String>,
) -> Result<Json<Value>, StatusCode> {
    projects_tasks_list(State(state)).await
}

async fn projects_tasks_create(
    State(state): State<BridgeState>,
    Json(mut body): Json<Value>,
) -> Result<Json<Value>, StatusCode> {
    if body.get("id").and_then(|v| v.as_str()).unwrap_or("").is_empty() {
        body.as_object_mut()
            .unwrap()
            .insert("id".into(), json!(format!("task-{}", chrono_ish())));
    }
    if body.get("board").and_then(|v| v.as_str()).unwrap_or("").is_empty() {
        body.as_object_mut()
            .unwrap()
            .insert("board".into(), json!("backlog"));
    }
    let task = value_to_task(body.clone())?;
    let mut g = state
        .projects
        .lock()
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    g.upsert_task(task);
    Ok(Json(body))
}

async fn projects_tasks_create_id(
    State(state): State<BridgeState>,
    AxumPath(pid): AxumPath<String>,
    Json(mut body): Json<Value>,
) -> Result<Json<Value>, StatusCode> {
    if body.get("id").and_then(|v| v.as_str()).unwrap_or("").is_empty() {
        body.as_object_mut()
            .unwrap()
            .insert("id".into(), json!(format!("task-{}", chrono_ish())));
    }
    if body.get("board").and_then(|v| v.as_str()).unwrap_or("").is_empty() {
        body.as_object_mut()
            .unwrap()
            .insert("board".into(), json!("backlog"));
    }
    body.as_object_mut()
        .unwrap()
        .insert("drive_object_id".into(), json!(pid));
    let task = value_to_task(body.clone())?;
    let mut g = state
        .projects
        .lock()
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    g.board.id = pid;
    g.upsert_task(task);
    Ok(Json(body))
}

async fn projects_task_delete(
    State(state): State<BridgeState>,
    AxumPath(id): AxumPath<String>,
) -> StatusCode {
    if let Ok(mut g) = state.projects.lock() {
        g.remove_task(&id);
    }
    StatusCode::OK
}

async fn projects_task_delete_id(
    State(state): State<BridgeState>,
    AxumPath((_pid, tid)): AxumPath<(String, String)>,
) -> StatusCode {
    projects_task_delete(State(state), AxumPath(tid)).await
}

#[cfg(test)]
mod tests {
    use super::*;
    use axum::body::Body;
    use axum::http::Request;
    use http_body_util::BodyExt;
    use tower::ServiceExt;

    fn test_state() -> BridgeState {
        let roots = resolve_asset_roots();
        // Prefer repo ui for tests when bundled assets missing scripts.
        let manifest = PathBuf::from(env!("CARGO_MANIFEST_DIR"));
        let repo_ui = manifest.join("../../../ui");
        BridgeState {
            solo: Arc::new(Mutex::new(SoloState::default())),
            tables: Arc::new(Mutex::new(SoloTablesState::default())),
            pres: Arc::new(Mutex::new(SoloPresState::default())),
            projects: Arc::new(Mutex::new(SoloProjectsState::default())),
            docs_web: if roots.docs_web.join("index.html").is_file() {
                roots.docs_web
            } else {
                repo_ui.join("docs/web")
            },
            tables_web: roots.tables_web,
            presentations_web: roots.presentations_web,
            projects_web: roots.projects_web,
            office_assets: roots.office_assets,
            boot_js: roots.boot_js,
            skin_css: roots.skin_css,
        }
    }

    #[tokio::test]
    async fn create_get_roundtrip() {
        let state = test_state();
        let app = router(state.clone());
        let res = app
            .clone()
            .oneshot(
                Request::builder()
                    .method("POST")
                    .uri("/api/v1/docs")
                    .header("content-type", "application/json")
                    .body(Body::from(r#"{"name":"t.erad"}"#))
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(res.status(), StatusCode::OK);
        let res = app
            .oneshot(
                Request::builder()
                    .uri("/api/v1/docs/solo")
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(res.status(), StatusCode::OK);
        let bytes = res.into_body().collect().await.unwrap().to_bytes();
        let _doc: EradDocument = serde_json::from_slice(&bytes).unwrap();
        let g = state.solo.lock().unwrap();
        assert_eq!(g.title, "t.erad");
    }

    #[tokio::test]
    async fn snapshot_with_document_body() {
        let state = test_state();
        {
            let mut g = state.solo.lock().unwrap();
            g.new_doc();
        }
        let app = router(state.clone());
        let mut doc = EradDocument::empty();
        doc.id = "snap-test".into();
        let body = json!({ "document": doc });
        let res = app
            .oneshot(
                Request::builder()
                    .method("POST")
                    .uri("/api/v1/docs/solo/snapshot")
                    .header("content-type", "application/json")
                    .body(Body::from(serde_json::to_vec(&body).unwrap()))
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(res.status(), StatusCode::OK);
        let g = state.solo.lock().unwrap();
        assert_eq!(g.doc.id, "snap-test");
    }

    #[tokio::test]
    async fn docs_index_injects_boot() {
        let state = test_state();
        if !state.docs_web.join("index.html").is_file() {
            return;
        }
        let app = router(state);
        let res = app
            .oneshot(
                Request::builder()
                    .uri("/docs/solo")
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(res.status(), StatusCode::OK);
        let bytes = res.into_body().collect().await.unwrap().to_bytes();
        let html = String::from_utf8_lossy(&bytes);
        assert!(html.contains("solo-docs-boot.js"));
        assert!(html.contains("/docs-static/app.js"));
    }

    #[tokio::test]
    async fn hub_lists_products() {
        let state = test_state();
        let app = router(state);
        let res = app
            .oneshot(
                Request::builder()
                    .uri("/")
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(res.status(), StatusCode::OK);
        let bytes = res.into_body().collect().await.unwrap().to_bytes();
        let html = String::from_utf8_lossy(&bytes);
        assert!(html.contains("/docs/solo"));
        assert!(html.contains("/tables/solo"));
        assert!(html.contains("/presentations/solo"));
        assert!(html.contains("/projects/solo"));
    }

    #[tokio::test]
    async fn tables_create_get() {
        let state = test_state();
        let app = router(state.clone());
        let res = app
            .clone()
            .oneshot(
                Request::builder()
                    .method("POST")
                    .uri("/api/v1/tables")
                    .header("content-type", "application/json")
                    .body(Body::from(r#"{"name":"t.erat"}"#))
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(res.status(), StatusCode::OK);
        let res = app
            .oneshot(
                Request::builder()
                    .uri("/api/v1/tables/solo")
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(res.status(), StatusCode::OK);
        let g = state.tables.lock().unwrap();
        assert_eq!(g.sheet.name, "t.erat");
    }

    #[tokio::test]
    async fn presentations_create_get() {
        let state = test_state();
        let app = router(state);
        let res = app
            .clone()
            .oneshot(
                Request::builder()
                    .method("POST")
                    .uri("/api/v1/presentations")
                    .header("content-type", "application/json")
                    .body(Body::from(r#"{"name":"d.erap"}"#))
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(res.status(), StatusCode::OK);
        let res = app
            .oneshot(
                Request::builder()
                    .uri("/api/v1/presentations/solo")
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(res.status(), StatusCode::OK);
        let bytes = res.into_body().collect().await.unwrap().to_bytes();
        let deck: Value = serde_json::from_slice(&bytes).unwrap();
        assert!(deck.get("slides").and_then(|s| s.as_array()).is_some());
        assert!(deck["slides"][0].get("title_frame").is_some());
    }

    #[tokio::test]
    async fn presentations_frame_op_and_pptx_export() {
        let state = test_state();
        let app = router(state.clone());
        let _ = app
            .clone()
            .oneshot(
                Request::builder()
                    .method("POST")
                    .uri("/api/v1/presentations")
                    .header("content-type", "application/json")
                    .body(Body::from(r#"{"name":"d.erap"}"#))
                    .unwrap(),
            )
            .await
            .unwrap();
        let slide_id = {
            let g = state.pres.lock().unwrap();
            g.deck.slides[0].id.clone()
        };
        let block_id = {
            let g = state.pres.lock().unwrap();
            g.deck.slides[0].title_frame.blocks[0].id.clone()
        };
        let body = json!({
            "slide_id": slide_id,
            "frame": "title",
            "op": { "type": "insert_text", "block_id": block_id, "offset": 0, "text": "Z" }
        });
        let res = app
            .clone()
            .oneshot(
                Request::builder()
                    .method("POST")
                    .uri("/api/v1/presentations/solo/frame-op")
                    .header("content-type", "application/json")
                    .body(Body::from(serde_json::to_vec(&body).unwrap()))
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(res.status(), StatusCode::OK);
        let res = app
            .oneshot(
                Request::builder()
                    .method("POST")
                    .uri("/api/v1/presentations/solo/export/pptx")
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(res.status(), StatusCode::OK);
        let bytes = res.into_body().collect().await.unwrap().to_bytes();
        assert!(bytes.len() > 100);
    }

    #[tokio::test]
    async fn projects_board_and_tasks() {
        let state = test_state();
        let app = router(state);
        let res = app
            .clone()
            .oneshot(
                Request::builder()
                    .uri("/api/v1/projects/board")
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(res.status(), StatusCode::OK);
        let res = app
            .clone()
            .oneshot(
                Request::builder()
                    .method("POST")
                    .uri("/api/v1/projects/tasks")
                    .header("content-type", "application/json")
                    .body(Body::from(r#"{"title":"Hello","board":"backlog"}"#))
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(res.status(), StatusCode::OK);
        let res = app
            .oneshot(
                Request::builder()
                    .uri("/api/v1/projects/tasks")
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(res.status(), StatusCode::OK);
        let bytes = res.into_body().collect().await.unwrap().to_bytes();
        let tasks: Value = serde_json::from_slice(&bytes).unwrap();
        assert_eq!(tasks.as_array().unwrap().len(), 1);
    }
}
