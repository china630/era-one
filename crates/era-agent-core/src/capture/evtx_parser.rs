//! EVTX binary parser (cross-platform, GA S8-4).
//! Минимальный парсер XML-записей внутри EVTX для Sysmon process create.

use crate::builder;
use crate::config::Config;
use crate::Envelope;
use std::collections::HashSet;
use std::fs::File;
use std::io::{Read, Seek, SeekFrom};
use std::path::Path;

/// Читает новые записи из `.evtx` с offset (упрощённый XML scan внутри файла).
pub fn poll_evtx_file(
    cfg: &Config,
    path: &Path,
    offset: &mut u64,
    seen: &mut HashSet<String>,
) -> Vec<Envelope> {
    let mut file = match File::open(path) {
        Ok(f) => f,
        Err(_) => return Vec::new(),
    };
    let len = file.metadata().map(|m| m.len()).unwrap_or(0);
    if *offset >= len {
        return Vec::new();
    }
    let _ = file.seek(SeekFrom::Start(*offset));
    let mut buf = Vec::new();
    if file.read_to_end(&mut buf).is_err() {
        return Vec::new();
    }
    *offset = len;
    parse_evtx_xml_chunks(cfg, &buf, seen)
}

pub fn parse_evtx_xml_chunks(
    cfg: &Config,
    data: &[u8],
    seen: &mut HashSet<String>,
) -> Vec<Envelope> {
    let xml = String::from_utf8_lossy(data);
    let mut out = Vec::new();
    for chunk in xml.split("<Event ") {
        if !chunk.contains("EventID") {
            continue;
        }
        let image = extract_xml_data(chunk, "Image")
            .or_else(|| extract_xml_data(chunk, "ProcessName"))
            .unwrap_or_default();
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
        if !seen.insert(key) {
            continue;
        }
        out.push(builder::process_envelope(
            cfg, "create", pid, ppid, &image, &cmd, &user, false,
        ));
    }
    out
}

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
    fn evtx_xml_chunk_golden() {
        let sample = include_str!("../../testdata/sysmon_evtx_fragment.xml");
        let cfg = Config::dev_defaults();
        let mut seen = HashSet::new();
        let ev = parse_evtx_xml_chunks(&cfg, sample.as_bytes(), &mut seen);
        assert_eq!(ev.len(), 1);
    }
}
