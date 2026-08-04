//! Solo Presentations session + path I/O (B2). No Drive / WS.
//! Targets: Browser ❌ · Solo ✅ · Corporate ❌

use std::path::{Path, PathBuf};

use era_pres_core::convert::{export_pptx, import_pptx};
use era_pres_core::convert_odp::export_odp;
use era_pres_core::model::ErapDeck;
use serde::{Deserialize, Serialize};
use thiserror::Error;

use crate::license::{self, LicenseStatus, DEMO_SLIDE_CAP};

#[derive(Debug, Error)]
pub enum SoloPresError {
    #[error("{0}")]
    Msg(String),
    #[error(transparent)]
    Io(#[from] std::io::Error),
    #[error(transparent)]
    Json(#[from] serde_json::Error),
    #[error(transparent)]
    Any(#[from] anyhow::Error),
    #[error("demo limit: deck has {slides} slides (cap {cap}); activate license to save")]
    DemoLimit { slides: usize, cap: usize },
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PresSnapshot {
    pub path: Option<String>,
    pub dirty: bool,
    pub deck: ErapDeck,
    pub license: LicenseStatus,
    pub slide_count: usize,
}

#[derive(Debug)]
pub struct SoloPresState {
    pub path: Option<PathBuf>,
    pub deck: ErapDeck,
    pub dirty: bool,
    pub license_token: Option<String>,
}

impl Default for SoloPresState {
    fn default() -> Self {
        Self {
            path: None,
            deck: ErapDeck::empty(),
            dirty: false,
            license_token: std::env::var("ERA_SOLO_LICENSE")
                .ok()
                .filter(|s| !s.is_empty()),
        }
    }
}

impl SoloPresState {
    pub fn license_status(&self) -> LicenseStatus {
        license::status_pres(self.license_token.as_deref(), self.deck.slides.len())
    }

    pub fn snapshot(&self) -> PresSnapshot {
        PresSnapshot {
            path: self.path.as_ref().map(|p| p.display().to_string()),
            dirty: self.dirty,
            deck: self.deck.clone(),
            license: self.license_status(),
            slide_count: self.deck.slides.len(),
        }
    }

    pub fn new_deck(&mut self) {
        self.path = None;
        self.deck = ErapDeck::empty();
        self.dirty = false;
    }

    pub fn set_deck(&mut self, deck: ErapDeck) {
        self.deck = deck;
        self.dirty = true;
    }

    pub fn set_deck_json(&mut self, json: &str) -> Result<(), SoloPresError> {
        let deck: ErapDeck = serde_json::from_str(json)?;
        if deck.slides.is_empty() {
            return Err(SoloPresError::Msg("deck must have at least one slide".into()));
        }
        self.deck = deck;
        self.dirty = true;
        Ok(())
    }

    pub fn open_path(&mut self, path: &Path) -> Result<(), SoloPresError> {
        let bytes = std::fs::read(path)?;
        let ext = path
            .extension()
            .and_then(|e| e.to_str())
            .unwrap_or("")
            .to_ascii_lowercase();
        let mut deck = match ext.as_str() {
            "pptx" => import_pptx(&bytes)?,
            "erap" | "json" | "" => serde_json::from_slice(&bytes)?,
            "odp" => {
                return Err(SoloPresError::Msg(
                    "open .odp not supported in Solo yet; use .erap or .pptx".into(),
                ))
            }
            _ => {
                return Err(SoloPresError::Msg(format!(
                    "unsupported extension .{ext}"
                )))
            }
        };
        if deck.slides.is_empty() {
            deck.slides.push(era_pres_core::model::ErapSlide::new_blank());
        }
        if deck.name.is_empty() {
            deck.name = path
                .file_name()
                .and_then(|n| n.to_str())
                .unwrap_or("deck.erap")
                .to_string();
        }
        self.deck = deck;
        self.path = Some(path.to_path_buf());
        self.dirty = false;
        Ok(())
    }

    fn gate_save(&self) -> Result<(), SoloPresError> {
        let st = self.license_status();
        if !st.can_save {
            return Err(SoloPresError::DemoLimit {
                slides: self.deck.slides.len(),
                cap: DEMO_SLIDE_CAP,
            });
        }
        Ok(())
    }

    pub fn save_to(&mut self, path: &Path) -> Result<(), SoloPresError> {
        self.gate_save()?;
        let ext = path
            .extension()
            .and_then(|e| e.to_str())
            .unwrap_or("erap")
            .to_ascii_lowercase();
        match ext.as_str() {
            "pptx" => {
                let bytes = export_pptx(&self.deck)?;
                std::fs::write(path, bytes)?;
            }
            "odp" => {
                let bytes = export_odp(&self.deck)?;
                std::fs::write(path, bytes)?;
            }
            _ => {
                let mut deck = self.deck.clone();
                if deck.name.is_empty() {
                    deck.name = path
                        .file_name()
                        .and_then(|n| n.to_str())
                        .unwrap_or("deck.erap")
                        .to_string();
                }
                let bytes = serde_json::to_vec_pretty(&deck)?;
                std::fs::write(path, bytes)?;
                self.deck = deck;
            }
        }
        self.path = Some(path.to_path_buf());
        self.dirty = false;
        Ok(())
    }

    pub fn save(&mut self) -> Result<(), SoloPresError> {
        let path = self
            .path
            .clone()
            .ok_or_else(|| SoloPresError::Msg("no path; use Save As".into()))?;
        self.save_to(&path)
    }

    pub fn export_pptx_bytes(&self) -> Result<Vec<u8>, SoloPresError> {
        Ok(export_pptx(&self.deck)?)
    }

    pub fn export_odp_bytes(&self) -> Result<Vec<u8>, SoloPresError> {
        Ok(export_odp(&self.deck)?)
    }

    pub fn import_pptx_bytes(&mut self, data: &[u8], name: Option<String>) -> Result<(), SoloPresError> {
        let mut deck = import_pptx(data)?;
        if let Some(n) = name {
            deck.name = n;
        }
        if deck.slides.is_empty() {
            deck.slides.push(era_pres_core::model::ErapSlide::new_blank());
        }
        self.path = None;
        self.deck = deck;
        self.dirty = true;
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::tempdir;

    #[test]
    fn erap_roundtrip() {
        let dir = tempdir().unwrap();
        let path = dir.path().join("t.erap");
        let mut st = SoloPresState::default();
        st.deck.slides[0].set_title_plain("Hello");
        st.save_to(&path).unwrap();
        let mut st2 = SoloPresState::default();
        st2.open_path(&path).unwrap();
        assert_eq!(st2.deck.slides[0].title(), "Hello");
    }
}
