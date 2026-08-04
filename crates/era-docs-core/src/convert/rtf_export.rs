//! RTF export — thicker lite (tables, strike, header/footer, image stubs).

use crate::model::{Block, BlockType, EradDocument, ListType};

fn rtf_escape(s: &str) -> String {
    let mut out = String::with_capacity(s.len());
    for ch in s.chars() {
        match ch {
            '\\' => out.push_str("\\\\"),
            '{' => out.push_str("\\{"),
            '}' => out.push_str("\\}"),
            '\n' => out.push_str("\\par "),
            c if (c as u32) > 127 => {
                out.push_str(&format!("\\u{}?", c as u32));
            }
            c => out.push(c),
        }
    }
    out
}

fn table_rows(block: &Block) -> Vec<Vec<String>> {
    if let Some(ref cells) = block.table_cells {
        if !cells.is_empty() {
            return cells.clone();
        }
    }
    let text: String = block.inlines.iter().map(|i| i.text.as_str()).collect();
    text.lines()
        .filter(|l| !l.is_empty())
        .map(|row| row.split('\t').map(|c| c.to_string()).collect())
        .collect()
}

fn emit_table(body: &mut String, block: &Block) {
    let rows = table_rows(block);
    if rows.is_empty() {
        body.push_str(r"\par [table]");
        return;
    }
    let cols = rows.iter().map(|r| r.len()).max().unwrap_or(1).max(1);
    for row in rows {
        body.push_str(r"\trowd\trgaph108");
        for i in 0..cols {
            let x = (i + 1) * 1440;
            body.push_str(&format!(r"\cellx{x}"));
        }
        body.push(' ');
        for i in 0..cols {
            let cell = row.get(i).map(|s| s.as_str()).unwrap_or("");
            body.push_str(r"\intbl ");
            body.push_str(&rtf_escape(cell));
            body.push_str(r"\cell ");
        }
        body.push_str(r"\row ");
    }
}

fn emit_spans(body: &mut String, block: &Block) {
    for span in &block.inlines {
        if span.bold {
            body.push_str(r"\b ");
        }
        if span.italic {
            body.push_str(r"\i ");
        }
        if span.underline {
            body.push_str(r"\ul ");
        }
        if span.strike {
            body.push_str(r"\strike ");
        }
        body.push_str(&rtf_escape(&span.text));
        if span.strike {
            body.push_str(r"\strike0 ");
        }
        if span.underline {
            body.push_str(r"\ul0 ");
        }
        if span.italic {
            body.push_str(r"\i0 ");
        }
        if span.bold {
            body.push_str(r"\b0 ");
        }
    }
}

/// Export a thicker-lite RTF document from ERA blocks.
pub fn export_rtf(doc: &EradDocument) -> Vec<u8> {
    let mut body = String::from(r"{\rtf1\ansi\deff0{\fonttbl{\f0 Arial;}}\f0\fs24 ");
    if !doc.header.text.is_empty() {
        body.push_str(r"{\header ");
        body.push_str(&rtf_escape(&doc.header.text));
        body.push_str(r"}\par ");
    }
    for block in &doc.blocks {
        match block.block_type {
            BlockType::PageBreak => {
                body.push_str(r"\page ");
                continue;
            }
            BlockType::Image => {
                let stub = match &block.image_url {
                    Some(url) if !url.is_empty() => format!("[image:{url}]"),
                    _ => {
                        let t: String = block.inlines.iter().map(|i| i.text.as_str()).collect();
                        if t.is_empty() {
                            "[image]".into()
                        } else {
                            format!("[image:{t}]")
                        }
                    }
                };
                body.push_str(r"\par ");
                body.push_str(&rtf_escape(&stub));
                continue;
            }
            BlockType::Table => {
                emit_table(&mut body, block);
                continue;
            }
            BlockType::Toc => {
                let t: String = block.inlines.iter().map(|i| i.text.as_str()).collect();
                body.push_str(r"\par ");
                body.push_str(&rtf_escape(&t));
                continue;
            }
            BlockType::Bookmark => {
                let name = block.bookmark_name.as_deref().unwrap_or("bookmark");
                body.push_str(r"\par ");
                body.push_str(&rtf_escape(&format!("[{name}]")));
                continue;
            }
            _ => {}
        }
        if matches!(block.block_type, BlockType::Heading) {
            let lvl = block.heading_level.max(1).min(6);
            let fs = 36 - (lvl.saturating_sub(1) * 4);
            body.push_str(&format!(r"\par\b\fs{fs} "));
        } else if matches!(block.block_type, BlockType::TextBox) {
            body.push_str(r"\par\box\brdrs ");
        } else if matches!(block.block_type, BlockType::ListItem) {
            let bullet = match block.list_type {
                Some(ListType::Ordered) => r"\par\pntext 1. ",
                _ => r"\par\bullet ",
            };
            body.push_str(bullet);
        } else if block.style_name.as_deref() == Some("horizontal_line") {
            body.push_str(r"\par\brdrb\brdrs\brdrw10\brsp20 ");
        } else {
            body.push_str(r"\par ");
        }
        emit_spans(&mut body, block);
        if matches!(block.block_type, BlockType::Heading) {
            body.push_str(r"\b0\fs24 ");
        }
    }
    if !doc.footer.text.is_empty() {
        body.push_str(r"{\footer ");
        body.push_str(&rtf_escape(&doc.footer.text));
        body.push('}');
    }
    body.push('}');
    body.into_bytes()
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::model::{Block, InlineSpan, TextAlign};

    fn blk(id: &str, block_type: BlockType, inlines: Vec<InlineSpan>) -> Block {
        Block {
            id: id.into(),
            block_type,
            heading_level: 0,
            list_type: None,
            align: TextAlign::Left,
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
            inlines,
        }
    }

    #[test]
    fn export_rtf_contains_text() {
        let mut doc = EradDocument::empty();
        let mut span = InlineSpan::plain("Hello RTF");
        span.bold = true;
        doc.blocks = vec![blk("b1", BlockType::Paragraph, vec![span])];
        let bytes = export_rtf(&doc);
        let s = String::from_utf8_lossy(&bytes);
        assert!(s.contains(r"{\rtf1"));
        assert!(s.contains("Hello RTF"));
        assert!(s.contains(r"\b "));
    }

    /// Golden: fixed EradDocument → key substrings in RTF bytes.
    #[test]
    fn export_rtf_olite_p1_golden_substrings() {
        let mut doc = EradDocument::empty();
        doc.header.text = "HdrRtf".into();
        doc.footer.text = "FtrRtf".into();

        let mut h = blk("h", BlockType::Heading, vec![InlineSpan::plain("Title")]);
        h.heading_level = 1;

        let mut strike = InlineSpan::plain("gone");
        strike.strike = true;

        let mut table = blk("t", BlockType::Table, vec![]);
        table.table_cells = Some(vec![
            vec!["R1C1".into(), "R1C2".into()],
            vec!["R2C1".into(), "R2C2".into()],
        ]);

        let mut img = blk("img", BlockType::Image, vec![]);
        img.image_url = Some("https://era.local/a.png".into());

        doc.blocks = vec![
            h,
            blk("p", BlockType::Paragraph, vec![strike]),
            table,
            img,
        ];

        let bytes = export_rtf(&doc);
        let s = String::from_utf8_lossy(&bytes);
        assert!(s.contains(r"{\rtf1"));
        assert!(s.contains(r"{\header "));
        assert!(s.contains("HdrRtf"));
        assert!(s.contains(r"{\footer "));
        assert!(s.contains("FtrRtf"));
        assert!(s.contains(r"\b\fs36 "));
        assert!(s.contains("Title"));
        assert!(s.contains(r"\strike "));
        assert!(s.contains("gone"));
        assert!(s.contains(r"\trowd"));
        assert!(s.contains(r"\cell"));
        assert!(s.contains(r"\row"));
        assert!(s.contains("R1C1"));
        assert!(s.contains("R2C2"));
        assert!(s.contains("[image:https://era.local/a.png]"));
    }
}
