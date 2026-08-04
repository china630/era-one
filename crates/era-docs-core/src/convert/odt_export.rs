//! ODT (OpenDocument Text) export — thicker lite (inline marks, breaks, tables).

use std::io::{Cursor, Write};

use anyhow::Result;
use zip::write::SimpleFileOptions;
use zip::{CompressionMethod, ZipWriter};

use crate::model::{Block, BlockType, EradDocument, InlineSpan, ListMarker, ListType, TextAlign};

const ODT_MIME: &str = "application/vnd.oasis.opendocument.text";

fn xml_escape(s: &str) -> String {
    s.chars()
        .map(|c| match c {
            '&' => "&amp;".into(),
            '<' => "&lt;".into(),
            '>' => "&gt;".into(),
            '"' => "&quot;".into(),
            c => c.to_string(),
        })
        .collect()
}

fn span_xml(span: &InlineSpan) -> String {
    let mut inner = xml_escape(&span.text);
    if span.bold {
        inner = format!("<text:span text:style-name=\"Strong\">{inner}</text:span>");
    }
    if span.italic {
        inner = format!("<text:span text:style-name=\"Emphasis\">{inner}</text:span>");
    }
    if span.underline {
        inner = format!("<text:span text:style-name=\"Underline\">{inner}</text:span>");
    }
    if span.strike {
        inner = format!("<text:span text:style-name=\"Strike\">{inner}</text:span>");
    }
    if let Some(ref url) = span.link_url {
        if !url.is_empty() {
            return format!(
                r#"<text:a xlink:href="{}">{inner}</text:a>"#,
                xml_escape(url)
            );
        }
    }
    if span.bold || span.italic || span.underline || span.strike {
        return inner;
    }
    // Plain text — still wrap when font/color present for LibreOffice friendliness.
    if span.font_family.is_some() || span.font_size_pt.is_some() || span.color.is_some() {
        return format!("<text:span>{inner}</text:span>");
    }
    inner
}

fn inlines_xml(block: &Block) -> String {
    if block.inlines.is_empty() {
        return String::new();
    }
    block.inlines.iter().map(span_xml).collect()
}

fn para_style(block: &Block) -> &'static str {
    if block.style_name.as_deref() == Some("horizontal_line") {
        return "Horizontal_20_Line";
    }
    match block.align {
        TextAlign::Center => "Center",
        TextAlign::Right => "Right",
        TextAlign::Justify => "Justify",
        TextAlign::Left => "Standard",
    }
}

fn list_style_name(block: &Block) -> &'static str {
    match block.list_marker {
        Some(ListMarker::Circle) => "BulletCircle",
        Some(ListMarker::Square) => "BulletSquare",
        Some(ListMarker::Decimal) => "Numbering",
        Some(ListMarker::LowerAlpha) => "NumberingAlpha",
        Some(ListMarker::LowerRoman) => "NumberingRoman",
        Some(ListMarker::Disc) => "Bullet",
        None => match block.list_type {
            Some(ListType::Ordered) => "Numbering",
            _ => "Bullet",
        },
    }
}

fn list_marker_prefix(block: &Block) -> &'static str {
    match block.list_marker {
        Some(ListMarker::Circle) => "○ ",
        Some(ListMarker::Square) => "■ ",
        Some(ListMarker::Disc) => "• ",
        Some(ListMarker::Decimal) => "1. ",
        Some(ListMarker::LowerAlpha) => "a. ",
        Some(ListMarker::LowerRoman) => "i. ",
        None => match block.list_type {
            Some(ListType::Ordered) => "1. ",
            _ => "• ",
        },
    }
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

fn table_xml(block: &Block) -> String {
    let rows = table_rows(block);
    if rows.is_empty() {
        return r#"<text:p text:style-name="Standard">[table]</text:p>"#.into();
    }
    let cols = rows.iter().map(|r| r.len()).max().unwrap_or(1).max(1);
    let mut xml = format!(
        r#"<table:table table:name="Table1"><table:table-column table:number-columns-repeated="{cols}"/>"#
    );
    for row in rows {
        xml.push_str("<table:table-row>");
        for cell in row {
            xml.push_str(&format!(
                r#"<table:table-cell office:value-type="string"><text:p>{}</text:p></table:table-cell>"#,
                xml_escape(&cell)
            ));
        }
        xml.push_str("</table:table-row>");
    }
    xml.push_str("</table:table>");
    xml
}

fn image_placeholder(block: &Block, inlines: &str) -> String {
    if let Some(ref url) = block.image_url {
        if !url.is_empty() {
            return format!("[image:{}]", xml_escape(url));
        }
    }
    if inlines.is_empty() {
        "[image]".into()
    } else {
        inlines.to_string()
    }
}

fn build_content_xml(doc: &EradDocument) -> String {
    let mut body = String::new();
    if !doc.header.text.is_empty() {
        body.push_str(&format!(
            r#"<text:p text:style-name="Header">{}</text:p>"#,
            xml_escape(&doc.header.text)
        ));
    }
    for block in &doc.blocks {
        let t = inlines_xml(block);
        match block.block_type {
            BlockType::PageBreak => {
                body.push_str(
                    r#"<text:p text:style-name="Standard"><text:soft-page-break/></text:p>"#,
                );
            }
            BlockType::SectionBreak => {
                body.push_str(
                    r#"<text:p text:style-name="Standard">— Section break —</text:p><text:p text:style-name="Standard"><text:soft-page-break/></text:p>"#,
                );
            }
            BlockType::Heading => {
                let lvl = block.heading_level.max(1).min(6);
                body.push_str(&format!(
                    r#"<text:h text:outline-level="{lvl}" text:style-name="Heading_20_{lvl}">{t}</text:h>"#
                ));
            }
            BlockType::ListItem => {
                let style = list_style_name(block);
                let prefix = list_marker_prefix(block);
                body.push_str(&format!(
                    r#"<text:list text:style-name="{style}"><text:list-item><text:p>{prefix}{t}</text:p></text:list-item></text:list>"#
                ));
            }
            BlockType::Footnote => {
                body.push_str(&format!(
                    r#"<text:p text:style-name="Footnote"><text:span text:style-name="Emphasis">[{}]</text:span> {t}</text:p>"#,
                    xml_escape(block.bookmark_name.as_deref().unwrap_or("fn"))
                ));
            }
            BlockType::Image => {
                let alt = image_placeholder(block, &t);
                body.push_str(&format!(
                    r#"<text:p text:style-name="Standard">{alt}</text:p>"#
                ));
            }
            BlockType::Table => {
                body.push_str(&table_xml(block));
            }
            BlockType::TextBox => {
                body.push_str(&format!(
                    r#"<text:p text:style-name="Text_20_box">{t}</text:p>"#
                ));
            }
            BlockType::Bookmark | BlockType::Toc => {
                body.push_str(&format!(
                    r#"<text:p text:style-name="Standard">{t}</text:p>"#
                ));
            }
            _ => {
                let style = if block.style_name.as_deref() == Some("title") {
                    "Title"
                } else {
                    para_style(block)
                };
                body.push_str(&format!(r#"<text:p text:style-name="{style}">{t}</text:p>"#));
            }
        }
    }
    if !doc.footer.text.is_empty() {
        body.push_str(&format!(
            r#"<text:p text:style-name="Footer">{}</text:p>"#,
            xml_escape(&doc.footer.text)
        ));
    }
    format!(
        r#"<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
  xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0"
  xmlns:style="urn:oasis:names:tc:opendocument:xmlns:style:1.0"
  xmlns:table="urn:oasis:names:tc:opendocument:xmlns:table:1.0"
  xmlns:fo="urn:oasis:names:tc:opendocument:xmlns:xsl-fo-compatible:1.0"
  xmlns:xlink="http://www.w3.org/1999/xlink"
  office:version="1.2">
  <office:automatic-styles>
    <style:style style:name="Strong" style:family="text"><style:text-properties fo:font-weight="bold"/></style:style>
    <style:style style:name="Emphasis" style:family="text"><style:text-properties fo:font-style="italic"/></style:style>
    <style:style style:name="Underline" style:family="text"><style:text-properties style:text-underline-style="solid"/></style:style>
    <style:style style:name="Strike" style:family="text"><style:text-properties style:text-line-through-style="solid"/></style:style>
  </office:automatic-styles>
  <office:body>
    <office:text>
{body}
    </office:text>
  </office:body>
</office:document-content>"#
    )
}

/// Export a thicker-lite ODT package from ERA blocks (inline marks, tables, breaks).
pub fn export_odt(doc: &EradDocument) -> Result<Vec<u8>> {
    let content = build_content_xml(doc);
    let styles = r#"<?xml version="1.0" encoding="UTF-8"?>
<office:document-styles xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
  xmlns:style="urn:oasis:names:tc:opendocument:xmlns:style:1.0"
  xmlns:fo="urn:oasis:names:tc:opendocument:xmlns:xsl-fo-compatible:1.0"
  xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0"
  office:version="1.2">
  <office:styles>
    <style:style style:name="Standard" style:family="paragraph" style:class="text"/>
    <style:style style:name="Title" style:family="paragraph" style:parent-style-name="Standard">
      <style:text-properties fo:font-size="24pt" fo:font-weight="bold"/>
    </style:style>
    <style:style style:name="Heading_20_1" style:family="paragraph" style:parent-style-name="Standard">
      <style:text-properties fo:font-size="18pt" fo:font-weight="bold"/>
    </style:style>
    <style:style style:name="Heading_20_2" style:family="paragraph" style:parent-style-name="Standard">
      <style:text-properties fo:font-size="16pt" fo:font-weight="bold"/>
    </style:style>
    <style:style style:name="Heading_20_3" style:family="paragraph" style:parent-style-name="Standard">
      <style:text-properties fo:font-size="14pt" fo:font-weight="bold"/>
    </style:style>
    <style:style style:name="Center" style:family="paragraph" style:parent-style-name="Standard">
      <style:paragraph-properties fo:text-align="center"/>
    </style:style>
    <style:style style:name="Right" style:family="paragraph" style:parent-style-name="Standard">
      <style:paragraph-properties fo:text-align="end"/>
    </style:style>
    <style:style style:name="Justify" style:family="paragraph" style:parent-style-name="Standard">
      <style:paragraph-properties fo:text-align="justify"/>
    </style:style>
    <style:style style:name="Header" style:family="paragraph" style:parent-style-name="Standard"/>
    <style:style style:name="Footer" style:family="paragraph" style:parent-style-name="Standard"/>
    <style:style style:name="Footnote" style:family="paragraph" style:parent-style-name="Standard"/>
    <style:style style:name="Text_20_box" style:family="paragraph" style:parent-style-name="Standard"/>
    <style:style style:name="Horizontal_20_Line" style:family="paragraph" style:parent-style-name="Standard">
      <style:paragraph-properties fo:border-bottom="0.74pt solid #000000" fo:padding-bottom="0.04in"/>
    </style:style>
  </office:styles>
</office:document-styles>"#;
    let meta = r#"<?xml version="1.0" encoding="UTF-8"?>
<office:document-meta xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
  xmlns:meta="urn:oasis:names:tc:opendocument:xmlns:meta:1.0" office:version="1.2">
  <office:meta>
    <meta:generator>ERA Documents</meta:generator>
  </office:meta>
</office:document-meta>"#;
    let manifest = r#"<?xml version="1.0" encoding="UTF-8"?>
<manifest:manifest xmlns:manifest="urn:oasis:names:tc:opendocument:xmlns:manifest:1.0" manifest:version="1.2">
  <manifest:file-entry manifest:full-path="/" manifest:media-type="application/vnd.oasis.opendocument.text"/>
  <manifest:file-entry manifest:full-path="content.xml" manifest:media-type="text/xml"/>
  <manifest:file-entry manifest:full-path="styles.xml" manifest:media-type="text/xml"/>
  <manifest:file-entry manifest:full-path="meta.xml" manifest:media-type="text/xml"/>
</manifest:manifest>"#;

    let mut buf = Cursor::new(Vec::new());
    {
        let mut zip = ZipWriter::new(&mut buf);
        let stored = SimpleFileOptions::default().compression_method(CompressionMethod::Stored);
        zip.start_file("mimetype", stored)?;
        zip.write_all(ODT_MIME.as_bytes())?;
        let opts = SimpleFileOptions::default().compression_method(CompressionMethod::Deflated);
        zip.start_file("META-INF/manifest.xml", opts)?;
        zip.write_all(manifest.as_bytes())?;
        zip.start_file("content.xml", opts)?;
        zip.write_all(content.as_bytes())?;
        zip.start_file("styles.xml", opts)?;
        zip.write_all(styles.as_bytes())?;
        zip.start_file("meta.xml", opts)?;
        zip.write_all(meta.as_bytes())?;
        zip.finish()?;
    }
    Ok(buf.into_inner())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::model::{Block, InlineSpan, ListMarker, ListType, TextAlign};
    use std::io::Read;
    use zip::ZipArchive;

    fn content_xml(bytes: &[u8]) -> String {
        let mut archive = ZipArchive::new(Cursor::new(bytes)).unwrap();
        let mut content = String::new();
        archive
            .by_name("content.xml")
            .unwrap()
            .read_to_string(&mut content)
            .unwrap();
        content
    }

    fn styles_xml(bytes: &[u8]) -> String {
        let mut archive = ZipArchive::new(Cursor::new(bytes)).unwrap();
        let mut styles = String::new();
        archive
            .by_name("styles.xml")
            .unwrap()
            .read_to_string(&mut styles)
            .unwrap();
        styles
    }

    fn blk(
        id: &str,
        block_type: BlockType,
        inlines: Vec<InlineSpan>,
    ) -> Block {
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
    fn export_odt_mimetype() {
        let mut doc = EradDocument::empty();
        doc.blocks = vec![blk(
            "b1",
            BlockType::Paragraph,
            vec![InlineSpan::plain("Hello ODT")],
        )];
        let bytes = export_odt(&doc).expect("odt");
        assert!(bytes
            .windows(ODT_MIME.len())
            .any(|w| w == ODT_MIME.as_bytes()));
    }

    #[test]
    fn export_odt_preserves_bold_and_heading() {
        let mut bold = InlineSpan::plain("Bold");
        bold.bold = true;
        let mut doc = EradDocument::empty();
        let mut h = blk("h", BlockType::Heading, vec![InlineSpan::plain("Title")]);
        h.heading_level = 1;
        doc.blocks = vec![
            h,
            blk(
                "p",
                BlockType::Paragraph,
                vec![bold, InlineSpan::plain(" plain")],
            ),
        ];
        let content = content_xml(&export_odt(&doc).expect("odt"));
        assert!(content.contains("text:outline-level=\"1\""));
        assert!(content.contains("text:style-name=\"Strong\""));
        assert!(content.contains("Bold"));
    }

    /// Golden: fixed EradDocument → key substrings in ODT content/styles XML.
    #[test]
    fn export_odt_olite_p1_golden_substrings() {
        let mut doc = EradDocument::empty();
        doc.header.text = "HdrLite".into();
        doc.footer.text = "FtrLite".into();

        let mut table = blk("t1", BlockType::Table, vec![]);
        table.table_cells = Some(vec![
            vec!["A1".into(), "B1".into()],
            vec!["A2".into(), "B2".into()],
        ]);

        let mut img = blk("img", BlockType::Image, vec![]);
        img.image_url = Some("drive://obj/img1".into());

        let mut list = blk(
            "li",
            BlockType::ListItem,
            vec![InlineSpan::plain("Item")],
        );
        list.list_type = Some(ListType::Bullet);
        list.list_marker = Some(ListMarker::Square);

        let mut hr = blk("hr", BlockType::Paragraph, vec![]);
        hr.style_name = Some("horizontal_line".into());

        let tsv = blk(
            "tsv",
            BlockType::Table,
            vec![InlineSpan::plain("X\tY\nP\tQ")],
        );

        doc.blocks = vec![table, img, list, hr, tsv];

        let bytes = export_odt(&doc).expect("odt");
        let content = content_xml(&bytes);
        let styles = styles_xml(&bytes);

        assert!(content.contains(r#"text:style-name="Header""#));
        assert!(content.contains("HdrLite"));
        assert!(content.contains(r#"text:style-name="Footer""#));
        assert!(content.contains("FtrLite"));
        assert!(content.contains("<table:table"));
        assert!(content.contains("A1"));
        assert!(content.contains("B2"));
        assert!(content.contains("[image:drive://obj/img1]"));
        assert!(content.contains(r#"text:style-name="BulletSquare""#));
        assert!(content.contains("■ "));
        assert!(content.contains(r#"text:style-name="Horizontal_20_Line""#));
        assert!(content.contains("X"));
        assert!(content.contains("Q"));
        assert!(styles.contains("Horizontal_20_Line"));
        assert!(styles.contains("fo:border-bottom"));
    }
}
