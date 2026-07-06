//! Главный цикл агента: capture + plugins + flush (ADR-0019).

use crate::buffer::RingBuffer;
use crate::capture::CaptureBackend;
use crate::config::Config;
use crate::enforce::{self, EnforceEngine};
use crate::license_gate::{self, GateDecision};
use crate::register;
use crate::sanitize;
use crate::scheduler::PluginScheduler;
use crate::sender::Sender;
use crate::tamper;
use anyhow::{Context, Result};
use std::time::Duration;
use tokio::time;
use tracing::{info, warn};

/// Запуск orchestrator (вызывается из `era-agent` binary).
pub async fn run(cfg: Config) -> Result<()> {
    info!(
        "era-agent-core v{} tenant={} node={} gateway={}",
        crate::AGENT_VERSION,
        cfg.tenant_id,
        cfg.node_id,
        cfg.gateway_addr
    );

    let mut buf = RingBuffer::new(10_000);
    let mut capture = CaptureBackend::new(&cfg);
    let mut sender = Sender::connect(&cfg.gateway_addr, &cfg.agent_id, &cfg.tenant_id)
        .await
        .context("ingest client")?;
    let mut plugins = PluginScheduler::new(&cfg);

    let enforce_eng = load_enforcement(&cfg);

    register::spawn_heartbeat(cfg.clone());

    let shutdown = tokio::signal::ctrl_c();
    tokio::pin!(shutdown);
    let mut tick = time::interval(Duration::from_secs(cfg.poll_interval_secs));

    info!("сенсор активен; Ctrl+C для остановки");
    loop {
        tokio::select! {
            _ = &mut shutdown => {
                warn!("остановка; flush {} событий", buf.len());
                flush(&mut sender, &mut buf, cfg.batch_size).await?;
                break;
            }
            _ = tick.tick() => {
                for raw in capture.poll() {
                    let clean = sanitize::sanitize(raw, &cfg.pseudonym_key);
                    if let Some(eng) = enforce_eng.as_ref() {
                        if let Some(det) = enforce::check_process_envelope(&cfg, eng, &clean) {
                            buf.push(sanitize::sanitize(det, &cfg.pseudonym_key));
                        }
                    }
                    buf.push(clean);
                }
                if let Some(alert) = tamper::poll_tamper(&cfg) {
                    buf.push(sanitize::sanitize(alert, &cfg.pseudonym_key));
                }
                let plugin_events = plugins.tick_cron(&cfg);
                plugins.push_to_buffer(&mut buf, plugin_events);
                if let Err(e) = flush(&mut sender, &mut buf, cfg.batch_size).await {
                    warn!("отправка отложена (backpressure): {e:#}");
                }
            }
        }
    }

    info!("era-agent остановлен");
    Ok(())
}

fn load_enforcement(cfg: &Config) -> Option<EnforceEngine> {
    if std::env::var("ERA_ENFORCEMENT").unwrap_or_default() != "1" {
        return None;
    }
    if license_gate::check_module(cfg, "manage") != GateDecision::Allow {
        warn!("enforcement disabled: manage module not licensed");
        return None;
    }
    match enforce::fetch_policy(&cfg.control_plane_url) {
        Ok(p) => {
            info!("enforcement policy v{} mode={:?}", p.version, p.mode);
            Some(EnforceEngine::new(p))
        }
        Err(e) => {
            warn!("enforcement policy load failed: {e:#}");
            None
        }
    }
}

async fn flush(sender: &mut Sender, buf: &mut RingBuffer, batch_size: usize) -> Result<()> {
    let batch = buf.drain(batch_size);
    if batch.is_empty() {
        return Ok(());
    }
    let backup = batch.clone();
    match sender.push_batch(batch).await {
        Ok((ack, retry)) => {
            info!(
                "PushEvents status={} message={}",
                ack.status,
                ack.message
            );
            if let Some(events) = retry {
                warn!("backpressure: re-queue {} events", events.len());
                buf.requeue_front(events);
            }
            Ok(())
        }
        Err(e) => {
            warn!("PushEvents failed, re-queue {} events: {e:#}", backup.len());
            buf.requeue_front(backup);
            Err(e)
        }
    }
}
