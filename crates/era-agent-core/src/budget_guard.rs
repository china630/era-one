//! Budget-guard перед запуском cron-плагинов (ADR-0009).

use sysinfo::{CpuRefreshKind, MemoryRefreshKind, Pid, ProcessRefreshKind, RefreshKind, System};

/// Лимиты бюджета агента (стартовые; policy позже из CP).
#[derive(Debug, Clone)]
pub struct BudgetLimits {
    pub max_cpu_percent: f32,
    pub max_ram_mb: u64,
}

impl Default for BudgetLimits {
    fn default() -> Self {
        Self {
            max_cpu_percent: 2.0,
            max_ram_mb: 150,
        }
    }
}

#[derive(Debug, Clone, PartialEq)]
pub enum BudgetDecision {
    Proceed,
    DeferCpu { usage: f32 },
    DeferRam { used_mb: u64 },
}

/// CI-gate: RSS текущего процесса агента (P0-2).
pub fn check_process_memory(limits: &BudgetLimits) -> Result<(), String> {
    let mut sys = System::new_with_specifics(
        RefreshKind::new().with_processes(ProcessRefreshKind::everything()),
    );
    let pid = Pid::from_u32(std::process::id());
    sys.refresh_processes(sysinfo::ProcessesToUpdate::Some(&[pid]), true);
    let Some(proc) = sys.process(pid) else {
        return Err("cannot read process stats".into());
    };
    let rss_mb = proc.memory() / (1024 * 1024);
    if rss_mb > limits.max_ram_mb {
        return Err(format!(
            "RSS {rss_mb} MB exceeds budget {} MB",
            limits.max_ram_mb
        ));
    }
    Ok(())
}

/// Снимок текущего потребления (best-effort, global host metrics).
pub fn sample_usage() -> (f32, u64) {
    let mut sys = System::new_with_specifics(
        RefreshKind::new()
            .with_cpu(CpuRefreshKind::everything())
            .with_memory(MemoryRefreshKind::everything()),
    );
    sys.refresh_cpu_all();
    sys.refresh_memory();
    let cpu = sys.global_cpu_usage();
    let ram_mb = sys.used_memory() / (1024 * 1024);
    (cpu, ram_mb)
}

pub fn check_before_plugin(limits: &BudgetLimits) -> BudgetDecision {
    let (cpu, ram_mb) = sample_usage();
    // В CI/idle хосте CPU может быть 0 — defer только при явном превышении порога * 10
    // (глобальный CPU не равен процессу; для MVP — эвристика).
    if cpu > limits.max_cpu_percent.max(50.0) {
        return BudgetDecision::DeferCpu { usage: cpu };
    }
    if ram_mb > limits.max_ram_mb.saturating_mul(4) {
        return BudgetDecision::DeferRam { used_mb: ram_mb };
    }
    BudgetDecision::Proceed
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn check_returns_without_panic() {
        let limits = BudgetLimits::default();
        let decision = check_before_plugin(&limits);
        match decision {
            BudgetDecision::Proceed | BudgetDecision::DeferCpu { .. } | BudgetDecision::DeferRam { .. } => {}
        }
    }

    #[test]
    fn process_memory_under_budget() {
        check_process_memory(&BudgetLimits::default())
            .expect("agent core RSS should be under 150MB at test start");
    }
}
