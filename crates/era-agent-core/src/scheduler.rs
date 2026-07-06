//! Планировщик cron-плагинов (ADR-0019 §3).

use crate::budget_guard::{self, BudgetLimits};
use crate::config::Config;
use crate::license_gate::{self, GateDecision};
use crate::plugin::{inventory_record_to_envelope, parse_ndjson_line, PluginManifest, PluginMode};
use crate::plugin::runner::PluginRunner;
use crate::sanitize;
use crate::buffer::RingBuffer;
use std::collections::HashMap;
use std::time::{Duration, Instant};
use tracing::{debug, warn};

pub struct PluginScheduler {
    manifests: Vec<PluginManifest>,
    last_run: HashMap<String, Instant>,
    runner: PluginRunner,
    budget: BudgetLimits,
}

impl PluginScheduler {
    pub fn new(cfg: &Config) -> Self {
        let manifests = cfg.load_plugin_manifests();
        Self {
            manifests,
            last_run: HashMap::new(),
            runner: PluginRunner::new(cfg),
            budget: BudgetLimits::default(),
        }
    }

    /// Запускает due cron-плагины; возвращает envelopes для буфера.
    pub fn tick_cron(&mut self, cfg: &Config) -> Vec<crate::Envelope> {
        let mut out = Vec::new();
        let now = Instant::now();
        for manifest in self.manifests.clone() {
            if manifest.mode != PluginMode::Cron {
                continue;
            }
            let due = self
                .last_run
                .get(&manifest.name)
                .map(|t| now.duration_since(*t) >= Duration::from_secs(manifest.schedule_secs))
                .unwrap_or(true);
            if !due {
                continue;
            }
            match license_gate::check_module(cfg, &manifest.license_module) {
                GateDecision::Allow => {}
                other => {
                    debug!(plugin = %manifest.name, ?other, "license-gate skip");
                    continue;
                }
            }
            match budget_guard::check_before_plugin(&self.budget) {
                budget_guard::BudgetDecision::Proceed => {}
                defer => {
                    warn!(plugin = %manifest.name, ?defer, "budget-guard defer");
                    continue;
                }
            }
            match self.runner.exec(&manifest) {
                Ok(lines) => {
                    self.last_run.insert(manifest.name.clone(), now);
                    for line in lines {
                        if let Ok(rec) = parse_ndjson_line(&line) {
                            let raw = inventory_record_to_envelope(cfg, &rec);
                            out.push(sanitize::sanitize(raw, &cfg.pseudonym_key));
                        }
                    }
                }
                Err(e) => warn!(plugin = %manifest.name, "exec failed: {e:#}"),
            }
        }
        out
    }

    pub fn push_to_buffer(&self, buf: &mut RingBuffer, events: Vec<crate::Envelope>) {
        for e in events {
            buf.push(e);
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::config::Config;

    #[test]
    fn scheduler_empty_manifests() {
        let cfg = Config::dev_defaults();
        let mut sched = PluginScheduler::new(&cfg);
        assert!(sched.tick_cron(&cfg).is_empty());
    }
}
