use anyhow::{Context, Result};
use deadpool_postgres::{Config, Pool};
use serde_json::Value;
use tokio_postgres::NoTls;

use crate::sync::{DocOp, OpLog};

/// Postgres-backed doc session store (`era_platform.doc_sessions`).
pub struct PgPersist {
    pool: Pool,
}

impl PgPersist {
    /// Opens pool when `ERA_OFFICE_DATABASE_URL` (or `ERA_COMMS_DATABASE_URL`) is set.
    pub async fn from_env() -> Result<Option<Self>> {
        let dsn = std::env::var("ERA_OFFICE_DATABASE_URL")
            .or_else(|_| std::env::var("ERA_COMMS_DATABASE_URL"))
            .unwrap_or_default();
        if dsn.trim().is_empty() {
            return Ok(None);
        }
        let mut cfg = Config::new();
        cfg.url = Some(dsn);
        let pool = cfg.create_pool(None, NoTls).context("doc_sessions pg pool")?;
        Ok(Some(Self { pool }))
    }

    pub async fn ensure_session(&self, tenant_id: &str, object_id: &str) -> Result<()> {
        let client = self.pool.get().await?;
        client
            .execute(
                "INSERT INTO era_platform.doc_sessions (tenant_id, drive_object_id, version, ops_json)
                 VALUES ($1, $2, 0, '[]'::jsonb)
                 ON CONFLICT (tenant_id, drive_object_id) DO NOTHING",
                &[&tenant_id, &object_id],
            )
            .await?;
        Ok(())
    }

    pub async fn load_log(&self, tenant_id: &str, object_id: &str) -> Result<OpLog> {
        let client = self.pool.get().await?;
        let row = client
            .query_opt(
                "SELECT version, ops_json FROM era_platform.doc_sessions
                 WHERE tenant_id = $1 AND drive_object_id = $2",
                &[&tenant_id, &object_id],
            )
            .await?;
        if let Some(row) = row {
            let version: i64 = row.get(0);
            let json: Value = row.get(1);
            let ops: Vec<DocOp> = match json {
                Value::Array(arr) => arr
                    .into_iter()
                    .filter_map(|v| serde_json::from_value(v).ok())
                    .collect(),
                _ => Vec::new(),
            };
            Ok(OpLog {
                version: version.max(0) as u64,
                ops,
            })
        } else {
            Ok(OpLog::default())
        }
    }

    pub async fn save_log(&self, tenant_id: &str, object_id: &str, log: &OpLog) -> Result<()> {
        let client = self.pool.get().await?;
        let json = serde_json::to_value(&log.ops)?;
        let updated = client
            .execute(
                "UPDATE era_platform.doc_sessions
                 SET version = $3, ops_json = $4, updated_at = now()
                 WHERE tenant_id = $1 AND drive_object_id = $2",
                &[&tenant_id, &object_id, &(log.version as i64), &json],
            )
            .await?;
        if updated == 0 {
            client
                .execute(
                    "INSERT INTO era_platform.doc_sessions (tenant_id, drive_object_id, version, ops_json)
                     VALUES ($1, $2, $3, $4)",
                    &[&tenant_id, &object_id, &(log.version as i64), &json],
                )
                .await?;
        }
        Ok(())
    }

    pub async fn tenant_for_object(&self, object_id: &str) -> Result<Option<String>> {
        let client = self.pool.get().await?;
        let row = client
            .query_opt(
                "SELECT tenant_id FROM era_platform.doc_sessions WHERE drive_object_id = $1 LIMIT 1",
                &[&object_id],
            )
            .await?;
        Ok(row.map(|r| r.get(0)))
    }
}

#[cfg(test)]
mod persist_tests {
    use super::*;
    use crate::model::EradDocument;
    use crate::sync::{apply_op, DocOp};

    #[tokio::test]
    async fn pg_session_replay_roundtrip() {
        let Some(pg) = PgPersist::from_env().await.expect("from_env") else {
            return;
        };
        let tenant_id = format!("t-persist-{}", uuid::Uuid::new_v4());
        let object_id = format!("obj-{}", uuid::Uuid::new_v4());
        pg.ensure_session(&tenant_id, &object_id)
            .await
            .expect("ensure");

        let mut doc = EradDocument::empty();
        let block_id = doc.blocks[0].id.clone();
        let mut log = OpLog::default();
        let op = DocOp::InsertText {
            block_id,
            offset: 0,
            text: "persisted".into(),
                    marks: None,
        };
        apply_op(&mut doc, &op);
        log.append(op);

        pg.save_log(&tenant_id, &object_id, &log)
            .await
            .expect("save");

        let loaded = pg.load_log(&tenant_id, &object_id).await.expect("load");
        assert_eq!(loaded.version, 1);
        assert_eq!(loaded.ops.len(), 1);

        let replayed = loaded.replay(EradDocument::empty());
        assert!(replayed.plain_text().contains("persisted"));
    }
}
