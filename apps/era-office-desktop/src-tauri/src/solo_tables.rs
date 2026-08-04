//! Solo Tables session + path I/O (S5/B2). No Drive / WS.
//! Targets: Browser ❌ · Solo ✅ · Corporate ❌

use std::path::{Path, PathBuf};

use era_tables_core::calc::recalc;
use era_tables_core::convert::{export_xlsx, import_xlsx};
use era_tables_core::convert_ods::{export_ods, import_ods};
use era_tables_core::model::EratSheet;
use serde::{Deserialize, Serialize};
use thiserror::Error;

use crate::license::{self, LicenseMode, LicenseStatus, DEMO_CELL_CAP};

#[derive(Debug, Error)]
pub enum SoloTablesError {
    #[error("{0}")]
    Msg(String),
    #[error(transparent)]
    Io(#[from] std::io::Error),
    #[error(transparent)]
    Json(#[from] serde_json::Error),
    #[error(transparent)]
    Any(#[from] anyhow::Error),
    #[error("demo limit: sheet has {cells} nonempty cells (cap {cap}); activate license to save")]
    DemoLimit { cells: usize, cap: usize },
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SheetSnapshot {
    pub path: Option<String>,
    pub dirty: bool,
    pub sheet: EratSheet,
    pub license: LicenseStatus,
    pub nonempty_cells: usize,
}

#[derive(Debug)]
pub struct SoloTablesState {
    pub path: Option<PathBuf>,
    pub sheet: EratSheet,
    pub dirty: bool,
    pub license_token: Option<String>,
}

impl Default for SoloTablesState {
    fn default() -> Self {
        Self {
            path: None,
            sheet: EratSheet::empty(),
            dirty: false,
            license_token: std::env::var("ERA_SOLO_LICENSE").ok().filter(|s| !s.is_empty()),
        }
    }
}

/// Count nonempty cells across all tabs (value or formula).
pub fn nonempty_cell_count(sheet: &EratSheet) -> usize {
    let mut n = 0usize;
    for tab in &sheet.sheets {
        n += tab
            .cells
            .values()
            .filter(|c| !c.value.trim().is_empty() || !c.formula.trim().is_empty())
            .count();
    }
    if sheet.sheets.is_empty() {
        n += sheet
            .cells
            .values()
            .filter(|c| !c.value.trim().is_empty() || !c.formula.trim().is_empty())
            .count();
    }
    n
}

impl SoloTablesState {
    pub fn license_status(&self) -> LicenseStatus {
        license::status_tables(self.license_token.as_deref(), nonempty_cell_count(&self.sheet))
    }

    pub fn snapshot(&self) -> SheetSnapshot {
        let nonempty_cells = nonempty_cell_count(&self.sheet);
        SheetSnapshot {
            path: self.path.as_ref().map(|p| p.display().to_string()),
            dirty: self.dirty,
            sheet: self.sheet.clone(),
            license: self.license_status(),
            nonempty_cells,
        }
    }

    pub fn new_sheet(&mut self) {
        self.path = None;
        self.sheet = EratSheet::empty();
        self.dirty = false;
    }

    pub fn set_sheet_json(&mut self, json: &str) -> Result<(), SoloTablesError> {
        let mut sheet: EratSheet = serde_json::from_str(json)?;
        sheet.normalize_tabs();
        recalc(&mut sheet);
        sheet.flush_active_to_tab();
        self.sheet = sheet;
        self.dirty = true;
        Ok(())
    }

    pub fn open_path(&mut self, path: &Path) -> Result<(), SoloTablesError> {
        let bytes = std::fs::read(path)?;
        let ext = path
            .extension()
            .and_then(|e| e.to_str())
            .unwrap_or("")
            .to_ascii_lowercase();
        let mut sheet = match ext.as_str() {
            "xlsx" => import_xlsx(&bytes)?,
            "ods" => import_ods(&bytes)?,
            "erat" | "json" | "" => {
                let mut s: EratSheet = serde_json::from_slice(&bytes)?;
                s.normalize_tabs();
                recalc(&mut s);
                s.flush_active_to_tab();
                s
            }
            other => {
                return Err(SoloTablesError::Msg(format!(
                    "unsupported extension .{other}; use .erat, .xlsx, or .ods"
                )));
            }
        };
        sheet.normalize_tabs();
        self.sheet = sheet;
        self.path = if matches!(ext.as_str(), "xlsx" | "ods") {
            None
        } else {
            Some(path.to_path_buf())
        };
        self.dirty = matches!(ext.as_str(), "xlsx" | "ods");
        Ok(())
    }

    pub fn import_xlsx_bytes(&mut self, bytes: &[u8]) -> Result<(), SoloTablesError> {
        let mut sheet = import_xlsx(bytes)?;
        sheet.normalize_tabs();
        self.sheet = sheet;
        self.path = None;
        self.dirty = true;
        Ok(())
    }

    fn assert_can_save(&self) -> Result<(), SoloTablesError> {
        let st = self.license_status();
        if st.mode == LicenseMode::Licensed {
            return Ok(());
        }
        let n = nonempty_cell_count(&self.sheet);
        if n > DEMO_CELL_CAP {
            return Err(SoloTablesError::DemoLimit {
                cells: n,
                cap: DEMO_CELL_CAP,
            });
        }
        Ok(())
    }

    pub fn save_to(&mut self, path: &Path) -> Result<(), SoloTablesError> {
        self.assert_can_save()?;
        self.sheet.normalize_tabs();
        self.sheet.flush_active_to_tab();
        let json = serde_json::to_string_pretty(&self.sheet)?;
        if let Some(parent) = path.parent() {
            std::fs::create_dir_all(parent)?;
        }
        std::fs::write(path, json.as_bytes())?;
        self.path = Some(path.to_path_buf());
        self.dirty = false;
        Ok(())
    }

    pub fn save(&mut self) -> Result<(), SoloTablesError> {
        let path = self
            .path
            .clone()
            .ok_or_else(|| SoloTablesError::Msg("no path — use Save As".into()))?;
        self.save_to(&path)
    }

    pub fn export_xlsx_to(&self, path: &Path) -> Result<(), SoloTablesError> {
        let bytes = export_xlsx(&self.sheet)?;
        if let Some(parent) = path.parent() {
            std::fs::create_dir_all(parent)?;
        }
        std::fs::write(path, bytes)?;
        Ok(())
    }

    pub fn export_ods_to(&self, path: &Path) -> Result<(), SoloTablesError> {
        let bytes = export_ods(&self.sheet)?;
        if let Some(parent) = path.parent() {
            std::fs::create_dir_all(parent)?;
        }
        std::fs::write(path, bytes)?;
        Ok(())
    }

    pub fn set_license_token(&mut self, token: Option<String>) {
        self.license_token = token.filter(|s| !s.is_empty());
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::license::{mint_test_token, DEMO_CELL_CAP, MODULE_TABLES_SOLO};

    fn fat_sheet(n: usize) -> EratSheet {
        let mut s = EratSheet::empty();
        for i in 0..n {
            let addr = format!("A{}", i + 1);
            s.set_cell(&addr, format!("v{i}"), "");
        }
        s.flush_active_to_tab();
        s
    }

    #[test]
    fn erat_roundtrip_path() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("book.erat");
        let mut st = SoloTablesState::default();
        st.sheet = fat_sheet(2);
        st.save_to(&path).unwrap();
        let mut st2 = SoloTablesState::default();
        st2.open_path(&path).unwrap();
        assert_eq!(nonempty_cell_count(&st2.sheet), 2);
        assert_eq!(st2.path.as_ref().unwrap(), &path);
    }

    #[test]
    fn demo_cells_over_cap_blocks_save() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("big.erat");
        let mut st = SoloTablesState::default();
        st.sheet = fat_sheet(DEMO_CELL_CAP + 1);
        let err = st.save_to(&path).unwrap_err();
        assert!(matches!(err, SoloTablesError::DemoLimit { .. }));
    }

    #[test]
    fn licensed_can_save_over_cap() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("big.erat");
        let tok = mint_test_token(&[MODULE_TABLES_SOLO]);
        let mut st = SoloTablesState::default();
        st.set_license_token(Some(tok));
        st.sheet = fat_sheet(DEMO_CELL_CAP + 3);
        st.save_to(&path).unwrap();
        assert!(!st.dirty);
    }

    #[test]
    fn xlsx_export_import_smoke() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("out.xlsx");
        let mut st = SoloTablesState::default();
        st.sheet = fat_sheet(1);
        st.sheet.set_cell("B1", "", "=A1");
        st.sheet.flush_active_to_tab();
        st.export_xlsx_to(&path).unwrap();
        let bytes = std::fs::read(&path).unwrap();
        st.import_xlsx_bytes(&bytes).unwrap();
        assert!(nonempty_cell_count(&st.sheet) >= 1);
    }
}
