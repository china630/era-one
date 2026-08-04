use serde::{Deserialize, Serialize};

use crate::model::{
    Block, BlockType, DocComment, EradDocument, InlineSpan, ListMarker, ListType, TextAlign,
};
use crate::spans::{
    apply_marks_range, block_plain, coalesce_adjacent, delete_range_preserving,
    insert_text_preserving, split_inlines_at, MarksPatch,
};

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(tag = "type", rename_all = "snake_case")]
pub enum DocOp {
    InsertText {
        block_id: String,
        offset: usize,
        text: String,
        /// Optional typing-style marks for the inserted run.
        #[serde(default, skip_serializing_if = "Option::is_none")]
        marks: Option<InlineSpan>,
    },
    DeleteRange {
        block_id: String,
        start: usize,
        end: usize,
    },
    SetBlockType {
        block_id: String,
        block_type: BlockType,
        heading_level: u32,
    },
    InsertBlock {
        after_id: String,
        block: Block,
    },
    /// Split block at offset; tail becomes `new_block_id` inserted after.
    SplitBlock {
        block_id: String,
        offset: usize,
        new_block_id: String,
    },
    /// Merge `block_id` into the previous block (with_previous must be true).
    MergeBlocks {
        block_id: String,
        #[serde(default = "default_true")]
        with_previous: bool,
    },
    /// Apply marks to a character range (selection). Option fields = patch.
    SetMarksRange {
        block_id: String,
        start: usize,
        end: usize,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        bold: Option<bool>,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        italic: Option<bool>,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        underline: Option<bool>,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        strike: Option<bool>,
        /// `Some("")` clears; `None` leaves unchanged.
        #[serde(default, skip_serializing_if = "Option::is_none")]
        link_url: Option<String>,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        font_family: Option<String>,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        font_size_pt: Option<u32>,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        color: Option<String>,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        highlight: Option<String>,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        superscript: Option<bool>,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        subscript: Option<bool>,
    },
    /// Apply marks to the whole block (compat; implemented via SetMarksRange).
    SetInlineMarks {
        block_id: String,
        bold: bool,
        italic: bool,
        #[serde(default)]
        underline: bool,
        /// `Some("")` clears link; `None` preserves existing link_url.
        #[serde(default, skip_serializing_if = "Option::is_none")]
        link_url: Option<String>,
        #[serde(default)]
        strike: bool,
        #[serde(default)]
        superscript: bool,
        #[serde(default)]
        subscript: bool,
    },
    /// O-FMT-1: paragraph / list formatting attrs.
    SetBlockFormat {
        block_id: String,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        align: Option<TextAlign>,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        indent_mm: Option<u32>,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        line_spacing: Option<String>,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        space_before_pt: Option<u32>,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        space_after_pt: Option<u32>,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        list_type: Option<ListType>,
        /// Explicit clear: `Some(None)` not representable; send list_type null via
        /// clear_list flag.
        #[serde(default, skip_serializing_if = "Option::is_none")]
        clear_list: Option<bool>,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        list_level: Option<u32>,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        list_marker: Option<ListMarker>,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        list_restart: Option<bool>,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        style_name: Option<String>,
    },
    AddComment {
        id: String,
        block_id: String,
        #[serde(default)]
        author_id: String,
        text: String,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        start: Option<usize>,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        end: Option<usize>,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        quote: Option<String>,
    },
    ResolveComment {
        id: String,
    },
}

fn default_true() -> bool {
    true
}

fn opt_string_patch(v: &Option<String>) -> Option<Option<String>> {
    match v {
        None => None,
        Some(s) if s.is_empty() => Some(None),
        Some(s) => Some(Some(s.clone())),
    }
}

fn opt_u32_patch(v: &Option<u32>) -> Option<Option<u32>> {
    match v {
        None => None,
        Some(0) => Some(None),
        Some(n) => Some(Some(*n)),
    }
}

fn shift_range(start: usize, end: usize, at: usize, delta: isize) -> (usize, usize) {
    let shift = |x: usize| -> usize {
        if (x as isize) < (at as isize) {
            x
        } else if delta >= 0 {
            x.saturating_add(delta as usize)
        } else {
            x.saturating_sub((-delta) as usize)
        }
    };
    let s = shift(start);
    let e = shift(end);
    if s > e {
        (e, e)
    } else {
        (s, e)
    }
}

impl DocOp {
    /// Transform `self` against an already-applied concurrent `other` (insert OT lite).
    pub fn transform_against(&self, other: &DocOp) -> DocOp {
        match (self, other) {
            (
                DocOp::InsertText {
                    block_id,
                    offset,
                    text,
                    marks,
                },
                DocOp::InsertText {
                    block_id: ob,
                    offset: oo,
                    text: ot,
                    ..
                },
            ) if block_id == ob => {
                let new_offset = if *offset >= *oo {
                    offset.saturating_add(ot.len())
                } else {
                    *offset
                };
                DocOp::InsertText {
                    block_id: block_id.clone(),
                    offset: new_offset,
                    text: text.clone(),
                    marks: marks.clone(),
                }
            }
            (
                DocOp::DeleteRange {
                    block_id,
                    start,
                    end,
                },
                DocOp::InsertText {
                    block_id: ob,
                    offset: oo,
                    text: ot,
                    ..
                },
            ) if block_id == ob => {
                let len = ot.len();
                let (start, end) = if *oo <= *start {
                    (start.saturating_add(len), end.saturating_add(len))
                } else if *oo < *end {
                    (*start, end.saturating_add(len))
                } else {
                    (*start, *end)
                };
                DocOp::DeleteRange {
                    block_id: block_id.clone(),
                    start,
                    end,
                }
            }
            (
                DocOp::InsertText {
                    block_id,
                    offset,
                    text,
                    marks,
                },
                DocOp::DeleteRange {
                    block_id: ob,
                    start,
                    end,
                },
            ) if block_id == ob => {
                let del_len = end.saturating_sub(*start);
                let new_offset = if *offset >= *end {
                    offset.saturating_sub(del_len)
                } else if *offset > *start {
                    *start
                } else {
                    *offset
                };
                DocOp::InsertText {
                    block_id: block_id.clone(),
                    offset: new_offset,
                    text: text.clone(),
                    marks: marks.clone(),
                }
            }
            (
                DocOp::SetMarksRange {
                    block_id,
                    start,
                    end,
                    ..
                },
                DocOp::InsertText {
                    block_id: ob,
                    offset: oo,
                    text: ot,
                    ..
                },
            ) if block_id == ob => {
                let (start, end) = shift_range(*start, *end, *oo, ot.len() as isize);
                let mut op = self.clone();
                if let DocOp::SetMarksRange {
                    start: s, end: e, ..
                } = &mut op
                {
                    *s = start;
                    *e = end;
                }
                op
            }
            (
                DocOp::SetMarksRange {
                    block_id,
                    start,
                    end,
                    ..
                },
                DocOp::DeleteRange {
                    block_id: ob,
                    start: ds,
                    end: de,
                },
            ) if block_id == ob => {
                let del_len = de.saturating_sub(*ds) as isize;
                let (shifted_s, shifted_e) = shift_range(*start, *end, *de, -del_len);
                // Also clamp if range overlapped delete
                let mut s = shifted_s;
                let mut e = shifted_e;
                if *ds < e && *de > s {
                    if *ds <= s && *de >= e {
                        s = *ds;
                        e = *ds;
                    } else if *ds <= s {
                        s = *ds;
                        e = e.saturating_sub(del_len as usize);
                    } else if *de >= e {
                        e = *ds;
                    } else {
                        e = e.saturating_sub(del_len as usize);
                    }
                }
                let mut op = self.clone();
                if let DocOp::SetMarksRange {
                    start: rs, end: re, ..
                } = &mut op
                {
                    *rs = s.min(e);
                    *re = e;
                }
                op
            }
            (
                DocOp::InsertText {
                    block_id,
                    offset,
                    text,
                    marks,
                },
                DocOp::SplitBlock {
                    block_id: ob,
                    offset: so,
                    ..
                },
            ) if block_id == ob => {
                if *offset > *so {
                    // Insert lands in the new (tail) block — leave as-is; client
                    // should retarget. Conservative: clamp to split point on old block.
                    DocOp::InsertText {
                        block_id: block_id.clone(),
                        offset: *so,
                        text: text.clone(),
                        marks: marks.clone(),
                    }
                } else {
                    self.clone()
                }
            }
            (
                DocOp::SplitBlock {
                    block_id,
                    offset,
                    new_block_id,
                },
                DocOp::InsertText {
                    block_id: ob,
                    offset: oo,
                    text: ot,
                    ..
                },
            ) if block_id == ob => {
                let new_offset = if *offset >= *oo {
                    offset.saturating_add(ot.len())
                } else {
                    *offset
                };
                DocOp::SplitBlock {
                    block_id: block_id.clone(),
                    offset: new_offset,
                    new_block_id: new_block_id.clone(),
                }
            }
            // MergeBlocks: conservative — leave unchanged (snapshot reconcile).
            (_, DocOp::MergeBlocks { .. }) | (DocOp::MergeBlocks { .. }, _) => self.clone(),
            _ => self.clone(),
        }
    }
}

#[derive(Debug, Default, Clone)]
pub struct OpLog {
    pub version: u64,
    pub ops: Vec<DocOp>,
}

impl OpLog {
    pub fn append(&mut self, op: DocOp) {
        self.version += 1;
        self.ops.push(op);
    }

    pub fn replay(&self, mut doc: EradDocument) -> EradDocument {
        for op in &self.ops {
            apply_op(&mut doc, op);
        }
        doc
    }

    pub fn ops_since(&self, since: u64) -> &[DocOp] {
        let skip = since.min(self.ops.len() as u64) as usize;
        &self.ops[skip..]
    }
}

fn copy_block_attrs(from: &Block, id: String, inlines: Vec<InlineSpan>) -> Block {
    Block {
        id,
        block_type: from.block_type,
        heading_level: from.heading_level,
        list_type: from.list_type,
        align: from.align,
        line_spacing: from.line_spacing.clone(),
        indent_mm: from.indent_mm,
        space_before_pt: from.space_before_pt,
        space_after_pt: from.space_after_pt,
        list_level: from.list_level,
        list_marker: from.list_marker,
        list_restart: false,
        style_name: from.style_name.clone(),
        image_url: from.image_url.clone(),
        bookmark_name: from.bookmark_name.clone(),
        lang: from.lang.clone(),
        table_cells: from.table_cells.clone(),
        inlines,
    }
}

pub fn apply_op(doc: &mut EradDocument, op: &DocOp) {
    match op {
        DocOp::InsertText {
            block_id,
            offset,
            text,
            marks,
        } => {
            if let Some(block) = doc.blocks.iter_mut().find(|b| b.id == *block_id) {
                insert_text_preserving(&mut block.inlines, *offset, text, marks.as_ref());
            }
        }
        DocOp::DeleteRange {
            block_id,
            start,
            end,
        } => {
            if let Some(block) = doc.blocks.iter_mut().find(|b| b.id == *block_id) {
                delete_range_preserving(&mut block.inlines, *start, *end);
            }
        }
        DocOp::SetBlockType {
            block_id,
            block_type,
            heading_level,
        } => {
            if let Some(block) = doc.blocks.iter_mut().find(|b| b.id == *block_id) {
                block.block_type = *block_type;
                block.heading_level = *heading_level;
            }
        }
        DocOp::InsertBlock { after_id, block } => {
            if let Some(pos) = doc.blocks.iter().position(|b| b.id == *after_id) {
                doc.blocks.insert(pos + 1, block.clone());
            } else {
                doc.blocks.push(block.clone());
            }
        }
        DocOp::SplitBlock {
            block_id,
            offset,
            new_block_id,
        } => {
            if doc.blocks.iter().any(|b| b.id == *new_block_id) {
                return;
            }
            if let Some(pos) = doc.blocks.iter().position(|b| b.id == *block_id) {
                let (left, right) = split_inlines_at(&doc.blocks[pos].inlines, *offset);
                let attrs_src = doc.blocks[pos].clone();
                doc.blocks[pos].inlines = left;
                let new_block = copy_block_attrs(&attrs_src, new_block_id.clone(), right);
                doc.blocks.insert(pos + 1, new_block);
            }
        }
        DocOp::MergeBlocks {
            block_id,
            with_previous,
        } => {
            if !*with_previous {
                return;
            }
            if let Some(pos) = doc.blocks.iter().position(|b| b.id == *block_id) {
                if pos == 0 {
                    return;
                }
                let mut tail = std::mem::take(&mut doc.blocks[pos].inlines);
                doc.blocks[pos - 1].inlines.append(&mut tail);
                coalesce_adjacent(&mut doc.blocks[pos - 1].inlines);
                doc.blocks.remove(pos);
            }
        }
        DocOp::SetMarksRange {
            block_id,
            start,
            end,
            bold,
            italic,
            underline,
            strike,
            link_url,
            font_family,
            font_size_pt,
            color,
            highlight,
            superscript,
            subscript,
        } => {
            if let Some(block) = doc.blocks.iter_mut().find(|b| b.id == *block_id) {
                let patch = MarksPatch {
                    bold: *bold,
                    italic: *italic,
                    underline: *underline,
                    strike: *strike,
                    link_url: opt_string_patch(link_url),
                    font_family: opt_string_patch(font_family),
                    font_size_pt: opt_u32_patch(font_size_pt),
                    color: opt_string_patch(color),
                    highlight: opt_string_patch(highlight),
                    superscript: *superscript,
                    subscript: *subscript,
                };
                apply_marks_range(&mut block.inlines, *start, *end, &patch);
            }
        }
        DocOp::SetInlineMarks {
            block_id,
            bold,
            italic,
            underline,
            link_url,
            strike,
            superscript,
            subscript,
        } => {
            if let Some(block) = doc.blocks.iter_mut().find(|b| b.id == *block_id) {
                let end = block_plain(&block.inlines).len();
                let patch = MarksPatch {
                    bold: Some(*bold),
                    italic: Some(*italic),
                    underline: Some(*underline),
                    strike: Some(*strike),
                    link_url: match link_url {
                        None => None,
                        Some(u) if u.is_empty() => Some(None),
                        Some(u) => Some(Some(u.clone())),
                    },
                    font_family: None,
                    font_size_pt: None,
                    color: None,
                    highlight: None,
                    superscript: Some(*superscript),
                    subscript: Some(*subscript),
                };
                apply_marks_range(&mut block.inlines, 0, end, &patch);
            }
        }
        DocOp::SetBlockFormat {
            block_id,
            align,
            indent_mm,
            line_spacing,
            space_before_pt,
            space_after_pt,
            list_type,
            clear_list,
            list_level,
            list_marker,
            list_restart,
            style_name,
        } => {
            if let Some(block) = doc.blocks.iter_mut().find(|b| b.id == *block_id) {
                if let Some(a) = align {
                    block.align = *a;
                }
                if let Some(v) = indent_mm {
                    block.indent_mm = Some(*v);
                }
                if let Some(v) = line_spacing {
                    block.line_spacing = Some(v.clone());
                }
                if let Some(v) = space_before_pt {
                    block.space_before_pt = Some(*v);
                }
                if let Some(v) = space_after_pt {
                    block.space_after_pt = Some(*v);
                }
                if clear_list == &Some(true) {
                    block.list_type = None;
                    block.list_marker = None;
                    block.list_level = 0;
                    block.list_restart = false;
                    if block.block_type == BlockType::ListItem {
                        block.block_type = BlockType::Paragraph;
                    }
                } else if list_type.is_some() {
                    block.list_type = *list_type;
                    if block.list_type.is_some() {
                        block.block_type = BlockType::ListItem;
                    }
                }
                if let Some(v) = list_level {
                    block.list_level = *v;
                }
                if list_marker.is_some() {
                    block.list_marker = *list_marker;
                }
                if let Some(v) = list_restart {
                    block.list_restart = *v;
                }
                if let Some(v) = style_name {
                    block.style_name = if v.is_empty() {
                        None
                    } else {
                        Some(v.clone())
                    };
                }
            }
        }
        DocOp::AddComment {
            id,
            block_id,
            author_id,
            text,
            start,
            end,
            quote,
        } => {
            if doc.comments.iter().any(|c| c.id == *id) {
                return;
            }
            doc.comments.push(DocComment {
                id: id.clone(),
                block_id: block_id.clone(),
                author_id: author_id.clone(),
                text: text.clone(),
                resolved: false,
                start: *start,
                end: *end,
                quote: quote.clone(),
            });
        }
        DocOp::ResolveComment { id } => {
            if let Some(c) = doc.comments.iter_mut().find(|c| c.id == *id) {
                c.resolved = true;
            }
        }
    }
}

/// Apply concurrent ops with insert-OT transform against already-applied peers.
pub fn merge_concurrent(mut base: EradDocument, ops: &[DocOp]) -> EradDocument {
    let mut applied: Vec<DocOp> = Vec::with_capacity(ops.len());
    for op in ops {
        let mut transformed = op.clone();
        for prev in &applied {
            transformed = transformed.transform_against(prev);
        }
        apply_op(&mut base, &transformed);
        applied.push(transformed);
    }
    base
}

#[cfg(test)]
mod sync_tests {
    use super::*;

    #[test]
    fn sync_merge_two_clients() {
        let mut doc = EradDocument::empty();
        let block_id = doc.blocks[0].id.clone();
        doc.blocks[0].inlines[0].text = "Hello".into();

        let mut log = OpLog::default();
        log.append(DocOp::InsertText {
            block_id: block_id.clone(),
            offset: 5,
            text: " world".into(),
                    marks: None,
        });
        log.append(DocOp::InsertText {
            block_id: block_id.clone(),
            offset: 5,
            text: "!".into(),
                    marks: None,
        });

        let merged = log.replay(doc);
        assert!(merged.plain_text().contains("Hello"));
        assert!(merged.plain_text().contains("world"));
    }

    #[test]
    fn sync_replay_since_version() {
        let doc = EradDocument::empty();
        let block_id = doc.blocks[0].id.clone();
        let mut log = OpLog::default();
        log.append(DocOp::InsertText {
            block_id,
            offset: 0,
            text: "A".into(),
                    marks: None,
        });
        assert_eq!(log.ops_since(0).len(), 1);
        assert_eq!(log.ops_since(1).len(), 0);
    }

    #[test]
    fn sync_set_block_type_heading() {
        let doc = EradDocument::empty();
        let block_id = doc.blocks[0].id.clone();
        let mut log = OpLog::default();
        log.append(DocOp::SetBlockType {
            block_id,
            block_type: BlockType::Heading,
            heading_level: 2,
        });
        let out = log.replay(doc);
        assert_eq!(out.blocks[0].block_type, BlockType::Heading);
        assert_eq!(out.blocks[0].heading_level, 2);
    }

    #[test]
    fn sync_set_inline_marks_bold() {
        let mut doc = EradDocument::empty();
        let block_id = doc.blocks[0].id.clone();
        doc.blocks[0].inlines[0].text = "Hi".into();
        apply_op(
            &mut doc,
            &DocOp::SetInlineMarks {
                block_id: block_id.clone(),
                bold: true,
                italic: false,
                underline: true,
                link_url: Some("https://era.local".into()),
                strike: false,
                superscript: false,
                subscript: false,
            },
        );
        assert!(doc.blocks[0].inlines[0].bold);
        assert!(!doc.blocks[0].inlines[0].italic);
        assert!(doc.blocks[0].inlines[0].underline);
        assert_eq!(
            doc.blocks[0].inlines[0].link_url.as_deref(),
            Some("https://era.local")
        );
        assert_eq!(doc.blocks[0].inlines[0].text, "Hi");
        apply_op(
            &mut doc,
            &DocOp::AddComment {
                id: "c1".into(),
                block_id,
                author_id: "u1".into(),
                text: "note".into(),
                start: Some(0),
                end: Some(2),
                quote: Some("Hi".into()),
            },
        );
        assert_eq!(doc.comments.len(), 1);
        assert_eq!(doc.comments[0].text, "note");
        assert_eq!(doc.comments[0].start, Some(0));
        assert_eq!(doc.comments[0].end, Some(2));
        assert_eq!(doc.comments[0].quote.as_deref(), Some("Hi"));
    }

    #[test]
    fn mid_utf8_offset_insert_no_panic() {
        let mut doc = EradDocument::empty();
        let block_id = doc.blocks[0].id.clone();
        // "€" is 3 bytes; offset 1 is mid-rune
        doc.blocks[0].inlines[0].text = "€x".into();
        apply_op(
            &mut doc,
            &DocOp::InsertText {
                block_id: block_id.clone(),
                offset: 1,
                text: "!".into(),
                marks: None,
            },
        );
        let plain = doc.plain_text();
        assert!(plain.contains('€'));
        assert!(plain.contains('!'));
        assert!(plain.contains('x'));
    }

    #[test]
    fn mid_utf8_offset_delete_no_panic() {
        let mut doc = EradDocument::empty();
        let block_id = doc.blocks[0].id.clone();
        // bytes: a=0, €=1..3, b=4
        doc.blocks[0].inlines[0].text = "a€b".into();
        apply_op(
            &mut doc,
            &DocOp::DeleteRange {
                block_id,
                start: 2, // mid-"€" → floor to 1
                end: 4,   // char boundary at 'b'
            },
        );
        // Deletes "€" only → "ab"
        assert_eq!(doc.plain_text(), "ab");
    }

    #[test]
    fn concurrent_inserts_same_offset_both_appear() {
        let mut doc = EradDocument::empty();
        let block_id = doc.blocks[0].id.clone();
        doc.blocks[0].inlines[0].text = "Hello".into();

        let ops = [
            DocOp::InsertText {
                block_id: block_id.clone(),
                offset: 5,
                text: "A".into(),
                marks: None,
            },
            DocOp::InsertText {
                block_id: block_id.clone(),
                offset: 5,
                text: "B".into(),
                marks: None,
            },
        ];
        let merged = merge_concurrent(doc, &ops);
        let plain = merged.plain_text();
        assert!(plain.contains('A'), "missing A: {plain}");
        assert!(plain.contains('B'), "missing B: {plain}");
        // Order-stable: first-applied then second → HelloAB
        assert_eq!(plain, "HelloAB");
    }

    #[test]
    fn delete_range_transforms_against_insert() {
        let del = DocOp::DeleteRange {
            block_id: "b1".into(),
            start: 2,
            end: 4,
        };
        let ins = DocOp::InsertText {
            block_id: "b1".into(),
            offset: 0,
            text: "XX".into(),
                    marks: None,
        };
        let t = del.transform_against(&ins);
        match t {
            DocOp::DeleteRange { start, end, .. } => {
                assert_eq!(start, 4);
                assert_eq!(end, 6);
            }
            other => panic!("unexpected {other:?}"),
        }
    }

    #[test]
    fn fmt_enrichment_block_format_list_and_spacing() {
        let mut doc = EradDocument::empty();
        let block_id = doc.blocks[0].id.clone();
        apply_op(
            &mut doc,
            &DocOp::SetBlockFormat {
                block_id: block_id.clone(),
                align: Some(TextAlign::Justify),
                indent_mm: Some(12),
                line_spacing: Some("1.5".into()),
                space_before_pt: Some(6),
                space_after_pt: Some(12),
                clear_list: None,
                list_type: Some(ListType::Ordered),
                list_level: Some(1),
                list_marker: Some(ListMarker::Decimal),
                list_restart: Some(true),
                style_name: Some("quote".into()),
            },
        );
        let b = &doc.blocks[0];
        assert_eq!(b.align, TextAlign::Justify);
        assert_eq!(b.indent_mm, Some(12));
        assert_eq!(b.line_spacing.as_deref(), Some("1.5"));
        assert_eq!(b.space_before_pt, Some(6));
        assert_eq!(b.space_after_pt, Some(12));
        assert_eq!(b.list_type, Some(ListType::Ordered));
        assert_eq!(b.list_level, 1);
        assert_eq!(b.list_marker, Some(ListMarker::Decimal));
        assert!(b.list_restart);
        assert_eq!(b.style_name.as_deref(), Some("quote"));
        assert_eq!(b.block_type, BlockType::ListItem);
    }

    #[test]
    fn fmt_enrichment_superscript_marks() {
        let mut doc = EradDocument::empty();
        let block_id = doc.blocks[0].id.clone();
        doc.blocks[0].inlines[0].text = "H2O".into();
        apply_op(
            &mut doc,
            &DocOp::SetInlineMarks {
                block_id,
                bold: false,
                italic: false,
                underline: false,
                link_url: None,
                strike: false,
                superscript: true,
                subscript: false,
            },
        );
        assert!(doc.blocks[0].inlines[0].superscript);
        assert!(!doc.blocks[0].inlines[0].subscript);
    }

    #[test]
    fn split_block_mid_multi_span() {
        let mut doc = EradDocument::empty();
        let block_id = doc.blocks[0].id.clone();
        doc.blocks[0].inlines = vec![
            InlineSpan::plain("Hi"),
            InlineSpan {
                text: "There".into(),
                bold: true,
                ..InlineSpan::plain("")
            },
        ];
        doc.blocks[0].align = TextAlign::Center;
        apply_op(
            &mut doc,
            &DocOp::SplitBlock {
                block_id: block_id.clone(),
                offset: 2,
                new_block_id: "b-new".into(),
            },
        );
        assert_eq!(doc.blocks.len(), 2);
        assert_eq!(block_plain(&doc.blocks[0].inlines), "Hi");
        assert_eq!(block_plain(&doc.blocks[1].inlines), "There");
        assert!(doc.blocks[1].inlines[0].bold);
        assert_eq!(doc.blocks[1].align, TextAlign::Center);
        assert_eq!(doc.blocks[1].id, "b-new");
    }

    #[test]
    fn merge_blocks_coalesces_runs() {
        let mut doc = EradDocument::empty();
        let a = doc.blocks[0].id.clone();
        doc.blocks[0].inlines = vec![InlineSpan::plain("A")];
        doc.blocks.push(Block::paragraph("b2", "B"));
        apply_op(
            &mut doc,
            &DocOp::MergeBlocks {
                block_id: "b2".into(),
                with_previous: true,
            },
        );
        assert_eq!(doc.blocks.len(), 1);
        assert_eq!(doc.blocks[0].id, a);
        assert_eq!(block_plain(&doc.blocks[0].inlines), "AB");
    }

    #[test]
    fn set_marks_range_preserves_neighbors() {
        let mut doc = EradDocument::empty();
        let block_id = doc.blocks[0].id.clone();
        doc.blocks[0].inlines = vec![InlineSpan::plain("abcdef")];
        apply_op(
            &mut doc,
            &DocOp::SetMarksRange {
                block_id,
                start: 2,
                end: 4,
                bold: Some(true),
                italic: None,
                underline: None,
                strike: None,
                link_url: None,
                font_family: Some("Georgia".into()),
                font_size_pt: None,
                color: None,
                highlight: None,
                superscript: None,
                subscript: None,
            },
        );
        let inl = &doc.blocks[0].inlines;
        assert_eq!(inl.len(), 3);
        assert_eq!(inl[0].text, "ab");
        assert!(!inl[0].bold);
        assert_eq!(inl[1].text, "cd");
        assert!(inl[1].bold);
        assert_eq!(inl[1].font_family.as_deref(), Some("Georgia"));
        assert_eq!(inl[2].text, "ef");
        assert!(!inl[2].bold);
    }

    #[test]
    fn insert_text_preserves_bold_neighbor() {
        let mut doc = EradDocument::empty();
        let block_id = doc.blocks[0].id.clone();
        doc.blocks[0].inlines = vec![
            InlineSpan::plain("Hello"),
            InlineSpan {
                text: "World".into(),
                bold: true,
                ..InlineSpan::plain("")
            },
        ];
        apply_op(
            &mut doc,
            &DocOp::InsertText {
                block_id,
                offset: 5,
                text: " ".into(),
                marks: None,
            },
        );
        assert_eq!(doc.blocks[0].inlines.len(), 2);
        assert_eq!(doc.blocks[0].inlines[0].text, "Hello ");
        assert!(!doc.blocks[0].inlines[0].bold);
        assert!(doc.blocks[0].inlines[1].bold);
    }

    #[test]
    fn marks_range_transforms_against_insert() {
        let marks = DocOp::SetMarksRange {
            block_id: "b1".into(),
            start: 2,
            end: 5,
            bold: Some(true),
            italic: None,
            underline: None,
            strike: None,
            link_url: None,
            font_family: None,
            font_size_pt: None,
            color: None,
            highlight: None,
            superscript: None,
            subscript: None,
        };
        let ins = DocOp::InsertText {
            block_id: "b1".into(),
            offset: 0,
            text: "XX".into(),
            marks: None,
        };
        let t = marks.transform_against(&ins);
        match t {
            DocOp::SetMarksRange { start, end, .. } => {
                assert_eq!(start, 4);
                assert_eq!(end, 7);
            }
            other => panic!("unexpected {other:?}"),
        }
    }

    #[test]
    fn merge_concurrent_preserves_multi_span_marks() {
        let mut doc = EradDocument::empty();
        let block_id = doc.blocks[0].id.clone();
        doc.blocks[0].inlines = vec![InlineSpan::plain("abcdef")];
        apply_op(
            &mut doc,
            &DocOp::SetMarksRange {
                block_id: block_id.clone(),
                start: 2,
                end: 4,
                bold: Some(true),
                italic: None,
                underline: None,
                strike: None,
                link_url: None,
                font_family: None,
                font_size_pt: None,
                color: None,
                highlight: None,
                superscript: None,
                subscript: None,
            },
        );
        let ops = [DocOp::InsertText {
            block_id,
            offset: 0,
            text: "!".into(),
            marks: None,
        }];
        let merged = merge_concurrent(doc, &ops);
        assert!(merged.blocks[0].inlines.iter().any(|s| s.bold && s.text == "cd"));
        assert!(merged.plain_text().starts_with('!'));
    }
}
