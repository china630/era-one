//! ERA Tables pure core — `.erat` model, calc, convert, sync OpLog (S2/B0).
//!
//! No axum / Drive. Server surface: `era-tables-engine`.

pub mod calc;
pub mod convert;
pub mod convert_ods;
pub mod model;
pub mod sync;

pub use model::EratSheet;
