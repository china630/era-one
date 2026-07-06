//! gRPC-клиент IngestService (ADR-0008, GA-1 mTLS).

use crate::ingest_service_client::IngestServiceClient;
use crate::{BatchAck, EventBatch, Status};
use anyhow::{Context, Result};
use std::path::Path;
use tonic::transport::{Certificate, Channel, ClientTlsConfig, Endpoint, Identity};
use ulid::Ulid;

pub struct Sender {
    client: IngestServiceClient<Channel>,
    agent_id: String,
    tenant_id: String,
}

impl Sender {
    pub async fn connect(gateway_addr: &str, agent_id: &str, tenant_id: &str) -> Result<Self> {
        let endpoint = normalize_grpc_addr(gateway_addr);
        let mut ep = Endpoint::from_shared(endpoint)?;
        if let Some(tls) = client_tls_config()? {
            ep = ep.tls_config(tls)?;
        }
        let channel = ep.connect().await.context("connect ingest-gateway")?;
        Ok(Self {
            client: IngestServiceClient::new(channel),
            agent_id: agent_id.into(),
            tenant_id: tenant_id.into(),
        })
    }

    /// Отправка батча. При RETRY/THROTTLE возвращает события для re-queue.
    pub async fn push_batch(
        &mut self,
        events: Vec<crate::Envelope>,
    ) -> Result<(BatchAck, Option<Vec<crate::Envelope>>)> {
        if events.is_empty() {
            return Ok((
                BatchAck {
                    status: Status::Accepted as i32,
                    message: "empty".into(),
                    ..Default::default()
                },
                None,
            ));
        }
        let backup = events.clone();
        let batch_id = Ulid::new().to_bytes().to_vec();
        let batch = EventBatch {
            batch_id,
            agent_id: self.agent_id.clone(),
            tenant_id: self.tenant_id.clone(),
            events,
            producer_version: crate::AGENT_VERSION.into(),
        };
        let ack = self
            .client
            .push_events(batch)
            .await
            .context("PushEvents RPC")?
            .into_inner();
        let st = Status::try_from(ack.status).unwrap_or(Status::Unspecified);
        if st == Status::Retry || st == Status::Throttle {
            Ok((ack, Some(backup)))
        } else {
            Ok((ack, None))
        }
    }
}

fn client_tls_config() -> Result<Option<ClientTlsConfig>> {
    let ca = match std::env::var("ERA_TLS_CA") {
        Ok(p) => p,
        Err(_) => return Ok(None),
    };
    let ca_pem = std::fs::read(Path::new(&ca)).context("read ERA_TLS_CA")?;
    let mut tls = ClientTlsConfig::new().ca_certificate(Certificate::from_pem(ca_pem));
    if let (Ok(cert), Ok(key)) = (
        std::env::var("ERA_TLS_CLIENT_CERT"),
        std::env::var("ERA_TLS_CLIENT_KEY"),
    ) {
        let id = Identity::from_pem(std::fs::read(cert)?, std::fs::read(key)?);
        tls = tls.identity(id);
    }
    Ok(Some(tls))
}

fn normalize_grpc_addr(addr: &str) -> String {
    if addr.starts_with("http://") || addr.starts_with("https://") {
        addr.to_string()
    } else {
        format!("http://{addr}")
    }
}
