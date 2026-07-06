//! Linux capture: auditd EXECVE → Process Envelope (GA Wave-1, F-GA-2).
//!
//! Читает `ERA_AUDIT_LOG` (по умолчанию `/var/log/audit/audit.log`), парсит новые
//! записи `type=EXECVE` / `type=SYSCALL` с `exe=`. В prod mode stub не добавляется.

use crate::builder;
use crate::config::Config;
use crate::Envelope;
use std::collections::HashSet;
use std::fs::OpenOptions;
use std::io::{BufRead, BufReader, Seek, SeekFrom};
use std::path::PathBuf;

pub struct LinuxAuditCapture {
    cfg: Config,
    path: PathBuf,
    offset: u64,
    seen_keys: HashSet<String>,
}

impl LinuxAuditCapture {
    pub fn new(cfg: &Config) -> Self {
        let path = std::env::var("ERA_AUDIT_LOG")
            .map(PathBuf::from)
            .unwrap_or_else(|_| PathBuf::from("/var/log/audit/audit.log"));
        let mut s = Self {
            cfg: cfg.clone(),
            path,
            offset: 0,
            seen_keys: HashSet::new(),
        };
        s.init_offset();
        s
    }

    fn init_offset(&mut self) {
        if let Ok(meta) = std::fs::metadata(&self.path) {
            self.offset = meta.len();
        }
    }

    pub fn poll(&mut self) -> Vec<Envelope> {
        let file = match OpenOptions::new().read(true).open(&self.path) {
            Ok(f) => f,
            Err(e) => {
                return vec![builder::capture_degraded_envelope(
                    &self.cfg,
                    "linux_audit",
                    &format!("audit log unavailable: {e}"),
                )];
            }
        };
        let mut reader = BufReader::new(file);
        if reader.seek(SeekFrom::Start(self.offset)).is_err() {
            return Vec::new();
        }
        let mut out = Vec::new();
        let mut line = String::new();
        loop {
            line.clear();
            match reader.read_line(&mut line) {
                Ok(0) => break,
                Ok(_) => {
                    self.offset += line.len() as u64;
                    if let Some(ev) = self.parse_line(&line) {
                        out.push(ev);
                    }
                }
                Err(_) => break,
            }
        }
        out
    }

    fn parse_line(&mut self, line: &str) -> Option<Envelope> {
        if !line.contains("type=EXECVE") && !line.contains("type=SYSCALL") {
            return None;
        }
        let exe = extract_audit_field(line, "exe")?;
        let pid = extract_audit_field(line, "pid")
            .and_then(|s| s.parse::<u64>().ok())
            .unwrap_or(0);
        let ppid = extract_audit_field(line, "ppid")
            .and_then(|s| s.parse::<u64>().ok())
            .unwrap_or(0);
        let uid = extract_audit_field(line, "uid").unwrap_or_else(|| "unknown".into());
        let key = format!("{pid}:{exe}");
        if !self.seen_keys.insert(key) {
            return None;
        }
        let cmd = extract_audit_field(line, "a0")
            .map(|_| reconstruct_argv(line))
            .unwrap_or_default();
        Some(builder::process_envelope(
            &self.cfg,
            "create",
            pid,
            ppid,
            &exe,
            &cmd,
            &format!("uid:{uid}"),
            false,
        ))
    }
}

fn extract_audit_field(line: &str, key: &str) -> Option<String> {
    let needle = format!("{key}=");
    let start = line.find(&needle)? + needle.len();
    let rest = &line[start..];
    if rest.starts_with('"') {
        let end = rest[1..].find('"')? + 1;
        return Some(rest[1..end].to_string());
    }
    let end = rest
        .find(|c: char| c.is_whitespace() || c == ')')
        .unwrap_or(rest.len());
    Some(rest[..end].trim_matches('"').to_string())
}

fn reconstruct_argv(line: &str) -> String {
    let mut parts = Vec::new();
    let mut i = 0;
    while let Some(pos) = line[i..].find("a") {
        let idx = i + pos;
        let tail = &line[idx..];
        if let Some(eq) = tail.find('=') {
            let name = &tail[..eq];
            if name.chars().all(|c| c.is_ascii_digit() || c == 'a') && name.starts_with('a') {
                if let Some(val) = extract_audit_field(tail, name) {
                    parts.push(val);
                    i = idx + eq + 1;
                    continue;
                }
            }
        }
        i = idx + 1;
    }
    parts.join(" ")
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::config::Config;

    #[test]
    fn parses_execve_line() {
        let dir = std::env::temp_dir().join("era_audit_test");
        let _ = std::fs::create_dir_all(&dir);
        let path = dir.join("audit.log");
        let sample = concat!(
            "type=EXECVE msg=audit(1.2:3): pid=4242 ppid=1000 uid=1000 ",
            "exe=\"/usr/bin/curl\" a0=\"curl\" a1=\"http://127.0.0.1\"\n"
        );
        std::fs::write(&path, sample).unwrap();

        let mut cfg = Config::dev_defaults();
        cfg.node_id = "n-test".into();
        std::env::set_var("ERA_AUDIT_LOG", path.to_str().unwrap());
        let mut cap = LinuxAuditCapture::new(&cfg);
        cap.offset = 0;
        let ev = cap.poll();
        assert_eq!(ev.len(), 1);
        if let Some(crate::envelope::Payload::Process(p)) = &ev[0].payload {
            assert!(p.image_path.contains("curl"));
        } else {
            panic!("expected process payload");
        }
    }
}
