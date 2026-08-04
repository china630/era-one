//! ERA Documents pure core — `.erad` model, convert, sync OpLog (S2/B0).
//!
//! No axum / Drive / Postgres. Server surface: `era-docs-engine`.

pub mod canonical;
pub mod convert;
pub mod model;
pub mod spans;
pub mod sync;

pub use model::EradDocument;

pub mod wire {
    include!(concat!(env!("OUT_DIR"), "/era.v1.rs"));
}
