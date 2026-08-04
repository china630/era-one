//! Office document blob storage abstraction (S2/B0).
//!
//! Platform engines use a Drive-backed adapter; Solo (S3) uses [`LocalFsBackend`].

use std::path::{Path, PathBuf};

use anyhow::{bail, Context, Result};
use async_trait::async_trait;

/// Stable object id returned by [`StorageBackend::put`].
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ObjectRef {
    pub id: String,
}

/// Authoritative blob store for Office documents (bytes / JSON text).
#[async_trait]
pub trait StorageBackend: Send + Sync {
    /// Create (`object_id=None`) or replace version (`Some(id)`).
    async fn put(
        &self,
        tenant_id: &str,
        user_id: &str,
        name: &str,
        bytes: &[u8],
        object_id: Option<&str>,
    ) -> Result<ObjectRef>;

    /// Load object payload as UTF-8 JSON text.
    async fn get_json(&self, tenant_id: &str, user_id: &str, object_id: &str) -> Result<String>;
}

/// Local filesystem backend for Solo desktop (S3). Layout:
/// `{root}/{tenant_id}/{object_id}` plus `{object_id}.name` sidecar.
pub struct LocalFsBackend {
    root: PathBuf,
}

impl LocalFsBackend {
    pub fn new(root: impl Into<PathBuf>) -> Self {
        Self { root: root.into() }
    }

    fn tenant_dir(&self, tenant_id: &str) -> PathBuf {
        self.root.join(sanitize_segment(tenant_id))
    }

    fn object_path(&self, tenant_id: &str, object_id: &str) -> PathBuf {
        self.tenant_dir(tenant_id).join(sanitize_segment(object_id))
    }

    fn name_path(&self, tenant_id: &str, object_id: &str) -> PathBuf {
        let mut p = self.object_path(tenant_id, object_id);
        p.set_extension("name");
        p
    }
}

fn sanitize_segment(s: &str) -> String {
    s.chars()
        .map(|c| match c {
            '/' | '\\' | ':' | '\0' => '_',
            c => c,
        })
        .collect()
}

#[async_trait]
impl StorageBackend for LocalFsBackend {
    async fn put(
        &self,
        tenant_id: &str,
        user_id: &str,
        name: &str,
        bytes: &[u8],
        object_id: Option<&str>,
    ) -> Result<ObjectRef> {
        let _ = user_id;
        let id = match object_id.filter(|s| !s.is_empty()) {
            Some(id) => id.to_string(),
            None => uuid::Uuid::new_v4().to_string(),
        };
        let dir = self.tenant_dir(tenant_id);
        tokio::fs::create_dir_all(&dir)
            .await
            .with_context(|| format!("create {}", dir.display()))?;
        let path = self.object_path(tenant_id, &id);
        tokio::fs::write(&path, bytes)
            .await
            .with_context(|| format!("write {}", path.display()))?;
        let name_path = self.name_path(tenant_id, &id);
        tokio::fs::write(&name_path, name.as_bytes())
            .await
            .with_context(|| format!("write {}", name_path.display()))?;
        Ok(ObjectRef { id })
    }

    async fn get_json(&self, tenant_id: &str, user_id: &str, object_id: &str) -> Result<String> {
        let _ = user_id;
        let path = self.object_path(tenant_id, object_id);
        if !Path::new(&path).exists() {
            bail!("object not found: {object_id}");
        }
        let bytes = tokio::fs::read(&path)
            .await
            .with_context(|| format!("read {}", path.display()))?;
        String::from_utf8(bytes).context("object is not utf-8 json")
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn local_fs_roundtrip() {
        let dir = tempfile::tempdir().unwrap();
        let store = LocalFsBackend::new(dir.path());
        let put = store
            .put("t1", "u1", "memo.erad", br#"{"v":1}"#, None)
            .await
            .unwrap();
        assert!(!put.id.is_empty());
        let got = store.get_json("t1", "u1", &put.id).await.unwrap();
        assert_eq!(got, r#"{"v":1}"#);
        let put2 = store
            .put("t1", "u1", "memo.erad", br#"{"v":2}"#, Some(&put.id))
            .await
            .unwrap();
        assert_eq!(put2.id, put.id);
        let got2 = store.get_json("t1", "u1", &put.id).await.unwrap();
        assert_eq!(got2, r#"{"v":2}"#);
    }
}
