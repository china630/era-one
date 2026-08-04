//! Solo Documents session + path I/O (S3/B1). No Drive / WS.
//! Targets: Browser ❌ · Solo ✅ · Corporate ❌

use std::path::{Path, PathBuf};

use era_docs_core::canonical::to_canonical_json;
use era_docs_core::convert::{export_docx, import_docx};
use era_docs_core::model::EradDocument;
use serde::{Deserialize, Serialize};
use thiserror::Error;

use crate::license::{self, LicenseMode, LicenseStatus, DEMO_BLOCK_CAP};

#[derive(Debug, Error)]
pub enum SoloError {
    #[error("{0}")]
    Msg(String),
    #[error(transparent)]
    Io(#[from] std::io::Error),
    #[error(transparent)]
    Json(#[from] serde_json::Error),
    #[error(transparent)]
    Any(#[from] anyhow::Error),
    #[error("demo limit: document has {blocks} blocks (cap {cap}); activate license to save")]
    DemoLimit { blocks: usize, cap: usize },
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DocSnapshot {
    pub path: Option<String>,
    pub dirty: bool,
    pub doc: EradDocument,
    pub license: LicenseStatus,
}

#[derive(Debug)]
pub struct SoloState {
    pub path: Option<PathBuf>,
    pub doc: EradDocument,
    pub dirty: bool,
    pub license_token: Option<String>,
    /// Display title (Drive name stub for full UI).
    pub title: String,
}

impl Default for SoloState {
    fn default() -> Self {
        Self {
            path: None,
            doc: EradDocument::empty(),
            dirty: false,
            license_token: std::env::var("ERA_SOLO_LICENSE").ok().filter(|s| !s.is_empty()),
            title: "Untitled.erad".into(),
        }
    }
}

impl SoloState {
    pub fn license_status(&self) -> LicenseStatus {
        license::status(self.license_token.as_deref(), self.doc.blocks.len())
    }

    pub fn snapshot(&self) -> DocSnapshot {
        DocSnapshot {
            path: self.path.as_ref().map(|p| p.display().to_string()),
            dirty: self.dirty,
            doc: self.doc.clone(),
            license: self.license_status(),
        }
    }

    pub fn new_doc(&mut self) {
        self.path = None;
        self.doc = EradDocument::empty();
        self.dirty = false;
        self.title = "Untitled.erad".into();
    }

    pub fn set_doc_json(&mut self, json: &str) -> Result<(), SoloError> {
        self.doc = serde_json::from_str(json)?;
        self.dirty = true;
        Ok(())
    }

    pub fn open_path(&mut self, path: &Path) -> Result<(), SoloError> {
        let bytes = std::fs::read(path)?;
        let ext = path
            .extension()
            .and_then(|e| e.to_str())
            .unwrap_or("")
            .to_ascii_lowercase();
        self.doc = match ext.as_str() {
            "docx" => import_docx(&bytes)?,
            "erad" | "json" | "" => serde_json::from_slice(&bytes)?,
            other => {
                return Err(SoloError::Msg(format!(
                    "unsupported extension .{other}; use .erad or .docx"
                )));
            }
        };
        self.path = if ext == "docx" {
            None // import → Save As .erad
        } else {
            Some(path.to_path_buf())
        };
        self.dirty = ext == "docx";
        if let Some(name) = path.file_name() {
            self.title = name.to_string_lossy().into_owned();
        }
        Ok(())
    }

    pub fn import_docx_bytes(&mut self, bytes: &[u8]) -> Result<(), SoloError> {
        self.doc = import_docx(bytes)?;
        self.path = None;
        self.dirty = true;
        Ok(())
    }

    fn assert_can_save(&self) -> Result<(), SoloError> {
        let st = self.license_status();
        if st.mode == LicenseMode::Licensed {
            return Ok(());
        }
        let n = self.doc.blocks.len();
        if n > DEMO_BLOCK_CAP {
            return Err(SoloError::DemoLimit {
                blocks: n,
                cap: DEMO_BLOCK_CAP,
            });
        }
        Ok(())
    }

    pub fn save_to(&mut self, path: &Path) -> Result<(), SoloError> {
        self.assert_can_save()?;
        let json = to_canonical_json(&self.doc)?;
        if let Some(parent) = path.parent() {
            std::fs::create_dir_all(parent)?;
        }
        std::fs::write(path, json.as_bytes())?;
        self.path = Some(path.to_path_buf());
        self.dirty = false;
        if let Some(name) = path.file_name() {
            self.title = name.to_string_lossy().into_owned();
        }
        Ok(())
    }

    pub fn save(&mut self) -> Result<(), SoloError> {
        let path = self
            .path
            .clone()
            .ok_or_else(|| SoloError::Msg("no path — use Save As".into()))?;
        self.save_to(&path)
    }

    pub fn export_docx_to(&self, path: &Path) -> Result<(), SoloError> {
        let bytes = export_docx(&self.doc)?;
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
    use crate::license::{mint_test_token, MODULE_SOLO, DEMO_BLOCK_CAP};
    use era_docs_core::model::Block;

    fn fat_doc(n: usize) -> EradDocument {
        let mut doc = EradDocument::empty();
        doc.blocks.clear();
        for i in 0..n {
            doc.blocks.push(Block::paragraph(format!("b{i}"), format!("p{i}")));
        }
        doc
    }

    #[test]
    fn erad_roundtrip_path() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("memo.erad");
        let mut s = SoloState::default();
        s.doc = fat_doc(2);
        s.save_to(&path).unwrap();
        let mut s2 = SoloState::default();
        s2.open_path(&path).unwrap();
        assert_eq!(s2.doc.blocks.len(), 2);
        assert_eq!(s2.path.as_ref().unwrap(), &path);
    }

    #[test]
    fn demo_blocks_over_cap_blocks_save() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("big.erad");
        let mut s = SoloState::default();
        s.doc = fat_doc(DEMO_BLOCK_CAP + 1);
        let err = s.save_to(&path).unwrap_err();
        assert!(matches!(err, SoloError::DemoLimit { .. }));
    }

    #[test]
    fn licensed_can_save_over_cap() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("big.erad");
        let tok = mint_test_token(&[MODULE_SOLO]);
        let mut s = SoloState::default();
        s.set_license_token(Some(tok));
        s.doc = fat_doc(DEMO_BLOCK_CAP + 3);
        s.save_to(&path).unwrap();
        assert!(!s.dirty);
    }

    #[test]
    fn docx_export_import_smoke() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("out.docx");
        let mut s = SoloState::default();
        s.doc = fat_doc(1);
        s.export_docx_to(&path).unwrap();
        let bytes = std::fs::read(&path).unwrap();
        s.import_docx_bytes(&bytes).unwrap();
        assert!(!s.doc.blocks.is_empty());
    }
}
