//! ERA Presentations — `.erap` deck model + minimal pptx / odp.
//! Pure core lives in `era-pres-core`; this crate is the axum/Drive server.

pub mod auth;
pub mod drive_bind;
pub mod server;

pub mod convert {
    pub use era_pres_core::convert::*;
}

pub mod convert_odp {
    pub use era_pres_core::convert_odp::*;
}

pub mod model {
    pub use era_pres_core::model::*;
}

pub use era_pres_core::ErapDeck;
