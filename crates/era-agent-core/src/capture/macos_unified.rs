//! macOS production capture: unified log / Endpoint Security export (NDJSON).
//!
//! Источник: `ERA_MACOS_UNIFIED_JSONL` — NDJSON с полями process, pid, ppid, user, path.

use crate::builder;
use crate::config::Config;
use crate::Envelope;
use std::collections::HashSet;
use std::fs::File;
use std::io::{BufRead, BufReader, Seek, SeekFrom};
use std::path::PathBuf;

pub struct MacosUnifiedCapture {
    cfg: Config,
    jsonl: Option<PathBuf>,
    offset: u64,
    seen: HashSet<String>,
}

impl MacosUnifiedCapture {
    pub fn new(cfg: &Config) -> Self {
        let jsonl = std::env::var("ERA_MACOS_UNIFIED_JSONL").ok().map(PathBuf::from);
        Self {
            cfg: cfg.clone(),
            jsonl,
            offset: 0,
            seen: HashSet::new(),
        }
    }

    pub fn poll(&mut self) -> Vec<Envelope> {
        let Some(path) = self.jsonl.clone() else {
            return Vec::new();
        };
        self.poll_jsonl(&path)
    }

    fn poll_jsonl(&mut self, path: &PathBuf) -> Vec<Envelope> {
        let file = match File::open(path) {
            Ok(f) => f,
            Err(_) => return Vec::new(),
        };
        let mut reader = BufReader::new(file);
        let _ = reader.seek(SeekFrom::Start(self.offset));
        let mut out = Vec::new();
        let mut line = String::new();
        loop {
            line.clear();
            match reader.read_line(&mut line) {
                Ok(0) => break,
                Ok(n) => {
                    self.offset += n as u64;
                    if let Ok(v) = serde_json::from_str::<serde_json::Value>(&line) {
                        if let Some(ev) = json_to_envelope(&self.cfg, &v, &mut self.seen) {
                            out.push(ev);
                        }
                    }
                }
                Err(_) => break,
            }
        }
        out
    }
}

fn json_to_envelope(
    cfg: &Config,
    v: &serde_json::Value,
    seen: &mut HashSet<String>,
) -> Option<Envelope> {
    let image = v
        .get("process")
        .or_else(|| v.get("path"))
        .and_then(|x| x.as_str())?
        .to_string();
    let pid = v.get("pid").and_then(|x| x.as_u64()).unwrap_or(0);
    let ppid = v.get("ppid").and_then(|x| x.as_u64()).unwrap_or(0);
    let cmd = v
        .get("command")
        .or_else(|| v.get("command_line"))
        .and_then(|x| x.as_str())
        .unwrap_or("")
        .to_string();
    let user = v
        .get("user")
        .and_then(|x| x.as_str())
        .unwrap_or("unknown")
        .to_string();
    let key = format!("{pid}:{image}");
    if !seen.insert(key) {
        return None;
    }
    Some(builder::process_envelope(
        cfg, "create", pid, ppid, &image, &cmd, &user, false,
    ))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn macos_unified_jsonl() {
        let dir = std::env::temp_dir().join("era_macos_test");
        let _ = std::fs::create_dir_all(&dir);
        let path = dir.join("unified.jsonl");
        std::fs::write(
            &path,
            r#"{"process":"/usr/bin/bash","command":"bash -c id","pid":9001,"ppid":1,"user":"alice"}"#,
        )
        .unwrap();
        std::env::set_var("ERA_MACOS_UNIFIED_JSONL", path.to_str().unwrap());
        let cfg = Config::dev_defaults();
        let mut cap = MacosUnifiedCapture::new(&cfg);
        let ev = cap.poll();
        assert_eq!(ev.len(), 1);
    }
}
