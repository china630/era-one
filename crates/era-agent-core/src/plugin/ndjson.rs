//! NDJSON plugin records → Envelope (golden-stable mapping).

use crate::builder;
use crate::config::Config;
use crate::Envelope;
use anyhow::{Context, Result};
use prost_types::{value::Kind, Value};
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct SoftwareRecord {
    pub name: String,
    pub version: String,
    #[serde(default)]
    pub vendor: String,
    #[serde(default)]
    pub source: String,
}

/// Одна строка NDJSON от плагина.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct PluginRecord {
    pub domain: String,
    pub kind: String,
    #[serde(default)]
    pub hostname: String,
    #[serde(default)]
    pub platform: String,
    #[serde(default)]
    pub os_version: String,
    #[serde(default)]
    pub cpu_count: u32,
    #[serde(default)]
    pub total_memory_mb: u64,
    #[serde(default)]
    pub detail_json: String,
    #[serde(default)]
    pub fqdn: String,
    #[serde(default)]
    pub os_name: String,
    #[serde(default)]
    pub kernel: String,
    #[serde(default)]
    pub cpu_model: String,
    #[serde(default)]
    pub disk_total_gb: u64,
    #[serde(default)]
    pub serial_number: String,
    #[serde(default)]
    pub board_serial: String,
    #[serde(default)]
    pub manufacturer: String,
    #[serde(default)]
    pub model: String,
    #[serde(default)]
    pub mac_addrs: Vec<String>,
    #[serde(default)]
    pub ip_addrs: Vec<String>,
    #[serde(default)]
    pub software: Vec<SoftwareRecord>,
}

pub fn parse_ndjson_line(line: &str) -> Result<PluginRecord> {
    let line = line.trim();
    if line.is_empty() {
        anyhow::bail!("empty ndjson line");
    }
    serde_json::from_str(line).context("plugin ndjson parse")
}

/// Маппинг inventory-записи в Envelope (domain=inventory).
pub fn inventory_record_to_envelope(cfg: &Config, rec: &PluginRecord) -> Envelope {
    let mut fields = BTreeMap::new();
    insert_str(&mut fields, "hostname", &rec.hostname);
    insert_str(&mut fields, "platform", &rec.platform);
    insert_str(&mut fields, "os_version", &rec.os_version);
    insert_num(&mut fields, "cpu_count", rec.cpu_count as f64);
    insert_num(&mut fields, "total_memory_mb", rec.total_memory_mb as f64);
    if !rec.detail_json.is_empty() {
        insert_str(&mut fields, "detail", &rec.detail_json);
    }
    if !rec.fqdn.is_empty() {
        insert_str(&mut fields, "fqdn", &rec.fqdn);
    }
    if !rec.os_name.is_empty() {
        insert_str(&mut fields, "os_name", &rec.os_name);
    }
    if !rec.kernel.is_empty() {
        insert_str(&mut fields, "kernel", &rec.kernel);
    }
    if !rec.cpu_model.is_empty() {
        insert_str(&mut fields, "cpu_model", &rec.cpu_model);
    }
    if rec.disk_total_gb > 0 {
        insert_num(&mut fields, "disk_total_gb", rec.disk_total_gb as f64);
    }
    if !rec.serial_number.is_empty() {
        insert_str(&mut fields, "serial_number", &rec.serial_number);
    }
    if !rec.board_serial.is_empty() {
        insert_str(&mut fields, "board_serial", &rec.board_serial);
    }
    if !rec.manufacturer.is_empty() {
        insert_str(&mut fields, "manufacturer", &rec.manufacturer);
    }
    if !rec.model.is_empty() {
        insert_str(&mut fields, "model", &rec.model);
    }
    if !rec.mac_addrs.is_empty() {
        insert_str(&mut fields, "mac_addrs", &serde_json::to_string(&rec.mac_addrs).unwrap_or_default());
    }
    if !rec.ip_addrs.is_empty() {
        insert_str(&mut fields, "ip_addrs", &serde_json::to_string(&rec.ip_addrs).unwrap_or_default());
    }
    if !rec.software.is_empty() {
        insert_str(&mut fields, "software", &serde_json::to_string(&rec.software).unwrap_or_default());
    }
    builder::inventory_envelope_from_fields(cfg, &fields)
}

fn insert_str(fields: &mut BTreeMap<String, Value>, key: &str, val: &str) {
    if val.is_empty() {
        return;
    }
    fields.insert(
        key.into(),
        Value {
            kind: Some(Kind::StringValue(val.to_string())),
        },
    );
}

fn insert_num(fields: &mut BTreeMap<String, Value>, key: &str, val: f64) {
    fields.insert(
        key.into(),
        Value {
            kind: Some(Kind::NumberValue(val)),
        },
    );
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::config::Config;

    #[test]
    fn parse_inventory_line() {
        let line = r#"{"domain":"inventory","kind":"host_snapshot","hostname":"h1","platform":"linux","cpu_count":4}"#;
        let rec = parse_ndjson_line(line).expect("parse");
        assert_eq!(rec.domain, "inventory");
        let env = inventory_record_to_envelope(&Config::dev_defaults(), &rec);
        assert!(!env.event_id.is_empty());
    }
}
