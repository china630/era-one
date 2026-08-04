//! BYO-EDR Hub — нормализация стороннего EDR/SIEM feed в Envelope (ADR-0017 §3).

mod byo_edr;
mod dns;

pub use byo_edr::{
    parse_json_line, parse_syslog_line, ByoEdrConfig, ByoEdrError, GENERIC_SOURCE_TYPE,
};

pub use dns::{
    emit_dns_event, stub_sample_observation, DnsObservation, DnsTraceConfig, DNS_EMITTER_MODE,
};

pub mod modbus;

pub use modbus::{parse_modbus_frame, ModbusFrame};
