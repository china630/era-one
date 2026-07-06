//! E2E: inventory plugin subprocess → NDJSON → Envelope.

use era_agent_core::config::Config;
use era_agent_core::plugin::runner::PluginRunner;
use era_agent_core::plugin::{inventory_record_to_envelope, parse_ndjson_line, PluginManifest};
use std::env;
use std::path::PathBuf;

#[test]
fn inventory_plugin_abi_e2e() {
    let bin = match env::var("CARGO_BIN_EXE_era-plugin-inventory") {
        Ok(b) => b,
        Err(_) => {
            eprintln!("skip: build era-plugin-inventory first");
            return;
        }
    };
    let mut cfg = Config::dev_defaults();
    cfg.plugin_dir = PathBuf::from(".");
    cfg.dev_allow_plugins = true;
    let mut m = PluginManifest::inventory_dev();
    m.binary = bin;
    m.schedule_secs = 0;
    let runner = PluginRunner::new(&cfg);
    let lines = runner.exec(&m).expect("plugin exec");
    assert!(!lines.is_empty());
    let rec = parse_ndjson_line(&lines[0]).expect("ndjson");
    let env = inventory_record_to_envelope(&cfg, &rec);
    assert!(!env.event_id.is_empty());
}
