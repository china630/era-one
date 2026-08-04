//! ERA Office desktop — Solo hub (bridge) + SKU modes + Corporate (S4).
//! Targets: Browser ❌ · Solo ✅ · Corporate ✅

pub mod commands;
pub mod config;
pub mod corp;
pub mod license;
pub mod sku;
pub mod solo;
pub mod solo_bridge;
pub mod solo_pres;
pub mod solo_projects;
pub mod solo_tables;

use std::sync::{Arc, Mutex};

use commands::{AppState, BridgeUrlState, ConfigState, PresState, ProjectsState, TablesState};
use config::Profile;
use sku::Sku;
use solo::SoloState;
use solo_bridge::BridgeState;
use solo_pres::SoloPresState;
use solo_projects::SoloProjectsState;
use solo_tables::SoloTablesState;
use tauri::Manager;

fn try_navigate(app: &tauri::AppHandle, href: &str, title: &str) {
    let Ok(url) = href.parse::<url::Url>() else {
        return;
    };
    if let Some(win) = app.get_webview_window("main") {
        let _ = win.navigate(url);
        let _ = win.set_title(title);
    }
}

fn handle_open_payload(app: &tauri::AppHandle, args: &[String]) {
    let cfg = {
        let st = app.state::<ConfigState>();
        let cfg = st.0.lock().expect("config").clone();
        cfg
    };
    let bridge = {
        let st = app.state::<BridgeUrlState>();
        let bridge = st.0.lock().expect("bridge").clone();
        bridge
    };
    let Some(bh) = bridge else {
        return;
    };

    // File path argv → open in matching Solo product
    if let Some(path) = corp::file_path_from_args(args.iter().cloned()) {
        if let Some(file_sku) = Sku::from_path_ext(&path) {
            let p = std::path::PathBuf::from(&path);
            match file_sku {
                Sku::Docs => {
                    if let Ok(mut g) = bh.state.solo.lock() {
                        let _ = g.open_path(&p);
                    }
                }
                Sku::Tables => {
                    if let Ok(mut g) = bh.state.tables.lock() {
                        let _ = g.open_path(&p);
                    }
                }
                Sku::Presentations => {
                    if let Ok(mut g) = bh.state.pres.lock() {
                        let _ = g.open_path(&p);
                    }
                }
                Sku::Projects => {
                    if let Ok(mut g) = bh.state.projects.lock() {
                        let _ = g.open_path(&p);
                    }
                }
                Sku::Suite => {}
            }
            let href = solo_bridge::sku_href(&bh.base_url, file_sku);
            try_navigate(app, &href, file_sku.window_title());
            return;
        }
    }

    if let Ok(Some(target)) = corp::open_target_from_args(args.iter().cloned()) {
        if cfg.profile == Profile::Corporate {
            if let Ok(href) = corp::resolve_open(&cfg, &target) {
                try_navigate(app, &href, "ERA Office · Corporate");
                return;
            }
        }
        // Solo: route deep-link path to product
        let path = target.href.clone();
        let link_sku = Sku::from_open_path(&path).unwrap_or(Sku::Suite);
        let href = if path.starts_with("http://") || path.starts_with("https://") {
            path
        } else {
            solo_bridge::sku_href(&bh.base_url, link_sku)
        };
        try_navigate(app, &href, link_sku.window_title());
    }
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let cfg = config::load().unwrap_or_default();
    let sku = Sku::from_env_and_args();
    let pending_args: Vec<String> = std::env::args().collect();
    let solo = Arc::new(Mutex::new(SoloState::default()));
    let tables = Arc::new(Mutex::new(SoloTablesState::default()));
    let pres = Arc::new(Mutex::new(SoloPresState::default()));
    let projects = Arc::new(Mutex::new(SoloProjectsState::default()));

    tauri::Builder::default()
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_single_instance::init(|app, argv, _cwd| {
            handle_open_payload(app, &argv);
            if let Some(win) = app.get_webview_window("main") {
                let _ = win.set_focus();
            }
        }))
        .manage(AppState(solo.clone()))
        .manage(TablesState(tables.clone()))
        .manage(PresState(pres.clone()))
        .manage(ProjectsState(projects.clone()))
        .manage(ConfigState(Mutex::new(cfg)))
        .manage(BridgeUrlState(Mutex::new(None)))
        .manage(commands::SkuState(sku))
        .invoke_handler(tauri::generate_handler![
            commands::config_get,
            commands::config_set,
            commands::sku_get,
            commands::corp_go,
            commands::corp_open_deep_link,
            commands::corp_parse_deep_link,
            commands::solo_docs_href,
            commands::solo_docs_go,
            commands::solo_hub_go,
            commands::solo_entry_go,
            commands::doc_new,
            commands::doc_get,
            commands::doc_apply_local,
            commands::doc_open_path,
            commands::doc_save_path,
            commands::doc_save,
            commands::doc_export_docx_path,
            commands::doc_open_dialog,
            commands::doc_save_as_dialog,
            commands::doc_export_docx_dialog,
            commands::doc_import_docx_dialog,
            commands::license_status,
            commands::license_set_token,
            commands::sheet_new,
            commands::sheet_get,
            commands::sheet_apply_local,
            commands::sheet_license_status,
            commands::sheet_open_path,
            commands::sheet_save_path,
            commands::sheet_save,
            commands::sheet_open_dialog,
            commands::sheet_save_as_dialog,
            commands::sheet_export_xlsx_dialog,
            commands::sheet_import_xlsx_dialog,
        ])
        .setup(move |app| {
            let roots = solo_bridge::resolve_asset_roots();
            let bridge_state = BridgeState {
                solo: solo.clone(),
                tables: tables.clone(),
                pres: pres.clone(),
                projects: projects.clone(),
                docs_web: roots.docs_web,
                tables_web: roots.tables_web,
                presentations_web: roots.presentations_web,
                projects_web: roots.projects_web,
                office_assets: roots.office_assets,
                boot_js: roots.boot_js,
                skin_css: roots.skin_css,
            };
            let bh = tauri::async_runtime::block_on(solo_bridge::start(bridge_state))
                .map_err(|e| format!("solo_bridge: {e}"))?;
            {
                let st = app.state::<BridgeUrlState>();
                *st.0.lock().expect("bridge") = Some(bh);
            }

            let state = app.state::<ConfigState>();
            let cfg = state.0.lock().expect("config").clone();
            drop(state);

            // First-launch open from argv (file / deep link)
            handle_open_payload(&app.handle(), &pending_args);

            if cfg.corporate_ready() {
                if let Ok(href) = corp::workspace_href(&cfg, None) {
                    // Don't override if handle_open_payload already navigated Corporate
                    if pending_args.iter().all(|a| {
                        !a.starts_with("era-office:") && !a.starts_with("http")
                    }) {
                        try_navigate(&app.handle(), &href, "ERA Office · Corporate");
                    }
                }
            }
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running ERA Office desktop");
}
