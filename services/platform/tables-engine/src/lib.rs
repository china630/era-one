//! ERA Tables engine — HTTP/WS, Drive bind (ADR-0026).
//!
//! Pure model/calc/convert/OpLog live in `era-tables-core` (S2/B0).

pub use era_tables_core::{calc, convert, convert_ods, model, sync, EratSheet};

pub mod auth;
pub mod drive_bind;
pub mod server;
