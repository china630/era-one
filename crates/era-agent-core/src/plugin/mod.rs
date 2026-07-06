//! Plugin subprocess ABI: manifest, NDJSON runner (ADR-0019 §2).

pub mod manifest;
pub mod ndjson;
pub mod runner;

pub use manifest::{PluginManifest, PluginMode};
pub use ndjson::{inventory_record_to_envelope, parse_ndjson_line, PluginRecord};
pub use runner::PluginRunner;
