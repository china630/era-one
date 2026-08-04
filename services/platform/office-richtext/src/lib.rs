//! Shared rich-text model and ops for ERA Office (Docs frames / Presentation text frames).

pub mod frame;
pub mod model;
pub mod ops;
pub mod spans;

pub use frame::{FrameKey, TextFrame};
pub use model::{Block, BlockType, InlineSpan, ListMarker, ListType, TextAlign};
pub use ops::{apply_frame_op, transform_frame_op, FrameOp};
