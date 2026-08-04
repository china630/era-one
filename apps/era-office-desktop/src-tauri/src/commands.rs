//! Shared Tauri commands — config + corporate (S4), Solo docs (S3), Solo tables (S5).

use std::path::PathBuf;
use std::sync::{Arc, Mutex};

use tauri::{AppHandle, Manager, State};
use tauri_plugin_dialog::{DialogExt, FilePath};

use crate::config::{self, DesktopConfig, Profile};
use crate::corp::{self, CorpOpenTarget};
use crate::license::LicenseStatus;
use crate::sku::Sku;
use crate::solo::{DocSnapshot, SoloError, SoloState};
use crate::solo_bridge::{self, BridgeHandle};
use crate::solo_pres::SoloPresState;
use crate::solo_projects::SoloProjectsState;
use crate::solo_tables::{SheetSnapshot, SoloTablesError, SoloTablesState};

pub struct AppState(pub Arc<Mutex<SoloState>>);

pub struct TablesState(pub Arc<Mutex<SoloTablesState>>);

pub struct PresState(pub Arc<Mutex<SoloPresState>>);

pub struct ProjectsState(pub Arc<Mutex<SoloProjectsState>>);

pub struct ConfigState(pub Mutex<DesktopConfig>);

pub struct BridgeUrlState(pub Mutex<Option<BridgeHandle>>);

pub struct SkuState(pub Sku);

fn map_err(e: SoloError) -> String {
    e.to_string()
}

fn map_tables_err(e: SoloTablesError) -> String {
    e.to_string()
}

fn cfg_err(e: config::ConfigError) -> String {
    e.to_string()
}

fn corp_err(e: corp::CorpError) -> String {
    e.to_string()
}

#[derive(Clone, serde::Serialize)]
pub struct ConfigView {
    pub profile: Profile,
    pub server_url: Option<String>,
    /// True when no persisted config file yet (show profile chooser).
    pub first_run: bool,
    pub sku: String,
}

#[tauri::command]
pub fn config_get(
    state: State<'_, ConfigState>,
    sku: State<'_, SkuState>,
) -> Result<ConfigView, String> {
    let g = state.0.lock().map_err(|e| e.to_string())?;
    let first_run = config::default_config_path()
        .map(|p| !p.exists())
        .unwrap_or(true);
    Ok(ConfigView {
        profile: g.profile,
        server_url: g.server_url.clone(),
        first_run,
        sku: sku.0.as_str().to_string(),
    })
}

#[tauri::command]
pub fn sku_get(sku: State<'_, SkuState>) -> Result<String, String> {
    Ok(sku.0.as_str().to_string())
}

#[tauri::command]
pub fn config_set(
    state: State<'_, ConfigState>,
    profile: String,
    server_url: Option<String>,
) -> Result<DesktopConfig, String> {
    let profile = Profile::parse(&profile).ok_or_else(|| "invalid profile".to_string())?;
    let server_url = match server_url {
        Some(s) if !s.trim().is_empty() => {
            Some(DesktopConfig::normalize_server_url(&s).map_err(cfg_err)?)
        }
        _ => None,
    };
    let cfg = DesktopConfig {
        profile,
        server_url,
    };
    config::save(&cfg).map_err(cfg_err)?;
    let mut g = state.0.lock().map_err(|e| e.to_string())?;
    *g = cfg.clone();
    Ok(cfg)
}

#[tauri::command]
pub fn corp_go(
    app: AppHandle,
    state: State<'_, ConfigState>,
    path: Option<String>,
) -> Result<String, String> {
    let g = state.0.lock().map_err(|e| e.to_string())?;
    let href = corp::workspace_href(&g, path.as_deref()).map_err(corp_err)?;
    drop(g);
    navigate_main_title(&app, &href, "ERA Office · Corporate")?;
    Ok(href)
}

#[tauri::command]
pub fn corp_open_deep_link(
    app: AppHandle,
    state: State<'_, ConfigState>,
    raw: String,
) -> Result<String, String> {
    let target = corp::parse_open_url(&raw)
        .map_err(corp_err)?
        .ok_or_else(|| "not an era-office / http(s) link".to_string())?;
    let g = state.0.lock().map_err(|e| e.to_string())?;
    let href = corp::resolve_open(&g, &target).map_err(corp_err)?;
    drop(g);
    navigate_main_title(&app, &href, "ERA Office · Corporate")?;
    Ok(href)
}

#[tauri::command]
pub fn corp_parse_deep_link(raw: String) -> Result<Option<CorpOpenTarget>, String> {
    corp::parse_open_url(&raw).map_err(corp_err)
}

fn navigate_main(app: &AppHandle, href: &str) -> Result<(), String> {
    let url = href
        .parse::<url::Url>()
        .map_err(|e| format!("bad href: {e}"))?;
    let win = app
        .get_webview_window("main")
        .ok_or_else(|| "main window missing".to_string())?;
    win.navigate(url).map_err(|e| e.to_string())?;
    Ok(())
}

/// URL of Solo full Documents UI (local bridge).
#[tauri::command]
pub fn solo_docs_href(bridge: State<'_, BridgeUrlState>) -> Result<String, String> {
    let g = bridge.0.lock().map_err(|e| e.to_string())?;
    let h = g.as_ref().ok_or_else(|| "solo bridge not started".to_string())?;
    Ok(solo_bridge::docs_href(&h.base_url))
}

/// Navigate WebView to Solo product hub on loopback bridge.
#[tauri::command]
pub fn solo_hub_go(app: AppHandle, bridge: State<'_, BridgeUrlState>) -> Result<String, String> {
    let href = {
        let g = bridge.0.lock().map_err(|e| e.to_string())?;
        let h = g.as_ref().ok_or_else(|| "solo bridge not started".to_string())?;
        solo_bridge::hub_href(&h.base_url)
    };
    navigate_main_title(&app, &href, "ERA Office · Solo")?;
    Ok(href)
}

/// Navigate to SKU entry (hub for suite, direct product otherwise).
#[tauri::command]
pub fn solo_entry_go(
    app: AppHandle,
    bridge: State<'_, BridgeUrlState>,
    sku: State<'_, SkuState>,
) -> Result<String, String> {
    let sk = sku.0;
    let href = {
        let g = bridge.0.lock().map_err(|e| e.to_string())?;
        let h = g.as_ref().ok_or_else(|| "solo bridge not started".to_string())?;
        solo_bridge::sku_href(&h.base_url, sk)
    };
    navigate_main_title(&app, &href, sk.window_title())?;
    Ok(href)
}

/// Navigate WebView to full Documents SPA on loopback bridge.
#[tauri::command]
pub fn solo_docs_go(app: AppHandle, bridge: State<'_, BridgeUrlState>) -> Result<String, String> {
    let href = {
        let g = bridge.0.lock().map_err(|e| e.to_string())?;
        let h = g.as_ref().ok_or_else(|| "solo bridge not started".to_string())?;
        solo_bridge::docs_href(&h.base_url)
    };
    navigate_main(&app, &href)?;
    let _ = app
        .get_webview_window("main")
        .map(|w| w.set_title("ERA Documents · Solo"));
    Ok(href)
}

fn navigate_main_title(app: &AppHandle, href: &str, title: &str) -> Result<(), String> {
    navigate_main(app, href)?;
    if let Some(win) = app.get_webview_window("main") {
        let _ = win.set_title(title);
    }
    Ok(())
}

#[tauri::command]
pub fn doc_new(state: State<'_, AppState>) -> Result<DocSnapshot, String> {
    let mut g = state.0.lock().map_err(|e| e.to_string())?;
    g.new_doc();
    Ok(g.snapshot())
}

#[tauri::command]
pub fn doc_get(state: State<'_, AppState>) -> Result<DocSnapshot, String> {
    let g = state.0.lock().map_err(|e| e.to_string())?;
    Ok(g.snapshot())
}

#[tauri::command]
pub fn doc_apply_local(state: State<'_, AppState>, json: String) -> Result<DocSnapshot, String> {
    let mut g = state.0.lock().map_err(|e| e.to_string())?;
    g.set_doc_json(&json).map_err(map_err)?;
    Ok(g.snapshot())
}

#[tauri::command]
pub fn license_status(state: State<'_, AppState>) -> Result<LicenseStatus, String> {
    let g = state.0.lock().map_err(|e| e.to_string())?;
    Ok(g.license_status())
}

#[tauri::command]
pub fn license_set_token(
    docs: State<'_, AppState>,
    tables: State<'_, TablesState>,
    token: Option<String>,
) -> Result<LicenseStatus, String> {
    let mut d = docs.0.lock().map_err(|e| e.to_string())?;
    let mut t = tables.0.lock().map_err(|e| e.to_string())?;
    d.set_license_token(token.clone());
    t.set_license_token(token);
    Ok(d.license_status())
}

#[tauri::command]
pub fn doc_open_path(state: State<'_, AppState>, path: String) -> Result<DocSnapshot, String> {
    let mut g = state.0.lock().map_err(|e| e.to_string())?;
    g.open_path(PathBuf::from(path).as_path()).map_err(map_err)?;
    Ok(g.snapshot())
}

#[tauri::command]
pub fn doc_save_path(state: State<'_, AppState>, path: String) -> Result<DocSnapshot, String> {
    let mut g = state.0.lock().map_err(|e| e.to_string())?;
    g.save_to(PathBuf::from(path).as_path()).map_err(map_err)?;
    Ok(g.snapshot())
}

#[tauri::command]
pub fn doc_save(state: State<'_, AppState>) -> Result<DocSnapshot, String> {
    let mut g = state.0.lock().map_err(|e| e.to_string())?;
    g.save().map_err(map_err)?;
    Ok(g.snapshot())
}

#[tauri::command]
pub fn doc_export_docx_path(state: State<'_, AppState>, path: String) -> Result<(), String> {
    let g = state.0.lock().map_err(|e| e.to_string())?;
    g.export_docx_to(PathBuf::from(path).as_path())
        .map_err(map_err)
}

#[tauri::command]
pub async fn doc_open_dialog(
    app: tauri::AppHandle,
    state: State<'_, AppState>,
) -> Result<DocSnapshot, String> {
    let path = app
        .dialog()
        .file()
        .add_filter("ERA Documents", &["erad", "docx"])
        .add_filter("All", &["*"])
        .blocking_pick_file();
    let Some(FilePath::Path(p)) = path else {
        return Err("cancelled".into());
    };
    let mut g = state.0.lock().map_err(|e| e.to_string())?;
    g.open_path(&p).map_err(map_err)?;
    Ok(g.snapshot())
}

#[tauri::command]
pub async fn doc_save_as_dialog(
    app: tauri::AppHandle,
    state: State<'_, AppState>,
) -> Result<DocSnapshot, String> {
    let path = app
        .dialog()
        .file()
        .add_filter("ERA Document", &["erad"])
        .set_file_name("document.erad")
        .blocking_save_file();
    let Some(FilePath::Path(p)) = path else {
        return Err("cancelled".into());
    };
    let mut g = state.0.lock().map_err(|e| e.to_string())?;
    g.save_to(&p).map_err(map_err)?;
    Ok(g.snapshot())
}

#[tauri::command]
pub async fn doc_export_docx_dialog(
    app: tauri::AppHandle,
    state: State<'_, AppState>,
) -> Result<(), String> {
    let path = app
        .dialog()
        .file()
        .add_filter("Word", &["docx"])
        .set_file_name("export.docx")
        .blocking_save_file();
    let Some(FilePath::Path(p)) = path else {
        return Err("cancelled".into());
    };
    let g = state.0.lock().map_err(|e| e.to_string())?;
    g.export_docx_to(&p).map_err(map_err)
}

#[tauri::command]
pub async fn doc_import_docx_dialog(
    app: tauri::AppHandle,
    state: State<'_, AppState>,
) -> Result<DocSnapshot, String> {
    let path = app
        .dialog()
        .file()
        .add_filter("Word", &["docx"])
        .blocking_pick_file();
    let Some(FilePath::Path(p)) = path else {
        return Err("cancelled".into());
    };
    let bytes = std::fs::read(&p).map_err(|e| e.to_string())?;
    let mut g = state.0.lock().map_err(|e| e.to_string())?;
    g.import_docx_bytes(&bytes).map_err(map_err)?;
    Ok(g.snapshot())
}

#[tauri::command]
pub fn sheet_new(state: State<'_, TablesState>) -> Result<SheetSnapshot, String> {
    let mut g = state.0.lock().map_err(|e| e.to_string())?;
    g.new_sheet();
    Ok(g.snapshot())
}

#[tauri::command]
pub fn sheet_get(state: State<'_, TablesState>) -> Result<SheetSnapshot, String> {
    let g = state.0.lock().map_err(|e| e.to_string())?;
    Ok(g.snapshot())
}

#[tauri::command]
pub fn sheet_apply_local(
    state: State<'_, TablesState>,
    json: String,
) -> Result<SheetSnapshot, String> {
    let mut g = state.0.lock().map_err(|e| e.to_string())?;
    g.set_sheet_json(&json).map_err(map_tables_err)?;
    Ok(g.snapshot())
}

#[tauri::command]
pub fn sheet_license_status(state: State<'_, TablesState>) -> Result<LicenseStatus, String> {
    let g = state.0.lock().map_err(|e| e.to_string())?;
    Ok(g.license_status())
}

#[tauri::command]
pub fn sheet_open_path(
    state: State<'_, TablesState>,
    path: String,
) -> Result<SheetSnapshot, String> {
    let mut g = state.0.lock().map_err(|e| e.to_string())?;
    g.open_path(PathBuf::from(path).as_path())
        .map_err(map_tables_err)?;
    Ok(g.snapshot())
}

#[tauri::command]
pub fn sheet_save_path(
    state: State<'_, TablesState>,
    path: String,
) -> Result<SheetSnapshot, String> {
    let mut g = state.0.lock().map_err(|e| e.to_string())?;
    g.save_to(PathBuf::from(path).as_path())
        .map_err(map_tables_err)?;
    Ok(g.snapshot())
}

#[tauri::command]
pub fn sheet_save(state: State<'_, TablesState>) -> Result<SheetSnapshot, String> {
    let mut g = state.0.lock().map_err(|e| e.to_string())?;
    g.save().map_err(map_tables_err)?;
    Ok(g.snapshot())
}

#[tauri::command]
pub async fn sheet_open_dialog(
    app: tauri::AppHandle,
    state: State<'_, TablesState>,
) -> Result<SheetSnapshot, String> {
    let path = app
        .dialog()
        .file()
        .add_filter("ERA Tables", &["erat", "xlsx", "ods"])
        .add_filter("All", &["*"])
        .blocking_pick_file();
    let Some(FilePath::Path(p)) = path else {
        return Err("cancelled".into());
    };
    let mut g = state.0.lock().map_err(|e| e.to_string())?;
    g.open_path(&p).map_err(map_tables_err)?;
    Ok(g.snapshot())
}

#[tauri::command]
pub async fn sheet_save_as_dialog(
    app: tauri::AppHandle,
    state: State<'_, TablesState>,
) -> Result<SheetSnapshot, String> {
    let path = app
        .dialog()
        .file()
        .add_filter("ERA Table", &["erat"])
        .set_file_name("sheet.erat")
        .blocking_save_file();
    let Some(FilePath::Path(p)) = path else {
        return Err("cancelled".into());
    };
    let mut g = state.0.lock().map_err(|e| e.to_string())?;
    g.save_to(&p).map_err(map_tables_err)?;
    Ok(g.snapshot())
}

#[tauri::command]
pub async fn sheet_export_xlsx_dialog(
    app: tauri::AppHandle,
    state: State<'_, TablesState>,
) -> Result<(), String> {
    let path = app
        .dialog()
        .file()
        .add_filter("Excel", &["xlsx"])
        .set_file_name("export.xlsx")
        .blocking_save_file();
    let Some(FilePath::Path(p)) = path else {
        return Err("cancelled".into());
    };
    let g = state.0.lock().map_err(|e| e.to_string())?;
    g.export_xlsx_to(&p).map_err(map_tables_err)
}

#[tauri::command]
pub async fn sheet_import_xlsx_dialog(
    app: tauri::AppHandle,
    state: State<'_, TablesState>,
) -> Result<SheetSnapshot, String> {
    let path = app
        .dialog()
        .file()
        .add_filter("Excel", &["xlsx"])
        .blocking_pick_file();
    let Some(FilePath::Path(p)) = path else {
        return Err("cancelled".into());
    };
    let bytes = std::fs::read(&p).map_err(|e| e.to_string())?;
    let mut g = state.0.lock().map_err(|e| e.to_string())?;
    g.import_xlsx_bytes(&bytes).map_err(map_tables_err)?;
    Ok(g.snapshot())
}
