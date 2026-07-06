//! Синтетический capture для dev/CI.

use crate::capture::stub_event;
use crate::config::Config;
use crate::Envelope;

pub struct StubCapture {
    cfg: Config,
}

impl StubCapture {
    pub fn new() -> Self {
        Self {
            cfg: Config::dev_defaults(),
        }
    }

    pub fn poll(&mut self) -> Vec<Envelope> {
        vec![stub_event(&self.cfg)]
    }
}
