//! BYO-EDR Hub — нормализация стороннего EDR/SIEM feed в Envelope (ADR-0017 §3).

mod byo_edr;

pub use byo_edr::{
    parse_json_line, parse_syslog_line, ByoEdrConfig, ByoEdrError, GENERIC_SOURCE_TYPE,
};

pub mod modbus;

pub use modbus::{parse_modbus_frame, ModbusFrame};
