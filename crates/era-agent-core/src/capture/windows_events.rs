//! Windows capture: Sysmon/EVTX export → Process Envelope (GA Wave-1, F-GA-1).
//!
//! Источники (по приоритету):
//! 1. `ERA_SYSMON_JSONL` — NDJSON (поля image, command_line, pid, ppid, user)
//! 2. `ERA_SYSMON_EVtx` — путь к `.evtx` (Windows, через wevtutil export в sidecar)
//!
//! На non-Windows — no-op (CI собирает крейт без Win API).

use crate::builder;
use crate::config::Config;
use crate::Envelope;
use std::collections::HashSet;
use std::fs::File;
use std::io::{BufRead, BufReader, Seek, SeekFrom};
use std::path::PathBuf;

pub struct WindowsEventsCapture {
    cfg: Config,
    jsonl: Option<PathBuf>,
    offset: u64,
    seen: HashSet<String>,
}

impl WindowsEventsCapture {
    pub fn new(cfg: &Config) -> Self {
        let jsonl = std::env::var("ERA_SYSMON_JSONL").ok().map(PathBuf::from);
        Self {
            cfg: cfg.clone(),
            jsonl,
            offset: 0,
            seen: HashSet::new(),
        }
    }

    pub fn poll(&mut self) -> Vec<Envelope> {
        if let Some(path) = self.jsonl.clone() {
            return self.poll_jsonl(&path);
        }
        if let Ok(path) = std::env::var("ERA_SYSMON_EVtx") {
            return super::evtx_parser::poll_evtx_file(
                &self.cfg,
                std::path::Path::new(&path),
                &mut self.offset,
                &mut self.seen,
            );
        }
        #[cfg(target_os = "windows")]
        {
            return self.poll_wevtutil_export();
        }
        #[cfg(not(target_os = "windows"))]
        {
            Vec::new()
        }
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

    #[cfg(target_os = "windows")]
    fn poll_wevtutil_export(&mut self) -> Vec<Envelope> {
        use std::process::Command;
        let tmp = std::env::temp_dir().join("era_sysmon_export.jsonl");
        let status = Command::new("wevtutil")
            .args([
                "qe",
                "Microsoft-Windows-Sysmon/Operational",
                "/f:RenderedXml",
                "/c:32",
                "/rd:true",
            ])
            .output();
        let Ok(out) = status else {
            return Vec::new();
        };
        if !out.status.success() {
            return Vec::new();
        }
        let xml = String::from_utf8_lossy(&out.stdout);
        let mut events = Vec::new();
        for chunk in xml.split("<Event ") {
            if !chunk.contains("EventID") {
                continue;
            }
            let image = extract_xml_data(chunk, "Image").unwrap_or_default();
            if image.is_empty() {
                continue;
            }
            let pid = extract_xml_data(chunk, "ProcessId")
                .and_then(|s| s.parse().ok())
                .unwrap_or(0);
            let ppid = extract_xml_data(chunk, "ParentProcessId")
                .and_then(|s| s.parse().ok())
                .unwrap_or(0);
            let cmd = extract_xml_data(chunk, "CommandLine").unwrap_or_default();
            let user = extract_xml_data(chunk, "User").unwrap_or_else(|| "unknown".into());
            let key = format!("{pid}:{image}");
            if !self.seen.insert(key) {
                continue;
            }
            events.push(builder::process_envelope(
                &self.cfg,
                "create",
                pid,
                ppid,
                &image,
                &cmd,
                &user,
                false,
            ));
        }
        let _ = std::fs::write(&tmp, ""); // touch for observability
        events
    }

    #[cfg(target_os = "windows")]
    fn poll_evtx_export(&mut self, path: &PathBuf) -> Vec<Envelope> {
        super::evtx_parser::poll_evtx_file(&self.cfg, path, &mut self.offset, &mut self.seen)
    }
}

fn json_to_envelope(
    cfg: &Config,
    v: &serde_json::Value,
    seen: &mut HashSet<String>,
) -> Option<Envelope> {
    let image = v.get("image")?.as_str()?.to_string();
    let pid = v.get("pid").and_then(|x| x.as_u64()).unwrap_or(0);
    let ppid = v.get("ppid").and_then(|x| x.as_u64()).unwrap_or(0);
    let cmd = v
        .get("command_line")
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
        cfg,
        "create",
        pid,
        ppid,
        &image,
        &cmd,
        &user,
        false,
    ))
}

#[cfg(target_os = "windows")]
fn extract_xml_data(chunk: &str, name: &str) -> Option<String> {
    let open = format!("<Data Name=\"{name}\">");
    let start = chunk.find(&open)? + open.len();
    let end = chunk[start..].find("</Data>")? + start;
    Some(chunk[start..end].trim().to_string())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::config::Config;

    #[test]
    fn jsonl_sysmon_event() {
        let dir = std::env::temp_dir().join("era_sysmon_test");
        let _ = std::fs::create_dir_all(&dir);
        let path = dir.join("sysmon.jsonl");
        std::fs::write(
            &path,
            r#"{"image":"C:\\Windows\\System32\\cmd.exe","command_line":"cmd.exe /c whoami","pid":1234,"ppid":567,"user":"DOMAIN\\alice"}"#,
        )
        .unwrap();
        std::env::set_var("ERA_SYSMON_JSONL", path.to_str().unwrap());
        let cfg = Config::dev_defaults();
        let mut cap = WindowsEventsCapture::new(&cfg);
        cap.offset = 0;
        let ev = cap.poll();
        assert_eq!(ev.len(), 1);
    }
}
