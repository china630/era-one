use crate::model::{Block, EradDocument, InlineSpan};

/// Canonical JSON for golden tests (stable key order via serde).
pub fn to_canonical_json(doc: &EradDocument) -> serde_json::Result<String> {
    serde_json::to_string_pretty(doc)
}

pub fn from_canonical_json(s: &str) -> serde_json::Result<EradDocument> {
    serde_json::from_str(s)
}

/// Sort blocks and inlines for deterministic comparison.
pub fn normalize(mut doc: EradDocument) -> EradDocument {
    doc.blocks.sort_by(|a, b| a.id.cmp(&b.id));
    for block in &mut doc.blocks {
        let len = block.inlines.len();
        block.inlines.retain(|s| !s.text.is_empty() || len == 1);
    }
    doc
}

/// Blank volatile ids; keep document order (for golden structure compare).
pub fn strip_volatile_ids(mut doc: EradDocument) -> EradDocument {
    doc.id.clear();
    doc.tenant_id.clear();
    doc.drive_object_id.clear();
    for block in &mut doc.blocks {
        block.id.clear();
        let len = block.inlines.len();
        block.inlines.retain(|s| !s.text.is_empty() || len == 1);
    }
    doc
}

/// Structure equality ignoring volatile document/block ids (golden AC-O2).
pub fn structure_equiv(a: &EradDocument, b: &EradDocument) -> bool {
    let a = strip_volatile_ids(a.clone());
    let b = strip_volatile_ids(b.clone());
    if a.format != b.format || a.legacy_features_dropped != b.legacy_features_dropped {
        return false;
    }
    if a.blocks.len() != b.blocks.len() {
        return false;
    }
    a.blocks.iter().zip(b.blocks.iter()).all(|(ba, bb)| {
        ba.block_type == bb.block_type
            && ba.heading_level == bb.heading_level
            && ba.list_type == bb.list_type
            && ba.inlines == bb.inlines
    })
}

#[cfg(test)]
mod structure_tests {
    use super::*;
    use crate::model::{Block, BlockType, InlineSpan};

    #[test]
    fn structure_equiv_ignores_ids() {
        let a = EradDocument {
            id: "a".into(),
            tenant_id: String::new(),
            drive_object_id: String::new(),
            format: "erad".into(),
            blocks: vec![Block {
                id: "1".into(),
                block_type: BlockType::Paragraph,
                heading_level: 0,
                list_type: None,
                align: Default::default(),
                line_spacing: None,
                indent_mm: None,
                space_before_pt: None,
                space_after_pt: None,
                list_level: 0,
                list_marker: None,
                list_restart: false,
                style_name: None,
                image_url: None,
                bookmark_name: None,
                lang: None,
                table_cells: None,
                inlines: vec![InlineSpan::plain("Hello")],
            }],
            comments: vec![],
            page: Default::default(),
            header: Default::default(),
            footer: Default::default(),
            track_changes: false,
            revisions: vec![],
            styles: vec![],
            legacy_features_dropped: false,
        };
        let mut b = a.clone();
        b.id = "b".into();
        b.blocks[0].id = "2".into();
        assert!(structure_equiv(&a, &b));
        b.blocks[0].inlines[0].bold = true;
        assert!(!structure_equiv(&a, &b));
    }
}

pub fn block_text(block: &Block) -> String {
    block.inlines.iter().map(|s| s.text.as_str()).collect()
}

pub fn push_text(block: &mut Block, text: &str) {
    if let Some(last) = block.inlines.last_mut() {
        if !last.bold
            && !last.italic
            && !last.underline
            && !last.strike
            && last.link_url.is_none()
            && last.font_family.is_none()
            && last.font_size_pt.is_none()
            && last.color.is_none()
            && last.highlight.is_none()
        {
            last.text.push_str(text);
            return;
        }
    }
    block.inlines.push(InlineSpan::plain(text));
}
