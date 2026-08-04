use serde::{Deserialize, Serialize};

use crate::frame::TextFrame;
use crate::model::{Block, BlockType, InlineSpan, ListMarker, ListType, TextAlign};
use crate::spans::{
    apply_marks_range, coalesce_adjacent, delete_range_preserving, insert_text_preserving,
    split_inlines_at, MarksPatch,
};

/// Ops that mutate a single [`TextFrame`] (presentation shape / docs subset).
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(tag = "type", rename_all = "snake_case")]
pub enum FrameOp {
    InsertText {
        block_id: String,
        offset: usize,
        text: String,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        marks: Option<InlineSpan>,
    },
    DeleteRange {
        block_id: String,
        start: usize,
        end: usize,
    },
    SplitBlock {
        block_id: String,
        offset: usize,
        new_block_id: String,
    },
    MergeBlocks {
        block_id: String,
        #[serde(default = "default_true")]
        with_previous: bool,
    },
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
    SetBlockFormat {
        block_id: String,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        align: Option<TextAlign>,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        clear_list: Option<bool>,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        list_type: Option<ListType>,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        list_level: Option<u32>,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        list_marker: Option<ListMarker>,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        style_name: Option<String>,
    },
    InsertBlock {
        after_id: String,
        block: Block,
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
        inlines,
    }
}

pub fn apply_frame_op(frame: &mut TextFrame, op: &FrameOp) {
    frame.ensure_nonempty();
    match op {
        FrameOp::InsertText {
            block_id,
            offset,
            text,
            marks,
        } => {
            if let Some(block) = frame.blocks.iter_mut().find(|b| b.id == *block_id) {
                insert_text_preserving(&mut block.inlines, *offset, text, marks.as_ref());
            }
        }
        FrameOp::DeleteRange {
            block_id,
            start,
            end,
        } => {
            if let Some(block) = frame.blocks.iter_mut().find(|b| b.id == *block_id) {
                delete_range_preserving(&mut block.inlines, *start, *end);
            }
        }
        FrameOp::SplitBlock {
            block_id,
            offset,
            new_block_id,
        } => {
            if frame.blocks.iter().any(|b| b.id == *new_block_id) {
                return;
            }
            if let Some(pos) = frame.blocks.iter().position(|b| b.id == *block_id) {
                let (left, right) = split_inlines_at(&frame.blocks[pos].inlines, *offset);
                let attrs = frame.blocks[pos].clone();
                frame.blocks[pos].inlines = left;
                frame
                    .blocks
                    .insert(pos + 1, copy_block_attrs(&attrs, new_block_id.clone(), right));
            }
        }
        FrameOp::MergeBlocks {
            block_id,
            with_previous,
        } => {
            if !*with_previous {
                return;
            }
            if let Some(pos) = frame.blocks.iter().position(|b| b.id == *block_id) {
                if pos == 0 {
                    return;
                }
                let mut tail = std::mem::take(&mut frame.blocks[pos].inlines);
                frame.blocks[pos - 1].inlines.append(&mut tail);
                coalesce_adjacent(&mut frame.blocks[pos - 1].inlines);
                frame.blocks.remove(pos);
            }
        }
        FrameOp::SetMarksRange {
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
            if let Some(block) = frame.blocks.iter_mut().find(|b| b.id == *block_id) {
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
        FrameOp::SetBlockFormat {
            block_id,
            align,
            clear_list,
            list_type,
            list_level,
            list_marker,
            style_name,
        } => {
            if let Some(block) = frame.blocks.iter_mut().find(|b| b.id == *block_id) {
                if let Some(a) = align {
                    block.align = *a;
                }
                if clear_list == &Some(true) {
                    block.list_type = None;
                    block.list_marker = None;
                    block.list_level = 0;
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
                if let Some(v) = style_name {
                    block.style_name = if v.is_empty() {
                        None
                    } else {
                        Some(v.clone())
                    };
                }
            }
        }
        FrameOp::InsertBlock { after_id, block } => {
            if let Some(pos) = frame.blocks.iter().position(|b| b.id == *after_id) {
                frame.blocks.insert(pos + 1, block.clone());
            } else {
                frame.blocks.push(block.clone());
            }
        }
    }
}

/// Lite OT: shift offsets for insert/delete on the same block.
pub fn transform_frame_op(op: &FrameOp, other: &FrameOp) -> FrameOp {
    match (op, other) {
        (
            FrameOp::InsertText {
                block_id,
                offset,
                text,
                marks,
            },
            FrameOp::InsertText {
                block_id: ob,
                offset: oo,
                text: ot,
                ..
            },
        ) if block_id == ob => FrameOp::InsertText {
            block_id: block_id.clone(),
            offset: if *offset >= *oo {
                offset + ot.len()
            } else {
                *offset
            },
            text: text.clone(),
            marks: marks.clone(),
        },
        (
            FrameOp::SetMarksRange {
                block_id,
                start,
                end,
                ..
            },
            FrameOp::InsertText {
                block_id: ob,
                offset: oo,
                text: ot,
                ..
            },
        ) if block_id == ob => {
            let mut next = op.clone();
            if let FrameOp::SetMarksRange {
                start: s, end: e, ..
            } = &mut next
            {
                if *oo <= *s {
                    *s += ot.len();
                    *e += ot.len();
                } else if *oo < *e {
                    *e += ot.len();
                }
            }
            next
        }
        (
            FrameOp::SplitBlock {
                block_id,
                offset,
                new_block_id,
            },
            FrameOp::InsertText {
                block_id: ob,
                offset: oo,
                text: ot,
                ..
            },
        ) if block_id == ob => FrameOp::SplitBlock {
            block_id: block_id.clone(),
            offset: if *offset >= *oo {
                offset + ot.len()
            } else {
                *offset
            },
            new_block_id: new_block_id.clone(),
        },
        (_, FrameOp::MergeBlocks { .. }) | (FrameOp::MergeBlocks { .. }, _) => op.clone(),
        _ => op.clone(),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::spans::block_plain;

    #[test]
    fn split_and_marks_on_frame() {
        let mut frame = TextFrame::from_plain("abcdef");
        let id = frame.blocks[0].id.clone();
        apply_frame_op(
            &mut frame,
            &FrameOp::SetMarksRange {
                block_id: id.clone(),
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
        apply_frame_op(
            &mut frame,
            &FrameOp::SplitBlock {
                block_id: id,
                offset: 3,
                new_block_id: "n2".into(),
            },
        );
        assert_eq!(frame.blocks.len(), 2);
        assert_eq!(block_plain(&frame.blocks[0].inlines), "abc");
        assert_eq!(block_plain(&frame.blocks[1].inlines), "def");
    }
}
