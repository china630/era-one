use std::io::{Cursor, Read};

use anyhow::{Context, Result};
use quick_xml::events::Event;
use quick_xml::Reader;
use zip::ZipArchive;

use crate::model::{Block, BlockType, EradDocument, InlineSpan, ListType, TextAlign};

const W_NS: &str = "http://schemas.openxmlformats.org/wordprocessingml/2006/main";

/// Import a minimal docx (paragraphs, headings, bold/italic, lists).
pub fn import_docx(data: &[u8]) -> Result<EradDocument> {
    let cursor = Cursor::new(data);
    let mut archive = ZipArchive::new(cursor).context("open docx zip")?;
    let mut xml = String::new();
    archive
        .by_name("word/document.xml")
        .context("word/document.xml missing")?
        .read_to_string(&mut xml)?;
    parse_document_xml(&xml)
}

fn local_name(name: &[u8]) -> Vec<u8> {
    name.rsplit(|b| *b == b':')
        .next()
        .unwrap_or(name)
        .to_vec()
}

fn parse_document_xml(xml: &str) -> Result<EradDocument> {
    let mut reader = Reader::from_str(xml);
    reader.config_mut().trim_text(false);
    let mut doc = EradDocument::empty();
    doc.blocks.clear();
    doc.legacy_features_dropped = false;

    let mut buf = Vec::new();
    let mut in_body = false;
    let mut in_p = false;
    let mut in_r = false;
    let mut in_t = false;
    let mut bold = false;
    let mut italic = false;
    let mut underline = false;
    let mut strike = false;
    let mut current_block: Option<Block> = None;
    let mut p_style: Option<String> = None;
    let mut list_type: Option<ListType> = None;
    let mut list_level: u32 = 0;

    loop {
        match reader.read_event_into(&mut buf) {
            Ok(Event::Start(e)) | Ok(Event::Empty(e)) => {
                let name = local_name(e.name().as_ref());
                if name == b"body" {
                    in_body = true;
                } else if in_body && name == b"p" {
                    in_p = true;
                    p_style = None;
                    list_type = None;
                    list_level = 0;
                    current_block = Some(Block {
                        id: uuid::Uuid::new_v4().to_string(),
                        block_type: BlockType::Paragraph,
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
                        inlines: Vec::new(),
                    });
                } else if in_p && name == b"pStyle" {
                    for attr in e.attributes().flatten() {
                        if local_name(attr.key.as_ref()) == b"val" {
                            p_style = Some(String::from_utf8_lossy(&attr.value).into_owned());
                        }
                    }
                } else if in_p && name == b"numPr" {
                    list_type = Some(ListType::Bullet);
                } else if in_p && name == b"ilvl" {
                    for attr in e.attributes().flatten() {
                        if local_name(attr.key.as_ref()) == b"val" {
                            list_level = String::from_utf8_lossy(&attr.value)
                                .parse()
                                .unwrap_or(0);
                        }
                    }
                } else if in_p && name == b"numId" {
                    for attr in e.attributes().flatten() {
                        if local_name(attr.key.as_ref()) == b"val" && attr.value.as_ref() == b"1" {
                            list_type = Some(ListType::Ordered);
                        }
                    }
                } else if in_p && name == b"r" {
                    in_r = true;
                    bold = false;
                    italic = false;
                    underline = false;
                    strike = false;
                } else if in_r && name == b"b" {
                    bold = true;
                } else if in_r && name == b"i" {
                    italic = true;
                } else if in_r && name == b"u" {
                    underline = true;
                } else if in_r && name == b"strike" {
                    strike = true;
                } else if in_r && name == b"br" {
                    for attr in e.attributes().flatten() {
                        if local_name(attr.key.as_ref()) == b"type"
                            && attr.value.as_ref() == b"page"
                        {
                            if let Some(mut block) = current_block.take() {
                                block.block_type = BlockType::PageBreak;
                                block.inlines.clear();
                                doc.blocks.push(block);
                            }
                            current_block = Some(Block {
                                id: uuid::Uuid::new_v4().to_string(),
                                block_type: BlockType::Paragraph,
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
                                inlines: Vec::new(),
                            });
                        }
                    }
                } else if in_r && name == b"t" {
                    in_t = true;
                } else if name == b"tbl" || name == b"drawing" {
                    doc.legacy_features_dropped = true;
                }
            }
            Ok(Event::Text(t)) if in_t => {
                if let Some(block) = current_block.as_mut() {
                    let text = t.unescape().unwrap_or_default().into_owned();
                    block.inlines.push(InlineSpan {
                        text,
                        bold,
                        italic,
                        underline,
                        strike,
                        link_url: None,
                        font_family: None,
                        font_size_pt: None,
                        color: None,
                        highlight: None,
                        superscript: false,
                        subscript: false,
                    });
                }
            }
            Ok(Event::End(e)) => {
                let name = local_name(e.name().as_ref());
                if name == b"t" {
                    in_t = false;
                } else if name == b"r" {
                    in_r = false;
                } else if name == b"p" && in_p {
                    if let Some(mut block) = current_block.take() {
                        if let Some(style) = p_style.as_deref() {
                            if style.eq_ignore_ascii_case("Title") {
                                block.block_type = BlockType::Heading;
                                block.heading_level = 1;
                                block.style_name = Some("title".into());
                            } else if style.starts_with("Heading") {
                                block.block_type = BlockType::Heading;
                                block.heading_level = style
                                    .chars()
                                    .filter(|c| c.is_ascii_digit())
                                    .collect::<String>()
                                    .parse()
                                    .unwrap_or(1);
                            }
                        }
                        block.list_type = list_type;
                        if list_type.is_some() {
                            block.block_type = BlockType::ListItem;
                            block.list_level = list_level;
                            block.list_marker = Some(match list_type {
                                Some(ListType::Ordered) => crate::model::ListMarker::Decimal,
                                _ => crate::model::ListMarker::Disc,
                            });
                        }
                        if block.inlines.is_empty() {
                            block.inlines.push(InlineSpan::plain(""));
                        }
                        doc.blocks.push(block);
                    }
                    in_p = false;
                }
            }
            Ok(Event::Eof) => break,
            Err(e) => return Err(e.into()),
            _ => {}
        }
        buf.clear();
    }

    if doc.blocks.is_empty() {
        doc.blocks.push(Block {
            id: uuid::Uuid::new_v4().to_string(),
            block_type: BlockType::Paragraph,
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
            inlines: vec![InlineSpan::plain("")],
        });
    }
    Ok(doc)
}

/// Build minimal docx bytes for tests.
pub fn minimal_docx(paragraphs: &[&str]) -> Result<Vec<u8>> {
    let body: String = paragraphs
        .iter()
        .map(|p| {
            format!(
                r#"<w:p xmlns:w="{W_NS}"><w:r><w:t>{p}</w:t></w:r></w:p>"#
            )
        })
        .collect();
    let document_xml = format!(
        r#"<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="{W_NS}"><w:body>{body}</w:body></w:document>"#
    );
    let content_types = r#"<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>"#;
    let rels = r#"<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>"#;

    let mut buf = Cursor::new(Vec::new());
    {
        let mut zip = zip::ZipWriter::new(&mut buf);
        let opts = zip::write::SimpleFileOptions::default();
        zip.start_file("[Content_Types].xml", opts)?;
        std::io::Write::write_all(&mut zip, content_types.as_bytes())?;
        zip.start_file("_rels/.rels", opts)?;
        std::io::Write::write_all(&mut zip, rels.as_bytes())?;
        zip.start_file("word/document.xml", opts)?;
        std::io::Write::write_all(&mut zip, document_xml.as_bytes())?;
        zip.finish()?;
    }
    Ok(buf.into_inner())
}

/// Build docx with headings, mixed inline styles, and nested lists (test fixtures).
pub fn fixture_docx(body_xml: &str) -> Result<Vec<u8>> {
    let document_xml = format!(
        r#"<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="{W_NS}"><w:body>{body_xml}</w:body></w:document>"#
    );
    let content_types = r#"<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>"#;
    let rels = r#"<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>"#;

    let mut buf = Cursor::new(Vec::new());
    {
        let mut zip = zip::ZipWriter::new(&mut buf);
        let opts = zip::write::SimpleFileOptions::default();
        zip.start_file("[Content_Types].xml", opts)?;
        std::io::Write::write_all(&mut zip, content_types.as_bytes())?;
        zip.start_file("_rels/.rels", opts)?;
        std::io::Write::write_all(&mut zip, rels.as_bytes())?;
        zip.start_file("word/document.xml", opts)?;
        std::io::Write::write_all(&mut zip, document_xml.as_bytes())?;
        zip.finish()?;
    }
    Ok(buf.into_inner())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn import_minimal_docx() {
        let docx = minimal_docx(&["Hello memo"]).unwrap();
        let doc = import_docx(&docx).unwrap();
        assert!(doc.plain_text().contains("Hello memo"));
    }

    #[test]
    fn import_mixed_bold_italic_spans() {
        let body = format!(
            r#"<w:p xmlns:w="{W_NS}"><w:r><w:rPr><w:b/></w:rPr><w:t>Bold</w:t></w:r>
<w:r><w:t> and </w:t></w:r>
<w:r><w:rPr><w:i/></w:rPr><w:t>italic</w:t></w:r></w:p>"#
        );
        let docx = fixture_docx(&body).unwrap();
        let doc = import_docx(&docx).unwrap();
        assert_eq!(doc.plain_text(), "Bold and italic");
        assert!(doc.blocks[0].inlines.iter().any(|s| s.bold && s.text == "Bold"));
        assert!(doc.blocks[0].inlines.iter().any(|s| s.italic && s.text == "italic"));
    }

    #[test]
    fn fuzz_docx_smoke_random_bytes() {
        for i in 0..64 {
            let data: Vec<u8> = (0..i).map(|b| (b * 17) as u8).collect();
            let _ = import_docx(&data);
        }
    }
}
