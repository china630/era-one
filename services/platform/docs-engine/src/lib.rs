//! ERA Documents engine — HTTP/WS, Drive bind, persist (ADR-0026).
//!
//! Pure model/convert/OpLog live in `era-docs-core` (S2/B0).
//! Span algorithms are also published via `era-office-richtext` (shared with Presentations).

pub use era_docs_core::{canonical, convert, model, spans, sync, wire, EradDocument};
pub use era_office_richtext as richtext;

pub mod auth;
pub mod drive_bind;
pub mod intent;
pub mod persist;
pub mod server;
