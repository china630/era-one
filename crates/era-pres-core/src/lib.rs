//! ERA Presentations pure core — `.erap` model + pptx/odp convert (B1; ADR-0026).
//!
//! No axum / Drive / Postgres. Server surface: `era-presentations-engine`.

pub mod convert;
pub mod convert_odp;
pub mod model;

pub use model::ErapDeck;

#[cfg(test)]
mod tests {
    use super::*;
    use crate::convert::{export_pptx, import_pptx};
    use crate::model::{ErapDeck, ErapSlide};
    use era_office_richtext::{FrameKey, FrameOp};

    #[test]
    fn empty_deck_serde_roundtrip() {
        let deck = ErapDeck::empty();
        let json = serde_json::to_vec(&deck).expect("ser");
        let back: ErapDeck = serde_json::from_slice(&json).expect("de");
        assert_eq!(deck.slides.len(), back.slides.len());
        assert!(!back.slides.is_empty());
    }

    #[test]
    fn pptx_export_import_smoke() {
        let mut deck = ErapDeck::empty();
        deck.slides[0].set_title_plain("Hello");
        deck.slides.push(ErapSlide::new_blank());
        let bytes = export_pptx(&deck).expect("export");
        assert!(bytes.len() > 100);
        let back = import_pptx(&bytes).expect("import");
        assert!(!back.slides.is_empty());
    }

    #[test]
    fn frame_op_apply() {
        let mut slide = ErapSlide::new_blank();
        let block_id = slide.title_frame.blocks[0].id.clone();
        slide.apply_op(
            FrameKey::Title,
            &FrameOp::InsertText {
                block_id,
                offset: 0,
                text: "X".into(),
                marks: None,
            },
        );
        assert!(slide.title().starts_with('X'));
    }
}
