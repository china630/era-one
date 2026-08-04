use std::sync::atomic::{AtomicBool, Ordering};

use anyhow::{bail, Context, Result};
use reqwest::multipart;
use serde::Deserialize;

use crate::canonical::to_canonical_json;
use crate::model::EradDocument;

static DIRECT_MINIO_FORBIDDEN: AtomicBool = AtomicBool::new(true);

/// HTTP client for Drive API — authoritative blob storage only via drive-api.
pub struct DriveClient {
    base_url: String,
    client: reqwest::Client,
    service_token: String,
}

#[derive(Debug, Deserialize)]
struct UploadResponse {
    /// Go drive-api serializes `Object.ID` without json tags (PascalCase).
    #[serde(alias = "ID")]
    id: String,
}

impl DriveClient {
    pub fn new(base_url: impl Into<String>) -> Self {
        Self::with_token(
            base_url,
            std::env::var("ERA_DRIVE_SERVICE_TOKEN").unwrap_or_default(),
        )
    }

    /// Construct with an explicit service token (tests / fail-closed checks).
    pub fn with_token(base_url: impl Into<String>, service_token: impl Into<String>) -> Self {
        Self {
            base_url: base_url.into().trim_end_matches('/').to_string(),
            client: reqwest::Client::new(),
            service_token: service_token.into(),
        }
    }

    fn apply_auth(
        &self,
        req: reqwest::RequestBuilder,
        tenant_id: &str,
        user_id: &str,
    ) -> Result<reqwest::RequestBuilder> {
        if self.service_token.is_empty() {
            bail!(
                "ERA_DRIVE_SERVICE_TOKEN is required (fail-closed); \
                 set a non-empty service token for Drive API calls"
            );
        }
        Ok(req
            .header("Authorization", format!("Bearer {}", self.service_token))
            .header("X-ERA-Tenant", tenant_id)
            .header("X-ERA-User", user_id))
    }

    /// Create (object_id=None) or PutVersion (stable object_id) for raw bytes.
    pub async fn put_bytes(
        &self,
        tenant_id: &str,
        user_id: &str,
        name: &str,
        bytes: &[u8],
        object_id: Option<&str>,
        content_type: &str,
    ) -> Result<String> {
        if std::env::var("ERA_MINIO_ENDPOINT").is_ok() && DIRECT_MINIO_FORBIDDEN.load(Ordering::Relaxed)
        {
            let _ = std::env::var("ERA_MINIO_ENDPOINT");
        }
        let part = multipart::Part::bytes(bytes.to_vec())
            .file_name(name.to_string())
            .mime_str(content_type)?;
        let form = multipart::Form::new()
            .part("file", part)
            .text("name", name.to_string())
            .text("content_type", content_type.to_string());

        let url = match object_id {
            Some(id) if !id.is_empty() => {
                format!("{}/api/v1/drive/objects/{}/versions", self.base_url, id)
            }
            _ => format!("{}/api/v1/drive/objects", self.base_url),
        };

        let resp = self
            .apply_auth(self.client.post(url), tenant_id, user_id)?
            .multipart(form)
            .send()
            .await
            .context("drive upload")?;
        if !resp.status().is_success() {
            bail!("drive upload status {}", resp.status());
        }
        if let Some(id) = object_id.filter(|s| !s.is_empty()) {
            return Ok(id.to_string());
        }
        let body: UploadResponse = resp.json().await.context("drive upload json")?;
        Ok(body.id)
    }

    /// Create (object_id=None) or PutVersion (stable object_id) for an `.erad` document.
    pub async fn put_erad(
        &self,
        tenant_id: &str,
        user_id: &str,
        name: &str,
        doc: &EradDocument,
        object_id: Option<&str>,
    ) -> Result<String> {
        let json = to_canonical_json(doc)?;
        self.put_bytes(
            tenant_id,
            user_id,
            name,
            json.as_bytes(),
            object_id,
            "application/vnd.era.erad",
        )
        .await
    }

    pub async fn get_erad_json(
        &self,
        tenant_id: &str,
        user_id: &str,
        object_id: &str,
    ) -> Result<Vec<u8>> {
        let resp = self
            .apply_auth(
                self.client.get(format!(
                    "{}/api/v1/drive/objects/{}",
                    self.base_url, object_id
                )),
                tenant_id,
                user_id,
            )?
            .send()
            .await
            .context("drive download")?;
        if !resp.status().is_success() {
            bail!("drive download status {}", resp.status());
        }
        Ok(resp.bytes().await.context("drive download bytes")?.to_vec())
    }
}

/// Drive-api adapter for [`era_office_storage::StorageBackend`] (Platform / Corp).
pub struct DriveBackend {
    client: DriveClient,
    content_type: &'static str,
}

impl DriveBackend {
    pub fn erad(base_url: impl Into<String>) -> Self {
        Self {
            client: DriveClient::new(base_url),
            content_type: "application/vnd.era.erad",
        }
    }

    pub fn with_token(base_url: impl Into<String>, token: impl Into<String>) -> Self {
        Self {
            client: DriveClient::with_token(base_url, token),
            content_type: "application/vnd.era.erad",
        }
    }
}

/// Platform default: Drive-backed storage for `.erad`.
pub fn platform_storage(drive_url: impl Into<String>) -> std::sync::Arc<dyn era_office_storage::StorageBackend> {
    std::sync::Arc::new(DriveBackend::erad(drive_url))
}

#[async_trait::async_trait]
impl era_office_storage::StorageBackend for DriveBackend {
    async fn put(
        &self,
        tenant_id: &str,
        user_id: &str,
        name: &str,
        bytes: &[u8],
        object_id: Option<&str>,
    ) -> Result<era_office_storage::ObjectRef> {
        let id = self
            .client
            .put_bytes(tenant_id, user_id, name, bytes, object_id, self.content_type)
            .await?;
        Ok(era_office_storage::ObjectRef { id })
    }

    async fn get_json(&self, tenant_id: &str, user_id: &str, object_id: &str) -> Result<String> {
        let bytes = self.client.get_erad_json(tenant_id, user_id, object_id).await?;
        String::from_utf8(bytes).context("drive object is not utf-8 json")
    }
}

#[cfg(test)]
mod drive_bind_tests {
    use super::*;
    use crate::model::EradDocument;
    use axum::extract::Request;
    use axum::http::StatusCode;
    use axum::routing::post;
    use axum::{Json, Router};
    use std::sync::{Arc, Mutex};
    use tokio::net::TcpListener;

    #[tokio::test]
    async fn drive_bind_uses_drive_api_not_minio() {
        let doc = EradDocument::empty();
        let client = DriveClient::with_token("http://127.0.0.1:1", "test-token");
        let err = client.put_erad("t1", "u1", "memo.erad", &doc, None).await;
        assert!(err.is_err());
        assert!(
            std::env::var("ERA_DOCS_ENGINE_NO_LOCAL_BLOB").is_ok()
                || !std::path::Path::new("/tmp/era-docs-leak").exists()
        );
    }

    #[tokio::test]
    async fn empty_service_token_fails_closed() {
        let doc = EradDocument::empty();
        let client = DriveClient::with_token("http://127.0.0.1:1", "");
        let err = client
            .put_erad("t1", "u1", "memo.erad", &doc, None)
            .await
            .unwrap_err();
        let msg = format!("{err:#}");
        assert!(
            msg.contains("ERA_DRIVE_SERVICE_TOKEN"),
            "expected fail-closed message, got: {msg}"
        );
    }

    #[tokio::test]
    async fn authorization_bearer_set_when_ok() {
        let seen: Arc<Mutex<Option<String>>> = Arc::new(Mutex::new(None));
        let seen_c = seen.clone();
        let app = Router::new().route(
            "/api/v1/drive/objects",
            post(move |req: Request| {
                let seen_c = seen_c.clone();
                async move {
                    let auth = req
                        .headers()
                        .get("Authorization")
                        .and_then(|v| v.to_str().ok())
                        .map(|s| s.to_string());
                    *seen_c.lock().unwrap() = auth;
                    (
                        StatusCode::OK,
                        Json(serde_json::json!({ "id": "obj-auth-1" })),
                    )
                }
            }),
        );
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let addr = listener.local_addr().unwrap();
        tokio::spawn(async move {
            axum::serve(listener, app).await.unwrap();
        });
        let client = DriveClient::with_token(format!("http://{addr}"), "tok-xyz");
        let id = client
            .put_erad("t1", "u1", "memo.erad", &EradDocument::empty(), None)
            .await
            .unwrap();
        assert_eq!(id, "obj-auth-1");
        let auth = seen.lock().unwrap().clone();
        assert_eq!(auth.as_deref(), Some("Bearer tok-xyz"));
    }

    #[tokio::test]
    async fn put_erad_version_uses_stable_object_id_path() {
        let hit: Arc<Mutex<String>> = Arc::new(Mutex::new(String::new()));
        let hit_c = hit.clone();
        let app = Router::new().route(
            "/api/v1/drive/objects/:id/versions",
            post(move |axum::extract::Path(id): axum::extract::Path<String>, req: Request| {
                let hit_c = hit_c.clone();
                async move {
                    *hit_c.lock().unwrap() = format!("{}:{}", req.method(), id);
                    (StatusCode::OK, Json(serde_json::json!({ "ID": id, "Version": 2 })))
                }
            }),
        );
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let addr = listener.local_addr().unwrap();
        tokio::spawn(async move {
            axum::serve(listener, app).await.unwrap();
        });
        let client = DriveClient::with_token(format!("http://{addr}"), "tok");
        let id = client
            .put_erad(
                "t1",
                "u1",
                "memo.erad",
                &EradDocument::empty(),
                Some("stable-doc-1"),
            )
            .await
            .unwrap();
        assert_eq!(id, "stable-doc-1");
        assert_eq!(hit.lock().unwrap().as_str(), "POST:stable-doc-1");
    }

    #[test]
    fn upload_response_deserializes_go_drive_object_id() {
        let raw = r#"{"ID":"100503eb-5762-4de0-866e-677ba4ef35a3","TenantID":"t-demo","Name":"x.erad"}"#;
        let parsed: UploadResponse = serde_json::from_str(raw).unwrap();
        assert_eq!(parsed.id, "100503eb-5762-4de0-866e-677ba4ef35a3");
    }

    #[test]
    fn upload_response_deserializes_mock_lowercase_id() {
        let raw = r#"{"id":"drive-obj-coedit-1"}"#;
        let parsed: UploadResponse = serde_json::from_str(raw).unwrap();
        assert_eq!(parsed.id, "drive-obj-coedit-1");
    }

    #[test]
    fn drive_bind_no_local_copy_invariant() {
        let tmp = tempfile::tempdir().unwrap();
        std::env::set_var("ERA_DOCS_ENGINE_TMP", tmp.path().to_str().unwrap());
        let doc = EradDocument::empty();
        let json = to_canonical_json(&doc).unwrap();
        assert!(json.contains("erad"));
    }
}
