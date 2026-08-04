//! ERA Projects pure core — `.eraj` board + tasks on disk (C1).
//! Compatible with Platform Drive MIME `application/vnd.era.eraj` where possible.
//! No Postgres.

use anyhow::{bail, Result};
use serde::{Deserialize, Serialize};
use serde_json::Value;

pub const FORMAT: &str = "eraj";
pub const MIME: &str = "application/vnd.era.eraj";

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct ProjectTask {
    pub id: String,
    #[serde(default)]
    pub title: String,
    #[serde(default = "default_board")]
    pub board: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub drive_object_id: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub assignee: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub due_date: Option<String>,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub labels: Vec<String>,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub checklist: Vec<Value>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub priority: Option<String>,
    #[serde(default)]
    pub sort_key: i64,
    /// Extra fields from Platform / UI (forward-compat).
    #[serde(flatten)]
    pub extra: serde_json::Map<String, Value>,
}

fn default_board() -> String {
    "backlog".into()
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct ErajBoard {
    #[serde(default = "fmt")]
    pub format: String,
    #[serde(default)]
    pub id: String,
    #[serde(default)]
    pub name: String,
    #[serde(default)]
    pub tasks: Vec<ProjectTask>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub version: Option<u64>,
}

fn fmt() -> String {
    FORMAT.into()
}

impl Default for ErajBoard {
    fn default() -> Self {
        Self::empty()
    }
}

impl ErajBoard {
    pub fn empty() -> Self {
        Self {
            format: FORMAT.into(),
            id: String::new(),
            name: "Untitled.eraj".into(),
            tasks: Vec::new(),
            version: Some(0),
        }
    }

    pub fn validate(&self) -> Result<()> {
        if self.format != FORMAT && !self.format.is_empty() {
            bail!("unexpected format {:?}, want {FORMAT}", self.format);
        }
        let mut seen = std::collections::HashSet::new();
        for t in &self.tasks {
            if t.id.is_empty() {
                bail!("task id required");
            }
            if !seen.insert(t.id.clone()) {
                bail!("duplicate task id {}", t.id);
            }
        }
        Ok(())
    }

    pub fn to_json_bytes(&self) -> Result<Vec<u8>> {
        self.validate()?;
        Ok(serde_json::to_vec_pretty(self)?)
    }

    pub fn from_json_bytes(bytes: &[u8]) -> Result<Self> {
        let mut board: Self = serde_json::from_slice(bytes)?;
        if board.format.is_empty() {
            board.format = FORMAT.into();
        }
        board.validate()?;
        Ok(board)
    }

    pub fn load_path(path: &std::path::Path) -> Result<Self> {
        let bytes = std::fs::read(path)?;
        Self::from_json_bytes(&bytes)
    }

    pub fn save_path(&self, path: &std::path::Path) -> Result<()> {
        let bytes = self.to_json_bytes()?;
        if let Some(parent) = path.parent() {
            std::fs::create_dir_all(parent)?;
        }
        std::fs::write(path, bytes)?;
        Ok(())
    }

    pub fn upsert_task(&mut self, task: ProjectTask) {
        if let Some(existing) = self.tasks.iter_mut().find(|t| t.id == task.id) {
            *existing = task;
        } else {
            self.tasks.push(task);
        }
    }

    pub fn remove_task(&mut self, id: &str) -> bool {
        let before = self.tasks.len();
        self.tasks.retain(|t| t.id != id);
        self.tasks.len() != before
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn roundtrip_empty() {
        let b = ErajBoard::empty();
        let bytes = b.to_json_bytes().unwrap();
        let back = ErajBoard::from_json_bytes(&bytes).unwrap();
        assert_eq!(back.format, FORMAT);
        assert!(back.tasks.is_empty());
    }

    #[test]
    fn roundtrip_with_tasks() {
        let mut b = ErajBoard::empty();
        b.name = "Sprint.eraj".into();
        b.upsert_task(ProjectTask {
            id: "t1".into(),
            title: "Ship".into(),
            board: "doing".into(),
            drive_object_id: None,
            assignee: Some("alex".into()),
            due_date: None,
            labels: vec!["p0".into()],
            checklist: vec![],
            priority: Some("high".into()),
            sort_key: 1,
            extra: Default::default(),
        });
        let bytes = b.to_json_bytes().unwrap();
        let back = ErajBoard::from_json_bytes(&bytes).unwrap();
        assert_eq!(back.tasks.len(), 1);
        assert_eq!(back.tasks[0].title, "Ship");
        assert_eq!(back.tasks[0].board, "doing");
    }

    #[test]
    fn reject_dup_ids() {
        let raw = r#"{"format":"eraj","name":"x","tasks":[{"id":"a","title":"1"},{"id":"a","title":"2"}]}"#;
        assert!(ErajBoard::from_json_bytes(raw.as_bytes()).is_err());
    }
}
