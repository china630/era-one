//! Solo Projects session + `.eraj` path I/O (C2). No Postgres.
//! Targets: Browser ❌ · Solo ✅ · Corporate ❌

use std::path::{Path, PathBuf};

use era_projects_core::{ErajBoard, ProjectTask};
use serde::{Deserialize, Serialize};
use thiserror::Error;

use crate::license::{self, LicenseStatus, DEMO_TASK_CAP};

#[derive(Debug, Error)]
pub enum SoloProjectsError {
    #[error("{0}")]
    Msg(String),
    #[error(transparent)]
    Io(#[from] std::io::Error),
    #[error(transparent)]
    Json(#[from] serde_json::Error),
    #[error(transparent)]
    Any(#[from] anyhow::Error),
    #[error("demo limit: board has {tasks} tasks (cap {cap}); activate license to save")]
    DemoLimit { tasks: usize, cap: usize },
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProjectsSnapshot {
    pub path: Option<String>,
    pub dirty: bool,
    pub board: ErajBoard,
    pub license: LicenseStatus,
    pub task_count: usize,
}

#[derive(Debug)]
pub struct SoloProjectsState {
    pub path: Option<PathBuf>,
    pub board: ErajBoard,
    pub dirty: bool,
    pub license_token: Option<String>,
}

impl Default for SoloProjectsState {
    fn default() -> Self {
        Self {
            path: None,
            board: ErajBoard::empty(),
            dirty: false,
            license_token: std::env::var("ERA_SOLO_LICENSE")
                .ok()
                .filter(|s| !s.is_empty()),
        }
    }
}

impl SoloProjectsState {
    pub fn license_status(&self) -> LicenseStatus {
        license::status_projects(self.license_token.as_deref(), self.board.tasks.len())
    }

    pub fn snapshot(&self) -> ProjectsSnapshot {
        ProjectsSnapshot {
            path: self.path.as_ref().map(|p| p.display().to_string()),
            dirty: self.dirty,
            board: self.board.clone(),
            license: self.license_status(),
            task_count: self.board.tasks.len(),
        }
    }

    pub fn new_board(&mut self) {
        self.path = None;
        self.board = ErajBoard::empty();
        self.dirty = false;
    }

    pub fn open_path(&mut self, path: &Path) -> Result<(), SoloProjectsError> {
        let board = ErajBoard::load_path(path)?;
        self.board = board;
        self.path = Some(path.to_path_buf());
        self.dirty = false;
        Ok(())
    }

    fn gate_save(&self) -> Result<(), SoloProjectsError> {
        let st = self.license_status();
        if !st.can_save {
            return Err(SoloProjectsError::DemoLimit {
                tasks: self.board.tasks.len(),
                cap: DEMO_TASK_CAP,
            });
        }
        Ok(())
    }

    pub fn save_to(&mut self, path: &Path) -> Result<(), SoloProjectsError> {
        self.gate_save()?;
        if self.board.name.is_empty() {
            self.board.name = path
                .file_name()
                .and_then(|n| n.to_str())
                .unwrap_or("board.eraj")
                .to_string();
        }
        self.board.save_path(path)?;
        self.path = Some(path.to_path_buf());
        self.dirty = false;
        Ok(())
    }

    pub fn save(&mut self) -> Result<(), SoloProjectsError> {
        let path = self
            .path
            .clone()
            .ok_or_else(|| SoloProjectsError::Msg("no path; use Save As".into()))?;
        self.save_to(&path)
    }

    pub fn upsert_task(&mut self, task: ProjectTask) {
        self.board.upsert_task(task);
        self.dirty = true;
    }

    pub fn remove_task(&mut self, id: &str) -> bool {
        let ok = self.board.remove_task(id);
        if ok {
            self.dirty = true;
        }
        ok
    }

    pub fn set_name(&mut self, name: &str) {
        self.board.name = name.to_string();
        self.dirty = true;
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::tempdir;

    #[test]
    fn eraj_roundtrip() {
        let dir = tempdir().unwrap();
        let path = dir.path().join("b.eraj");
        let mut st = SoloProjectsState::default();
        st.board.name = "Sprint.eraj".into();
        st.upsert_task(ProjectTask {
            id: "t1".into(),
            title: "Do".into(),
            board: "backlog".into(),
            drive_object_id: None,
            assignee: None,
            due_date: None,
            labels: vec![],
            checklist: vec![],
            priority: None,
            sort_key: 0,
            extra: Default::default(),
        });
        st.save_to(&path).unwrap();
        let mut st2 = SoloProjectsState::default();
        st2.open_path(&path).unwrap();
        assert_eq!(st2.board.tasks.len(), 1);
        assert_eq!(st2.board.tasks[0].title, "Do");
    }
}
