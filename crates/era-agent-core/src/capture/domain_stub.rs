//! Multi-domain synthetic capture: process + network + auth + file (F2-2).

use crate::builder;
use crate::config::Config;
use crate::Envelope;

pub struct DomainCapture {
    cfg: Config,
    tick: u64,
}

impl DomainCapture {
    pub fn new(cfg: Config) -> Self {
        Self { cfg, tick: 0 }
    }

    pub fn poll(&mut self) -> Vec<Envelope> {
        self.tick += 1;
        let mut out = vec![builder::process_envelope(
            &self.cfg,
            "create",
            4242,
            1000,
            "C:/Windows/System32/powershell.exe",
            "powershell -enc ABC123",
            "alice",
            false,
        )];
        out.push(builder::network_envelope(
            &self.cfg,
            "192.168.10.50",
            445,
        ));
        out.push(builder::auth_envelope(&self.cfg, "alice", false));
        out.push(builder::file_envelope(
            &self.cfg,
            "C:/Users/Public/Temp/malware.bin",
        ));
        out
    }
}
