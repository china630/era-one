//! Span-aware text editing for `Block.inlines` (no flatten).

use crate::model::InlineSpan;

/// Floor `offset` to a UTF-8 char boundary (never mid-rune). Caps at `s.len()`.
pub fn floor_char_boundary(s: &str, offset: usize) -> usize {
    if offset >= s.len() {
        return s.len();
    }
    s.char_indices()
        .map(|(i, _)| i)
        .take_while(|&i| i <= offset)
        .last()
        .unwrap_or(0)
}

pub fn block_plain(inlines: &[InlineSpan]) -> String {
    inlines.iter().map(|s| s.text.as_str()).collect()
}

fn marks_eq(a: &InlineSpan, b: &InlineSpan) -> bool {
    a.bold == b.bold
        && a.italic == b.italic
        && a.underline == b.underline
        && a.strike == b.strike
        && a.link_url == b.link_url
        && a.font_family == b.font_family
        && a.font_size_pt == b.font_size_pt
        && a.color == b.color
        && a.highlight == b.highlight
        && a.superscript == b.superscript
        && a.subscript == b.subscript
}

fn with_text(marks: &InlineSpan, text: String) -> InlineSpan {
    InlineSpan {
        text,
        bold: marks.bold,
        italic: marks.italic,
        underline: marks.underline,
        strike: marks.strike,
        link_url: marks.link_url.clone(),
        font_family: marks.font_family.clone(),
        font_size_pt: marks.font_size_pt,
        color: marks.color.clone(),
        highlight: marks.highlight.clone(),
        superscript: marks.superscript,
        subscript: marks.subscript,
    }
}

/// Merge adjacent spans that share identical marks.
pub fn coalesce_adjacent(inlines: &mut Vec<InlineSpan>) {
    if inlines.is_empty() {
        return;
    }
    let mut out: Vec<InlineSpan> = Vec::with_capacity(inlines.len());
    for span in inlines.drain(..) {
        if span.text.is_empty() {
            continue;
        }
        if let Some(last) = out.last_mut() {
            if marks_eq(last, &span) {
                last.text.push_str(&span.text);
                continue;
            }
        }
        out.push(span);
    }
    if out.is_empty() {
        out.push(InlineSpan::plain(""));
    }
    *inlines = out;
}

/// Split `inlines` so that `offset` falls on a span boundary. Returns index of the
/// span that starts at `offset` (or `inlines.len()` if offset == total len).
pub fn split_at(inlines: &mut Vec<InlineSpan>, offset: usize) -> usize {
    let plain = block_plain(inlines);
    let offset = floor_char_boundary(&plain, offset);
    if offset == 0 {
        return 0;
    }
    if offset >= plain.len() {
        return inlines.len();
    }
    let mut acc = 0usize;
    let mut i = 0usize;
    while i < inlines.len() {
        let len = inlines[i].text.len();
        if acc + len == offset {
            return i + 1;
        }
        if acc + len > offset {
            let local = offset - acc;
            let local = floor_char_boundary(&inlines[i].text, local);
            let right_text = inlines[i].text[local..].to_string();
            inlines[i].text.truncate(local);
            let right = with_text(&inlines[i], right_text);
            inlines.insert(i + 1, right);
            if inlines[i].text.is_empty() {
                inlines.remove(i);
                return i;
            }
            return i + 1;
        }
        acc += len;
        i += 1;
    }
    inlines.len()
}

/// Slice inlines into `[0..offset)` and `[offset..)`.
pub fn split_inlines_at(inlines: &[InlineSpan], offset: usize) -> (Vec<InlineSpan>, Vec<InlineSpan>) {
    let mut left = inlines.to_vec();
    let idx = split_at(&mut left, offset);
    let right = left.split_off(idx);
    coalesce_adjacent(&mut left);
    let mut right = right;
    coalesce_adjacent(&mut right);
    (left, right)
}

/// Optional mark patch: `None` = leave unchanged; for strings `Some("")` clears.
#[derive(Debug, Clone, Default)]
pub struct MarksPatch {
    pub bold: Option<bool>,
    pub italic: Option<bool>,
    pub underline: Option<bool>,
    pub strike: Option<bool>,
    pub link_url: Option<Option<String>>,
    pub font_family: Option<Option<String>>,
    pub font_size_pt: Option<Option<u32>>,
    pub color: Option<Option<String>>,
    pub highlight: Option<Option<String>>,
    pub superscript: Option<bool>,
    pub subscript: Option<bool>,
}

fn apply_patch(span: &mut InlineSpan, patch: &MarksPatch) {
    if let Some(v) = patch.bold {
        span.bold = v;
    }
    if let Some(v) = patch.italic {
        span.italic = v;
    }
    if let Some(v) = patch.underline {
        span.underline = v;
    }
    if let Some(v) = patch.strike {
        span.strike = v;
    }
    if let Some(ref v) = patch.link_url {
        span.link_url = v.clone();
    }
    if let Some(ref v) = patch.font_family {
        span.font_family = v.clone();
    }
    if let Some(v) = patch.font_size_pt {
        span.font_size_pt = v;
    }
    if let Some(ref v) = patch.color {
        span.color = v.clone();
    }
    if let Some(ref v) = patch.highlight {
        span.highlight = v.clone();
    }
    if let Some(v) = patch.superscript {
        span.superscript = v;
    }
    if let Some(v) = patch.subscript {
        span.subscript = v;
    }
}

pub fn apply_marks_range(inlines: &mut Vec<InlineSpan>, start: usize, end: usize, patch: &MarksPatch) {
    let plain = block_plain(inlines);
    let start = floor_char_boundary(&plain, start);
    let end = floor_char_boundary(&plain, end).max(start);
    if start >= end {
        return;
    }
    // Split end first so earlier indices stay stable when we split start.
    let _ = split_at(inlines, end);
    let left_idx = split_at(inlines, start);
    let end_idx = split_at(inlines, end);
    for span in inlines.iter_mut().take(end_idx).skip(left_idx) {
        apply_patch(span, patch);
    }
    coalesce_adjacent(inlines);
}

/// Insert `text` at `offset`, inheriting marks from the span at the caret
/// (or `typing` if provided).
pub fn insert_text_preserving(
    inlines: &mut Vec<InlineSpan>,
    offset: usize,
    text: &str,
    typing: Option<&InlineSpan>,
) {
    if text.is_empty() {
        return;
    }
    if inlines.is_empty() {
        inlines.push(match typing {
            Some(m) => with_text(m, text.to_string()),
            None => InlineSpan::plain(text),
        });
        return;
    }
    let plain = block_plain(inlines);
    let offset = floor_char_boundary(&plain, offset);
    let idx = split_at(inlines, offset);
    let template = if let Some(t) = typing {
        t.clone()
    } else if idx > 0 {
        inlines[idx - 1].clone()
    } else if idx < inlines.len() {
        inlines[idx].clone()
    } else {
        InlineSpan::plain("")
    };
    inlines.insert(idx, with_text(&template, text.to_string()));
    coalesce_adjacent(inlines);
}

pub fn delete_range_preserving(inlines: &mut Vec<InlineSpan>, start: usize, end: usize) {
    let plain = block_plain(inlines);
    let start = floor_char_boundary(&plain, start);
    let end = floor_char_boundary(&plain, end).max(start);
    if start >= end {
        return;
    }
    let _ = split_at(inlines, end);
    let start_idx = split_at(inlines, start);
    let end_idx = split_at(inlines, end);
    inlines.drain(start_idx..end_idx);
    coalesce_adjacent(inlines);
}

#[cfg(test)]
mod span_tests {
    use super::*;

    fn bold(s: &str) -> InlineSpan {
        let mut sp = InlineSpan::plain(s);
        sp.bold = true;
        sp
    }

    #[test]
    fn insert_preserves_neighbors() {
        let mut inl = vec![InlineSpan::plain("Hello"), bold("World")];
        insert_text_preserving(&mut inl, 5, " ", None);
        assert_eq!(inl.len(), 2);
        assert_eq!(inl[0].text, "Hello ");
        assert!(!inl[0].bold);
        assert_eq!(inl[1].text, "World");
        assert!(inl[1].bold);
    }

    #[test]
    fn marks_range_middle() {
        let mut inl = vec![InlineSpan::plain("abcdef")];
        let mut patch = MarksPatch::default();
        patch.bold = Some(true);
        apply_marks_range(&mut inl, 2, 4, &patch);
        assert_eq!(inl.len(), 3);
        assert_eq!(inl[0].text, "ab");
        assert!(!inl[0].bold);
        assert_eq!(inl[1].text, "cd");
        assert!(inl[1].bold);
        assert_eq!(inl[2].text, "ef");
        assert!(!inl[2].bold);
    }

    #[test]
    fn split_inlines_multi_span() {
        let inl = vec![InlineSpan::plain("Hi"), bold("There")];
        let (l, r) = split_inlines_at(&inl, 2);
        assert_eq!(block_plain(&l), "Hi");
        assert_eq!(block_plain(&r), "There");
        assert!(r[0].bold);
    }
}
