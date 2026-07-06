//! Process capture через sysinfo (новые PID → ProcessEvent).

use crate::builder;
use crate::config::Config;
use crate::Envelope;
use std::collections::HashSet;
use sysinfo::{Pid, System};

pub struct SysinfoCapture {
    cfg: Config,
    sys: System,
    known: HashSet<u32>,
    bootstrapped: bool,
}

impl SysinfoCapture {
    pub fn new(cfg: &Config) -> Self {
        Self {
            cfg: cfg.clone(),
            sys: System::new(),
            known: HashSet::new(),
            bootstrapped: false,
        }
    }

    pub fn poll(&mut self) -> Vec<Envelope> {
        self.sys.refresh_processes(sysinfo::ProcessesToUpdate::All, true);
        let mut out = Vec::new();

        if !self.bootstrapped {
            for (pid, _) in self.sys.processes() {
                self.known.insert(pid.as_u32());
            }
            self.bootstrapped = true;
            if !crate::capture::production::is_production_mode() {
                out.push(crate::capture::stub_event(&self.cfg));
            }
            return out;
        }

        for (pid, proc) in self.sys.processes() {
            let pid_u = pid.as_u32();
            if self.known.contains(&pid_u) {
                continue;
            }
            self.known.insert(pid_u);
            let ppid = proc
                .parent()
                .map(|p: Pid| p.as_u32() as u64)
                .unwrap_or(0);
            let user = proc
                .user_id()
                .map(|u| format!("uid:{u:?}"))
                .unwrap_or_else(|| "unknown".into());
            let cmdline = proc
                .cmd()
                .iter()
                .map(|s| s.to_string_lossy())
                .collect::<Vec<_>>()
                .join(" ");
            out.push(builder::process_envelope(
                &self.cfg,
                "create",
                pid_u as u64,
                ppid,
                &proc.exe().map(|p| p.display().to_string()).unwrap_or_default(),
                &cmdline,
                &user,
                false,
            ));
        }

        if out.is_empty() && !crate::capture::production::is_production_mode() {
            out.push(crate::capture::stub_event(&self.cfg));
        }
        out
    }
}
