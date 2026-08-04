use std::io::{Cursor, Write};

use anyhow::Result;
use base64::Engine;
use zip::write::SimpleFileOptions;
use zip::CompressionMethod;

use crate::model::{ErapDeck, ErapSlide};

fn xml(s: &str) -> String {
    s.replace('&', "&amp;")
        .replace('<', "&lt;")
        .replace('>', "&gt;")
        .replace('"', "&quot;")
}

/// Parse `#rrggbb` / `#rgb`, or bake the first hex color found (gradients).
fn parse_solid_hex(s: &str) -> Option<String> {
    let trimmed = s.trim();
    if let Some(hex) = exact_hex(trimmed) {
        return Some(hex);
    }
    let bytes = trimmed.as_bytes();
    let mut i = 0;
    while i < bytes.len() {
        if bytes[i] == b'#' {
            let after = &trimmed[i + 1..];
            let hex_len = after
                .chars()
                .take_while(|c| c.is_ascii_hexdigit())
                .count();
            if hex_len >= 6 {
                if let Some(hex) = exact_hex(&format!("#{}", &after[..6])) {
                    return Some(hex);
                }
            } else if hex_len >= 3 {
                if let Some(hex) = exact_hex(&format!("#{}", &after[..3])) {
                    return Some(hex);
                }
            }
        }
        i += 1;
    }
    None
}

fn exact_hex(s: &str) -> Option<String> {
    let s = s.trim();
    if !s.starts_with('#') {
        return None;
    }
    let h = &s[1..];
    if h.len() == 6 && h.chars().all(|c| c.is_ascii_hexdigit()) {
        return Some(format!("#{}", h.to_ascii_lowercase()));
    }
    if h.len() == 3 && h.chars().all(|c| c.is_ascii_hexdigit()) {
        let mut out = String::with_capacity(6);
        for c in h.chars() {
            out.push(c);
            out.push(c);
        }
        return Some(format!("#{}", out.to_ascii_lowercase()));
    }
    None
}

fn frame_has_bold(frame: &era_office_richtext::TextFrame) -> bool {
    frame
        .blocks
        .iter()
        .any(|b| b.inlines.iter().any(|sp| sp.bold))
}

fn frame_paras(frame: &era_office_richtext::TextFrame) -> String {
    if frame.blocks.is_empty() {
        return "<text:p></text:p>".into();
    }
    frame
        .blocks
        .iter()
        .map(|b| {
            if b.inlines.is_empty() {
                return "<text:p></text:p>".into();
            }
            let inner: String = b
                .inlines
                .iter()
                .map(|sp| {
                    let t = xml(&sp.text);
                    if sp.bold {
                        format!(r#"<text:span text:style-name="Tbold">{t}</text:span>"#)
                    } else {
                        t
                    }
                })
                .collect();
            format!("<text:p>{inner}</text:p>")
        })
        .collect()
}

fn paras_plain(text: &str) -> String {
    let lines: Vec<&str> = text.lines().collect();
    if lines.is_empty() || (lines.len() == 1 && lines[0].is_empty()) {
        return "<text:p></text:p>".into();
    }
    lines
        .iter()
        .map(|l| format!("<text:p>{}</text:p>", xml(l)))
        .collect::<Vec<_>>()
        .join("")
}

struct DataImage {
    ext: &'static str,
    mime: &'static str,
    bytes: Vec<u8>,
}

fn parse_data_image(url: &str) -> Option<DataImage> {
    let url = url.trim();
    if !url.starts_with("data:image/") {
        return None;
    }
    let rest = &url["data:image/".len()..];
    let (mime_sub, payload) = rest.split_once(';')?;
    if !payload.to_ascii_lowercase().contains("base64") {
        return None;
    }
    let (_marker, b64) = payload.split_once(',')?;
    let (ext, mime) = match mime_sub.to_ascii_lowercase().as_str() {
        "png" => ("png", "image/png"),
        "jpeg" | "jpg" => ("jpg", "image/jpeg"),
        _ => return None,
    };
    let bytes = base64::engine::general_purpose::STANDARD
        .decode(b64.trim())
        .ok()?;
    Some(DataImage { ext, mime, bytes })
}

struct PageBuild {
    xml: String,
    style_name: String,
    bg_hex: Option<String>,
    image: Option<(String, DataImage)>,
}

fn page_xml(
    slide: &ErapSlide,
    index: usize,
    theme_bg: Option<&str>,
    image_index: &mut u32,
) -> PageBuild {
    let n = index + 1;
    let title = frame_paras(&slide.title_frame);
    let body = match slide.layout.as_str() {
        "title_only" => String::new(),
        "two_column" => {
            let left = frame_paras(&slide.body_frame);
            let right = frame_paras(&slide.body2_frame);
            format!(
                r#"        <draw:frame svg:width="12cm" svg:height="12cm" svg:x="1.5cm" svg:y="5cm">
          <draw:text-box>{left}</draw:text-box>
        </draw:frame>
        <draw:frame svg:width="12cm" svg:height="12cm" svg:x="14.5cm" svg:y="5cm">
          <draw:text-box>{right}</draw:text-box>
        </draw:frame>
"#
            )
        }
        _ => {
            let b = frame_paras(&slide.body_frame);
            format!(
                r#"        <draw:frame svg:width="25cm" svg:height="12cm" svg:x="1.5cm" svg:y="5cm">
          <draw:text-box>{b}</draw:text-box>
        </draw:frame>
"#
            )
        }
    };
    let notes = if slide.notes.trim().is_empty() {
        String::new()
    } else {
        format!(
            r#"        <presentation:notes>
          <draw:page-thumbnail/>
          <draw:frame svg:width="17cm" svg:height="12cm" svg:x="2cm" svg:y="14cm">
            <draw:text-box>{}</draw:text-box>
          </draw:frame>
        </presentation:notes>
"#,
            paras_plain(&slide.notes)
        )
    };

    let bg_hex = slide
        .background
        .as_deref()
        .or(theme_bg)
        .and_then(parse_solid_hex);
    let style_name = if bg_hex.is_some() {
        format!("dpBg{n}")
    } else {
        "dp1".into()
    };

    let mut image_xml = String::new();
    let mut image_part = None;
    if let Some(url) = slide.image_url.as_deref() {
        if let Some(img) = parse_data_image(url) {
            *image_index += 1;
            let href = format!("Pictures/image{}.{}", *image_index, img.ext);
            image_xml = format!(
                r#"        <draw:frame svg:width="10cm" svg:height="6cm" svg:x="10cm" svg:y="10cm">
          <draw:image xlink:href="{href}" xlink:type="simple" xlink:show="embed" xlink:actuate="onLoad"/>
        </draw:frame>
"#
            );
            image_part = Some((href, img));
        }
    }

    let xml = format!(
        r#"      <draw:page draw:name="page{n}" draw:style-name="{style_name}" draw:master-page-name="Default">
        <draw:frame svg:width="25cm" svg:height="3cm" svg:x="1.5cm" svg:y="1cm">
          <draw:text-box>{title}</draw:text-box>
        </draw:frame>
{body}{image_xml}{notes}      </draw:page>
"#
    );
    PageBuild {
        xml,
        style_name,
        bg_hex,
        image: image_part,
    }
}

/// Thicker-lite ODF presentation (ODP): multi-line paras, notes, two-column, bg, images, bold.
pub fn export_odp(deck: &ErapDeck) -> Result<Vec<u8>> {
    let slides: Vec<ErapSlide> = if deck.slides.is_empty() {
        vec![ErapSlide::new_blank()]
    } else {
        deck.slides.clone()
    };
    let theme_bg = deck.theme_background.as_deref();

    let mut pages = String::new();
    let mut auto_styles = String::new();
    let mut need_tbold = false;
    let mut image_index = 0u32;
    let mut pictures: Vec<(String, &'static str, Vec<u8>)> = Vec::new();

    for (i, slide) in slides.iter().enumerate() {
        if frame_has_bold(&slide.title_frame)
            || frame_has_bold(&slide.body_frame)
            || frame_has_bold(&slide.body2_frame)
        {
            need_tbold = true;
        }
        let built = page_xml(slide, i, theme_bg, &mut image_index);
        if let Some(hex) = &built.bg_hex {
            auto_styles.push_str(&format!(
                r#"    <style:style style:name="{}" style:family="drawing-page">
      <style:drawing-page-properties fo:background-color="{hex}" presentation:background-visible="true"/>
    </style:style>
"#,
                built.style_name
            ));
        }
        if let Some((href, img)) = built.image {
            pictures.push((href, img.mime, img.bytes));
        }
        pages.push_str(&built.xml);
    }

    if need_tbold {
        auto_styles.push_str(
            r#"    <style:style style:name="Tbold" style:family="text">
      <style:text-properties fo:font-weight="bold"/>
    </style:style>
"#,
        );
    }

    let content = format!(
        r#"<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" xmlns:style="urn:oasis:names:tc:opendocument:xmlns:style:1.0" xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0" xmlns:draw="urn:oasis:names:tc:opendocument:xmlns:drawing:1.0" xmlns:presentation="urn:oasis:names:tc:opendocument:xmlns:presentation:1.0" xmlns:svg="urn:oasis:names:tc:opendocument:xmlns:svg-compatible:1.0" xmlns:fo="urn:oasis:names:tc:opendocument:xmlns:xsl-fo-compatible:1.0" xmlns:xlink="http://www.w3.org/1999/xlink" office:version="1.2">
  <office:automatic-styles>
{auto_styles}  </office:automatic-styles>
  <office:body>
    <office:presentation>
{pages}    </office:presentation>
  </office:body>
</office:document-content>"#
    );

    let styles = r#"<?xml version="1.0" encoding="UTF-8"?>
<office:document-styles xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" xmlns:style="urn:oasis:names:tc:opendocument:xmlns:style:1.0" xmlns:draw="urn:oasis:names:tc:opendocument:xmlns:drawing:1.0" xmlns:fo="urn:oasis:names:tc:opendocument:xmlns:xsl-fo-compatible:1.0" xmlns:presentation="urn:oasis:names:tc:opendocument:xmlns:presentation:1.0" office:version="1.2">
  <office:styles>
    <style:style style:name="dp1" style:family="drawing-page">
      <style:drawing-page-properties presentation:background-visible="true" presentation:background-objects-visible="true"/>
    </style:style>
    <style:style style:name="dpCustom" style:family="drawing-page">
      <style:drawing-page-properties presentation:background-visible="true"/>
    </style:style>
  </office:styles>
  <office:automatic-styles/>
  <office:master-styles>
    <style:master-page style:name="Default" style:page-layout-name="PM1">
      <style:drawing-page-properties presentation:display-footer="true"/>
    </style:master-page>
  </office:master-styles>
</office:document-styles>"#;

    let meta = format!(
        r#"<?xml version="1.0" encoding="UTF-8"?>
<office:document-meta xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" xmlns:meta="urn:oasis:names:tc:opendocument:xmlns:meta:1.0" office:version="1.2">
  <office:meta>
    <meta:generator>ERA Presentations</meta:generator>
    <dc:title xmlns:dc="http://purl.org/dc/elements/1.1/">{}</dc:title>
  </office:meta>
</office:document-meta>"#,
        xml(&deck.name)
    );

    let mut manifest_entries = String::from(
        r#"  <manifest:file-entry manifest:full-path="/" manifest:version="1.2" manifest:media-type="application/vnd.oasis.opendocument.presentation"/>
  <manifest:file-entry manifest:full-path="content.xml" manifest:media-type="text/xml"/>
  <manifest:file-entry manifest:full-path="styles.xml" manifest:media-type="text/xml"/>
  <manifest:file-entry manifest:full-path="meta.xml" manifest:media-type="text/xml"/>
"#,
    );
    if !pictures.is_empty() {
        manifest_entries.push_str(
            r#"  <manifest:file-entry manifest:full-path="Pictures/" manifest:media-type=""/>
"#,
        );
        for (href, mime, _) in &pictures {
            manifest_entries.push_str(&format!(
                r#"  <manifest:file-entry manifest:full-path="{href}" manifest:media-type="{mime}"/>
"#
            ));
        }
    }

    let manifest = format!(
        r#"<?xml version="1.0" encoding="UTF-8"?>
<manifest:manifest xmlns:manifest="urn:oasis:names:tc:opendocument:xmlns:manifest:1.0" manifest:version="1.2">
{manifest_entries}</manifest:manifest>"#
    );

    let mimetype = "application/vnd.oasis.opendocument.presentation";

    let mut buf = Cursor::new(Vec::new());
    {
        let mut zip = zip::ZipWriter::new(&mut buf);
        let store = SimpleFileOptions::default().compression_method(CompressionMethod::Stored);
        let deflate = SimpleFileOptions::default();
        zip.start_file("mimetype", store)?;
        zip.write_all(mimetype.as_bytes())?;
        zip.start_file("META-INF/manifest.xml", deflate)?;
        zip.write_all(manifest.as_bytes())?;
        zip.start_file("content.xml", deflate)?;
        zip.write_all(content.as_bytes())?;
        zip.start_file("styles.xml", deflate)?;
        zip.write_all(styles.as_bytes())?;
        zip.start_file("meta.xml", deflate)?;
        zip.write_all(meta.as_bytes())?;
        for (href, _mime, bytes) in &pictures {
            zip.start_file(href, deflate)?;
            zip.write_all(bytes)?;
        }
        zip.finish()?;
    }
    Ok(buf.into_inner())
}

#[cfg(test)]
mod tests {
    use super::*;
    use era_office_richtext::{Block, InlineSpan, TextFrame};
    use std::io::Read;
    use zip::ZipArchive;

    #[test]
    fn export_odp_contains_mimetype_and_opendocument_presentation() {
        let mut deck = ErapDeck::empty();
        deck.slides[0].set_title_plain("Hello");
        deck.slides[0].set_body_plain("World");
        let bytes = export_odp(&deck).unwrap();
        let s = String::from_utf8_lossy(&bytes);
        assert!(s.contains("mimetype"), "zip must contain mimetype entry");
        assert!(
            s.contains("opendocument.presentation"),
            "zip must declare opendocument.presentation"
        );
        assert!(bytes.windows(2).any(|w| w == b"PK"), "must be a zip");
    }

    #[test]
    fn export_odp_includes_notes_and_multiline_body() {
        let mut deck = ErapDeck::empty();
        deck.slides[0].set_title_plain("T");
        deck.slides[0].set_body_plain("Line1\nLine2");
        deck.slides[0].notes = "Speak slowly".into();
        let bytes = export_odp(&deck).unwrap();
        let mut archive = ZipArchive::new(Cursor::new(&bytes)).unwrap();
        let mut content = String::new();
        archive
            .by_name("content.xml")
            .unwrap()
            .read_to_string(&mut content)
            .unwrap();
        assert!(content.contains("presentation:notes"));
        assert!(content.contains("Speak slowly"));
        assert!(content.contains("Line1"));
        assert!(content.contains("Line2"));
    }

    #[test]
    fn export_odp_image_solid_bg_bold() {
        let mut deck = ErapDeck::empty();
        deck.theme_background = Some("#112233".into());
        deck.slides[0].set_title_plain("Title");
        let mut bold_span = InlineSpan::plain("BoldBit");
        bold_span.bold = true;
        let mut body_block = Block::paragraph("b1", "");
        body_block.inlines = vec![InlineSpan::plain("Hi "), bold_span];
        deck.slides[0].body_frame = TextFrame {
            blocks: vec![body_block],
        };
        deck.slides[0].image_url = Some(
            "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
                .into(),
        );

        let bytes = export_odp(&deck).unwrap();
        let mut archive = ZipArchive::new(Cursor::new(&bytes)).unwrap();
        let names: Vec<String> = (0..archive.len())
            .map(|i| archive.by_index(i).unwrap().name().to_string())
            .collect();
        assert!(
            names.iter().any(|n| n.starts_with("Pictures/")),
            "zip must contain Pictures: {names:?}"
        );

        let mut content = String::new();
        archive
            .by_name("content.xml")
            .unwrap()
            .read_to_string(&mut content)
            .unwrap();
        assert!(
            content.contains("draw:image") || content.contains("Pictures/"),
            "content must reference image"
        );
        assert!(
            content.contains("fo:background-color=\"#112233\"")
                || content.contains("fo:background-color"),
            "content must set solid bg"
        );
        assert!(
            content.contains("text:style-name=\"Tbold\"") && content.contains("Tbold"),
            "content must define/use bold style"
        );
    }
}
