use std::io::{Cursor, Read, Write};

use anyhow::{bail, Result};
use base64::Engine;
use era_office_richtext::TextAlign;
use zip::ZipArchive;

use crate::model::{ErapDeck, ErapSlide};

fn frame_paras_xml(frame: &era_office_richtext::TextFrame) -> String {
    frame
        .blocks
        .iter()
        .map(|b| {
            let p_pr = match b.align {
                TextAlign::Center => r#"<a:pPr algn="ctr"/>"#,
                TextAlign::Right => r#"<a:pPr algn="r"/>"#,
                _ => "",
            };
            let runs: String = b
                .inlines
                .iter()
                .map(|sp| {
                    let mut rpr = String::new();
                    if sp.bold {
                        rpr.push_str(r#"<a:rPr b="1"/>"#);
                    }
                    format!("<a:r>{rpr}<a:t>{}</a:t></a:r>", xml(&sp.text))
                })
                .collect();
            format!("<a:p>{p_pr}{runs}</a:p>")
        })
        .collect()
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
        return Some(h.to_ascii_uppercase());
    }
    if h.len() == 3 && h.chars().all(|c| c.is_ascii_hexdigit()) {
        let mut out = String::with_capacity(6);
        for c in h.chars() {
            out.push(c);
            out.push(c);
        }
        return Some(out.to_ascii_uppercase());
    }
    None
}

fn bg_xml(slide: &ErapSlide, theme_bg: Option<&str>) -> String {
    let src = slide
        .background
        .as_deref()
        .or(theme_bg)
        .and_then(parse_solid_hex);
    match src {
        Some(hex) => format!(
            r#"<p:bg><p:bgPr><a:solidFill><a:srgbClr val="{hex}"/></a:solidFill></p:bgPr></p:bg>"#
        ),
        None => String::new(),
    }
}

struct DataImage {
    ext: &'static str,
    bytes: Vec<u8>,
}

fn parse_data_image(url: &str) -> Option<DataImage> {
    let url = url.trim();
    if !url.starts_with("data:image/") {
        return None;
    }
    let rest = &url["data:image/".len()..];
    let (mime_sub, payload) = rest.split_once(';')?;
    let (_marker, b64) = payload.split_once(',')?;
    if !payload.to_ascii_lowercase().contains("base64") {
        return None;
    }
    let ext = match mime_sub.to_ascii_lowercase().as_str() {
        "png" => "png",
        "jpeg" | "jpg" => "jpg",
        _ => return None,
    };
    let bytes = base64::engine::general_purpose::STANDARD
        .decode(b64.trim())
        .ok()?;
    Some(DataImage { ext, bytes })
}

fn notes_slide_xml(notes: &str) -> String {
    format!(
        r#"<?xml version="1.0" encoding="UTF-8"?>
<p:notes xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:cSld><p:spTree>
    <p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr>
    <p:grpSpPr/>
    <p:sp><p:nvSpPr><p:cNvPr id="2" name="Notes"/><p:cNvSpPr/><p:nvPr><p:ph type="body"/></p:nvPr></p:nvSpPr>
      <p:txBody><a:p><a:r><a:t>{}</a:t></a:r></a:p></p:txBody>
    </p:sp>
  </p:spTree></p:cSld>
</p:notes>"#,
        xml(notes)
    )
}

fn slide_xml(
    slide: &ErapSlide,
    theme_bg: Option<&str>,
    image_rid: Option<&str>,
) -> String {
    let title_xml = frame_paras_xml(&slide.title_frame);
    let body_xml = match slide.layout.as_str() {
        "title_only" => String::new(),
        _ => frame_paras_xml(&slide.body_frame),
    };
    let body2_shape = if slide.layout == "two_column" {
        let body2_xml = frame_paras_xml(&slide.body2_frame);
        format!("    <p:sp><p:txBody>{body2_xml}</p:txBody></p:sp>\n")
    } else {
        String::new()
    };
    let bg = bg_xml(slide, theme_bg);
    let pic = if let Some(rid) = image_rid {
        format!(
            r#"    <p:pic>
      <p:nvPicPr><p:cNvPr id="10" name="Picture"/><p:cNvPicPr/><p:nvPr/></p:nvPicPr>
      <p:blipFill><a:blip r:embed="{rid}"/><a:stretch><a:fillRect/></a:stretch></p:blipFill>
      <p:spPr><a:xfrm><a:off x="4572000" y="3200400"/><a:ext cx="4572000" cy="2743200"/></a:xfrm></p:spPr>
    </p:pic>
"#
        )
    } else {
        String::new()
    };
    format!(
        r#"<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:cSld>{bg}<p:spTree>
    <p:sp><p:txBody>{title_xml}</p:txBody></p:sp>
    <p:sp><p:txBody>{body_xml}</p:txBody></p:sp>
{body2_shape}{pic}  </p:spTree></p:cSld>
</p:sld>"#
    )
}

/// Minimal multi-slide pptx with presentation.xml + slideN parts.
pub fn export_pptx(deck: &ErapDeck) -> Result<Vec<u8>> {
    let slides: Vec<ErapSlide> = if deck.slides.is_empty() {
        vec![ErapSlide::new_blank()]
    } else {
        deck.slides.clone()
    };
    let theme_bg = deck.theme_background.as_deref();

    let mut overrides = String::new();
    let mut rels = String::new();
    let mut sld_id_lst = String::new();
    let mut need_png = false;
    let mut need_jpg = false;
    let mut media_parts: Vec<(String, Vec<u8>)> = Vec::new();
    let mut notes_parts: Vec<(usize, String)> = Vec::new();
    let mut slide_rels_parts: Vec<(usize, String)> = Vec::new();
    let mut slide_xmls: Vec<String> = Vec::new();
    let mut image_counter = 0u32;

    for (i, slide) in slides.iter().enumerate() {
        let n = i + 1;
        overrides.push_str(&format!(
            r#"  <Override PartName="/ppt/slides/slide{n}.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>
"#
        ));
        rels.push_str(&format!(
            r#"  <Relationship Id="rId{n}" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide{n}.xml"/>
"#
        ));
        sld_id_lst.push_str(&format!(r#"<p:sldId id="{}" r:id="rId{n}"/>"#, 256 + n));

        let mut slide_rel_entries = String::new();
        let mut image_rid: Option<String> = None;

        if let Some(url) = slide.image_url.as_deref() {
            if let Some(img) = parse_data_image(url) {
                image_counter += 1;
                let media_name = format!("image{image_counter}.{}", img.ext);
                let rid = "rIdImg1".to_string();
                slide_rel_entries.push_str(&format!(
                    r#"  <Relationship Id="{rid}" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="../media/{media_name}"/>
"#
                ));
                match img.ext {
                    "png" => need_png = true,
                    "jpg" => need_jpg = true,
                    _ => {}
                }
                media_parts.push((format!("ppt/media/{media_name}"), img.bytes));
                image_rid = Some(rid);
            }
        }

        if !slide.notes.trim().is_empty() {
            overrides.push_str(&format!(
                r#"  <Override PartName="/ppt/notesSlides/notesSlide{n}.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.notesSlide+xml"/>
"#
            ));
            slide_rel_entries.push_str(&format!(
                r#"  <Relationship Id="rIdNotes" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/notesSlide" Target="../notesSlides/notesSlide{n}.xml"/>
"#
            ));
            notes_parts.push((n, notes_slide_xml(&slide.notes)));
        }

        if !slide_rel_entries.is_empty() {
            slide_rels_parts.push((
                n,
                format!(
                    r#"<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
{slide_rel_entries}</Relationships>"#
                ),
            ));
        }

        slide_xmls.push(slide_xml(
            slide,
            theme_bg,
            image_rid.as_deref(),
        ));
    }

    let mut defaults = String::from(
        r#"  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
"#,
    );
    if need_png {
        defaults.push_str(
            r#"  <Default Extension="png" ContentType="image/png"/>
"#,
        );
    }
    if need_jpg {
        defaults.push_str(
            r#"  <Default Extension="jpg" ContentType="image/jpeg"/>
  <Default Extension="jpeg" ContentType="image/jpeg"/>
"#,
        );
    }

    let content_types = format!(
        r#"<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
{defaults}  <Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>
{overrides}</Types>"#
    );

    let presentation = format!(
        r#"<?xml version="1.0" encoding="UTF-8"?>
<p:presentation xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:sldIdLst>{sld_id_lst}</p:sldIdLst>
</p:presentation>"#
    );

    let presentation_rels = format!(
        r#"<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
{rels}</Relationships>"#
    );

    let root_rels = r#"<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/>
</Relationships>"#;

    let mut buf = Cursor::new(Vec::new());
    {
        let mut zip = zip::ZipWriter::new(&mut buf);
        let opts = zip::write::SimpleFileOptions::default();
        zip.start_file("[Content_Types].xml", opts)?;
        zip.write_all(content_types.as_bytes())?;
        zip.start_file("_rels/.rels", opts)?;
        zip.write_all(root_rels.as_bytes())?;
        zip.start_file("ppt/presentation.xml", opts)?;
        zip.write_all(presentation.as_bytes())?;
        zip.start_file("ppt/_rels/presentation.xml.rels", opts)?;
        zip.write_all(presentation_rels.as_bytes())?;
        for (i, xml_body) in slide_xmls.iter().enumerate() {
            let name = format!("ppt/slides/slide{}.xml", i + 1);
            zip.start_file(&name, opts)?;
            zip.write_all(xml_body.as_bytes())?;
        }
        for (n, rel_xml) in &slide_rels_parts {
            let name = format!("ppt/slides/_rels/slide{n}.xml.rels");
            zip.start_file(&name, opts)?;
            zip.write_all(rel_xml.as_bytes())?;
        }
        for (n, notes_xml) in &notes_parts {
            let name = format!("ppt/notesSlides/notesSlide{n}.xml");
            zip.start_file(&name, opts)?;
            zip.write_all(notes_xml.as_bytes())?;
        }
        for (path, bytes) in &media_parts {
            zip.start_file(path, opts)?;
            zip.write_all(bytes)?;
        }
        zip.finish()?;
    }
    Ok(buf.into_inner())
}

pub fn import_pptx(data: &[u8]) -> Result<ErapDeck> {
    let mut deck = ErapDeck::empty();
    if data.len() < 4 || &data[0..2] != b"PK" {
        return Ok(deck);
    }
    let mut archive = ZipArchive::new(Cursor::new(data))?;
    let mut names: Vec<String> = Vec::new();
    for i in 0..archive.len() {
        let f = archive.by_index(i)?;
        let name = f.name().to_string();
        if name.starts_with("ppt/slides/slide") && name.ends_with(".xml") {
            names.push(name);
        }
    }
    names.sort_by(|a, b| {
        let na = slide_num(a);
        let nb = slide_num(b);
        na.cmp(&nb)
    });
    if names.is_empty() {
        bail!("no slides in pptx");
    }
    let mut slides = Vec::new();
    for name in names {
        let mut slide_xml = String::new();
        {
            let mut file = archive.by_name(&name)?;
            file.read_to_string(&mut slide_xml)?;
        }
        let texts = extract_a_t(&slide_xml);
        let title = texts.first().cloned().unwrap_or_default();
        let body = texts.get(1).cloned().unwrap_or_default();
        let mut slide = ErapSlide::new_blank();
        slide.set_title_plain(title);
        // Multiple <a:t> after title become body paragraphs.
        slide.body_frame = era_office_richtext::TextFrame::from_plain(body);
        slides.push(slide);
    }
    if slides
        .iter()
        .all(|s| s.title().is_empty() && s.body().is_empty())
    {
        bail!("empty slide text");
    }
    deck.slides = slides;
    Ok(deck)
}

fn slide_num(name: &str) -> u32 {
    name.trim_start_matches("ppt/slides/slide")
        .trim_end_matches(".xml")
        .parse()
        .unwrap_or(0)
}

fn extract_a_t(xml: &str) -> Vec<String> {
    let mut out = Vec::new();
    let mut rest = xml;
    while let Some(start) = rest.find("<a:t>") {
        rest = &rest[start + 5..];
        if let Some(end) = rest.find("</a:t>") {
            out.push(unescape(&rest[..end]));
            rest = &rest[end + 6..];
        } else {
            break;
        }
    }
    out
}

fn xml(s: &str) -> String {
    s.replace('&', "&amp;")
        .replace('<', "&lt;")
        .replace('>', "&gt;")
}

fn unescape(s: &str) -> String {
    s.replace("&lt;", "<")
        .replace("&gt;", ">")
        .replace("&amp;", "&")
}

#[cfg(test)]
mod tests {
    use super::*;
    use era_office_richtext::{Block, TextFrame};

    #[test]
    fn golden_pptx_roundtrip_plain() {
        let mut deck = ErapDeck::empty();
        deck.slides[0].set_title_plain("Agenda");
        deck.slides[0].set_body_plain("Item");
        let path = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("testdata/az_deck.golden.txt");
        if std::env::var("UPDATE_GOLDEN").ok().as_deref() == Some("1") {
            std::fs::create_dir_all(path.parent().unwrap()).ok();
            std::fs::write(&path, deck.dump_plain()).unwrap();
        }
        assert!(path.exists());
        let want = std::fs::read_to_string(&path)
            .unwrap()
            .replace("\r\n", "\n")
            .trim_end()
            .to_string();
        assert_eq!(deck.dump_plain(), want);

        let bytes = export_pptx(&deck).unwrap();
        let back = import_pptx(&bytes).unwrap();
        assert_eq!(back.slides.len(), 1);
        assert_eq!(back.slides[0].title(), "Agenda");
    }

    #[test]
    fn multi_slide_export_import() {
        let mut deck = ErapDeck::empty();
        deck.slides[0].set_title_plain("One");
        deck.slides[0].set_body_plain("A");
        let mut s2 = ErapSlide::new_blank();
        s2.id = "s2".into();
        s2.set_title_plain("Two");
        s2.set_body_plain("B");
        deck.slides.push(s2);
        let bytes = export_pptx(&deck).unwrap();
        let back = import_pptx(&bytes).unwrap();
        assert_eq!(back.slides.len(), 2);
        assert_eq!(back.slides[0].title(), "One");
        assert_eq!(back.slides[1].title(), "Two");
    }

    #[test]
    fn export_pptx_embeds_data_image_and_notes() {
        let mut deck = ErapDeck::empty();
        deck.slides[0].set_title_plain("Talk");
        deck.slides[0].set_body_plain("Body");
        deck.slides[0].notes = "Speak".into();
        deck.slides[0].background = Some("#112233".into());
        deck.slides[0].image_url = Some(
            "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
                .into(),
        );
        // Align coverage (does not break import of plain a:t).
        let mut centered = Block::paragraph("b1", "Center me");
        centered.align = TextAlign::Center;
        deck.slides[0].body_frame = TextFrame {
            blocks: vec![centered],
        };

        let bytes = export_pptx(&deck).unwrap();
        let mut archive = ZipArchive::new(Cursor::new(&bytes)).unwrap();
        let names: Vec<String> = (0..archive.len())
            .map(|i| archive.by_index(i).unwrap().name().to_string())
            .collect();
        assert!(
            names.iter().any(|n| n.starts_with("ppt/media/")),
            "zip must contain media: {names:?}"
        );
        assert!(
            names
                .iter()
                .any(|n| n.starts_with("ppt/notesSlides/notesSlide")),
            "zip must contain notesSlides: {names:?}"
        );

        let mut slide1 = String::new();
        archive
            .by_name("ppt/slides/slide1.xml")
            .unwrap()
            .read_to_string(&mut slide1)
            .unwrap();
        assert!(
            slide1.contains(r#"srgbClr val="112233""#) || slide1.contains("112233"),
            "slide must emit solid bg: {slide1}"
        );
        assert!(slide1.contains("r:embed="));
        assert!(slide1.contains(r#"algn="ctr""#));

        let mut notes = String::new();
        archive
            .by_name("ppt/notesSlides/notesSlide1.xml")
            .unwrap()
            .read_to_string(&mut notes)
            .unwrap();
        assert!(notes.contains("Speak"));

        // import_pptx still works
        let back = import_pptx(&bytes).unwrap();
        assert_eq!(back.slides[0].title(), "Talk");
    }

    #[test]
    fn parse_solid_hex_rgb_and_gradient() {
        assert_eq!(parse_solid_hex("#112233").as_deref(), Some("112233"));
        assert_eq!(parse_solid_hex("#abc").as_deref(), Some("AABBCC"));
        assert_eq!(
            parse_solid_hex("linear-gradient(#ff0000, #00ff00)").as_deref(),
            Some("FF0000")
        );
        assert!(parse_solid_hex("not-a-color").is_none());
    }

    #[test]
    fn two_column_exports_body2_shape() {
        let mut deck = ErapDeck::empty();
        deck.slides[0].layout = "two_column".into();
        deck.slides[0].set_title_plain("T");
        deck.slides[0].set_body_plain("L");
        deck.slides[0].body2_frame = TextFrame::from_plain("R");
        let bytes = export_pptx(&deck).unwrap();
        let mut archive = ZipArchive::new(Cursor::new(&bytes)).unwrap();
        let mut slide1 = String::new();
        archive
            .by_name("ppt/slides/slide1.xml")
            .unwrap()
            .read_to_string(&mut slide1)
            .unwrap();
        assert_eq!(slide1.matches("<p:sp>").count(), 3);
        assert!(slide1.contains(">R</a:t>") || slide1.contains("<a:t>R</a:t>"));
    }
}
