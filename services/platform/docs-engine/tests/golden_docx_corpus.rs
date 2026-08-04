//! Table-driven docx golden corpus (AZ memo variants).

use era_docs_engine::convert::docx_import::{fixture_docx, import_docx, minimal_docx};
use era_docs_engine::convert::export_docx;

fn read_golden_text(path: &std::path::Path) -> String {
    std::fs::read_to_string(path)
        .expect("read golden")
        .replace("\r\n", "\n")
        .trim_end()
        .to_string()
}

fn assert_roundtrip(paragraphs: &[&str], golden_name: &str) {
    let docx = minimal_docx(paragraphs).expect("build docx");
    let imported = import_docx(&docx).expect("import");
    let golden_path = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("testdata")
        .join(golden_name);

    if std::env::var("UPDATE_GOLDEN").ok().as_deref() == Some("1") {
        std::fs::write(&golden_path, imported.plain_text()).expect("write golden");
    }

    assert!(
        golden_path.exists(),
        "missing golden {golden_name} (set UPDATE_GOLDEN=1 to create)"
    );
    let want = read_golden_text(&golden_path);
    assert_eq!(imported.plain_text(), want, "golden {golden_name}");

    let exported = export_docx(&imported).expect("export");
    let reimported = import_docx(&exported).expect("reimport");
    assert_eq!(imported.plain_text(), reimported.plain_text());
}

fn assert_fixture_roundtrip(body_xml: &str, golden_name: &str) {
    let docx = fixture_docx(body_xml).expect("build fixture docx");
    let imported = import_docx(&docx).expect("import");
    let golden_path = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("testdata")
        .join(golden_name);

    if std::env::var("UPDATE_GOLDEN").ok().as_deref() == Some("1") {
        std::fs::write(&golden_path, imported.plain_text()).expect("write golden");
    }

    assert!(
        golden_path.exists(),
        "missing golden {golden_name} (set UPDATE_GOLDEN=1 to create)"
    );
    let want = read_golden_text(&golden_path);
    assert_eq!(imported.plain_text(), want, "golden {golden_name}");

    let exported = export_docx(&imported).expect("export");
    let reimported = import_docx(&exported).expect("reimport");
    assert_eq!(imported.plain_text(), reimported.plain_text());
}

#[test]
fn golden_docx_three_paragraphs() {
    assert_roundtrip(
        &["Line one", "Line two", "Line three"],
        "az_three_lines.golden.txt",
    );
}

#[test]
fn golden_docx_single_line() {
    assert_roundtrip(&["Only line"], "az_single_line.golden.txt");
}

#[test]
fn golden_docx_unicode_az() {
    assert_roundtrip(
        &["Salam — Azərbaycan", "Rəsmi mətn"],
        "az_unicode.golden.txt",
    );
}

#[test]
fn golden_docx_heading_levels() {
    const W: &str = "http://schemas.openxmlformats.org/wordprocessingml/2006/main";
    let body = format!(
        r#"<w:p xmlns:w="{W}"><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>Title</w:t></w:r></w:p>
<w:p xmlns:w="{W}"><w:pPr><w:pStyle w:val="Heading2"/></w:pPr><w:r><w:t>Section</w:t></w:r></w:p>
<w:p xmlns:w="{W}"><w:pPr><w:pStyle w:val="Heading3"/></w:pPr><w:r><w:t>Sub</w:t></w:r></w:p>"#
    );
    assert_fixture_roundtrip(&body, "az_headings.golden.txt");
}

#[test]
fn golden_docx_mixed_spans() {
    const W: &str = "http://schemas.openxmlformats.org/wordprocessingml/2006/main";
    let body = format!(
        r#"<w:p xmlns:w="{W}"><w:r><w:rPr><w:b/></w:rPr><w:t>Bold</w:t></w:r>
<w:r><w:t> and </w:t></w:r>
<w:r><w:rPr><w:i/></w:rPr><w:t>italic</w:t></w:r></w:p>"#
    );
    assert_fixture_roundtrip(&body, "az_mixed_spans.golden.txt");
}

#[test]
fn golden_docx_nested_list() {
    const W: &str = "http://schemas.openxmlformats.org/wordprocessingml/2006/main";
    let body = format!(
        r#"<w:p xmlns:w="{W}"><w:pPr><w:numPr><w:ilvl w:val="0"/><w:numId w:val="2"/></w:numPr></w:pPr><w:r><w:t>Parent</w:t></w:r></w:p>
<w:p xmlns:w="{W}"><w:pPr><w:numPr><w:ilvl w:val="1"/><w:numId w:val="2"/></w:numPr></w:pPr><w:r><w:t>Child</w:t></w:r></w:p>"#
    );
    assert_fixture_roundtrip(&body, "az_nested_list.golden.txt");
}
