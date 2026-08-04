use std::collections::{BTreeMap, BTreeSet};
use std::io::{Cursor, Read, Write};

use anyhow::{bail, Context, Result};
use zip::write::SimpleFileOptions;
use zip::{CompressionMethod, ZipArchive, ZipWriter};

use crate::calc::recalc;
use crate::model::{Cell, EratSheet, SheetTab};

const ODS_MIME: &str = "application/vnd.oasis.opendocument.spreadsheet";
const BORDER_PT: &str = "0.74pt solid #000000";

/// Thicker-lite ODF spreadsheet (.ods): multi-sheet, formulas, number formats, wrap/border lite.
pub fn export_ods(sheet: &EratSheet) -> Result<Vec<u8>> {
    let content = build_content_xml(sheet);
    let styles = r#"<?xml version="1.0" encoding="UTF-8"?>
<office:document-styles xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
  xmlns:style="urn:oasis:names:tc:opendocument:xmlns:style:1.0"
  xmlns:fo="urn:oasis:names:tc:opendocument:xmlns:xsl-fo-compatible:1.0"
  xmlns:number="urn:oasis:names:tc:opendocument:xmlns:datastyle:1.0"
  office:version="1.2">
  <office:styles>
    <number:number-style style:name="N0"><number:number number:decimal-places="0" number:min-integer-digits="1"/></number:number-style>
    <number:number-style style:name="N2"><number:number number:decimal-places="2" number:min-integer-digits="1"/></number:number-style>
    <number:percentage-style style:name="N%"><number:number number:decimal-places="2" number:min-integer-digits="1"/><number:text>%</number:text></number:percentage-style>
    <style:style style:name="Default" style:family="table-cell"/>
    <style:style style:name="ce0" style:family="table-cell" style:data-style-name="N0"/>
    <style:style style:name="ce2" style:family="table-cell" style:data-style-name="N2"/>
    <style:style style:name="cep" style:family="table-cell" style:data-style-name="N%"/>
  </office:styles>
</office:document-styles>"#;
    let meta = r#"<?xml version="1.0" encoding="UTF-8"?>
<office:document-meta xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
  xmlns:meta="urn:oasis:names:tc:opendocument:xmlns:meta:1.0" office:version="1.2">
  <office:meta>
    <meta:generator>ERA Tables</meta:generator>
  </office:meta>
</office:document-meta>"#;
    let manifest = r#"<?xml version="1.0" encoding="UTF-8"?>
<manifest:manifest xmlns:manifest="urn:oasis:names:tc:opendocument:xmlns:manifest:1.0" manifest:version="1.2">
  <manifest:file-entry manifest:full-path="/" manifest:media-type="application/vnd.oasis.opendocument.spreadsheet"/>
  <manifest:file-entry manifest:full-path="content.xml" manifest:media-type="text/xml"/>
  <manifest:file-entry manifest:full-path="styles.xml" manifest:media-type="text/xml"/>
  <manifest:file-entry manifest:full-path="meta.xml" manifest:media-type="text/xml"/>
</manifest:manifest>"#;

    let mut buf = Cursor::new(Vec::new());
    {
        let mut zip = ZipWriter::new(&mut buf);
        let stored = SimpleFileOptions::default().compression_method(CompressionMethod::Stored);
        zip.start_file("mimetype", stored)?;
        zip.write_all(ODS_MIME.as_bytes())?;
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

/// Import ODF spreadsheet (.ods): multi-sheet table:table, cell text + table:formula → EratSheet.
pub fn import_ods(data: &[u8]) -> Result<EratSheet> {
    let mut archive = ZipArchive::new(Cursor::new(data)).context("open ods")?;
    let mut content = String::new();
    archive
        .by_name("content.xml")
        .context("content.xml")?
        .read_to_string(&mut content)?;

    let tabs = parse_ods_tables(&content);
    if tabs.is_empty() {
        bail!("no tables found in ods");
    }

    let mut sheet = EratSheet::empty();
    sheet.sheets = tabs;
    sheet.active_sheet = 0;
    sheet.normalize_tabs();
    sheet.sync_active_from_tab();

    let n = sheet.sheets.len();
    for i in 0..n {
        sheet.switch_sheet(i);
        recalc(&mut sheet);
        sheet.flush_active_to_tab();
    }
    if n > 0 {
        sheet.switch_sheet(0);
    }

    Ok(sheet)
}

fn build_content_xml(sheet: &EratSheet) -> String {
    let tabs: Vec<&SheetTab> = if sheet.sheets.is_empty() {
        Vec::new()
    } else {
        sheet.sheets.iter().collect()
    };
    let mut style_keys: BTreeSet<String> = BTreeSet::new();
    let cell_iter: Box<dyn Iterator<Item = &Cell>> = if tabs.is_empty() {
        Box::new(sheet.cells.values())
    } else {
        Box::new(tabs.iter().flat_map(|t| t.cells.values()))
    };
    for cell in cell_iter {
        let name = style_name_for(cell);
        if needs_auto_style(cell) {
            style_keys.insert(name);
        }
    }
    let auto_styles = build_automatic_styles(&style_keys);

    let mut tables = String::new();
    if tabs.is_empty() {
        tables.push_str(&table_xml("Sheet1", &sheet.cells, &[]));
    } else {
        for tab in tabs {
            tables.push_str(&table_xml(&tab.name, &tab.cells, &tab.merges));
        }
    }
    format!(
        r#"<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
  xmlns:table="urn:oasis:names:tc:opendocument:xmlns:table:1.0"
  xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0"
  xmlns:style="urn:oasis:names:tc:opendocument:xmlns:style:1.0"
  xmlns:fo="urn:oasis:names:tc:opendocument:xmlns:xsl-fo-compatible:1.0"
  office:version="1.2">
  <office:automatic-styles>
{auto}
  </office:automatic-styles>
  <office:body>
    <office:spreadsheet>
{tables}
    </office:spreadsheet>
  </office:body>
</office:document-content>"#,
        auto = auto_styles,
        tables = tables
    )
}

fn needs_auto_style(cell: &Cell) -> bool {
    cell.wrap || cell.border || !cell.border_sides.is_empty()
}

fn border_sides_key(cell: &Cell) -> String {
    if !cell.border_sides.is_empty() {
        cell.border_sides.clone()
    } else if cell.border {
        "tblr".into()
    } else {
        String::new()
    }
}

fn style_name_for(cell: &Cell) -> String {
    let base = match cell.format.as_str() {
        "0.00" => "ce2",
        "%" => "cep",
        _ => "Default",
    };
    if !needs_auto_style(cell) {
        return base.to_string();
    }
    let mut name = base.to_string();
    if cell.wrap {
        name.push_str("_w");
    }
    let sides = border_sides_key(cell);
    if !sides.is_empty() {
        name.push_str("_b");
        name.push_str(&sides);
    }
    name
}

fn build_automatic_styles(keys: &BTreeSet<String>) -> String {
    let mut out = String::new();
    for key in keys {
        let wrap = key.contains("_w");
        let sides = key
            .rsplit_once("_b")
            .map(|(_, s)| s.to_string())
            .unwrap_or_default();
        let parent = if key.starts_with("ce2") {
            "ce2"
        } else if key.starts_with("cep") {
            "cep"
        } else {
            "Default"
        };
        let data = match parent {
            "ce2" => r#" style:data-style-name="N2""#,
            "cep" => r#" style:data-style-name="N%""#,
            _ => "",
        };
        let mut props = String::new();
        if wrap {
            props.push_str(r#" fo:wrap-option="wrap""#);
        }
        if !sides.is_empty() {
            if sides == "tblr" {
                props.push_str(&format!(r#" fo:border="{BORDER_PT}""#));
            } else {
                if sides.contains('t') {
                    props.push_str(&format!(r#" fo:border-top="{BORDER_PT}""#));
                }
                if sides.contains('r') {
                    props.push_str(&format!(r#" fo:border-right="{BORDER_PT}""#));
                }
                if sides.contains('b') {
                    props.push_str(&format!(r#" fo:border-bottom="{BORDER_PT}""#));
                }
                if sides.contains('l') {
                    props.push_str(&format!(r#" fo:border-left="{BORDER_PT}""#));
                }
            }
        }
        out.push_str(&format!(
            r#"    <style:style style:name="{name}" style:family="table-cell" style:parent-style-name="{parent}"{data}>
      <style:table-cell-properties{props}/>
    </style:style>
"#,
            name = xml_escape(key),
            parent = parent,
            data = data,
            props = props
        ));
    }
    out
}

fn table_xml(
    table_name: &str,
    cells: &BTreeMap<String, Cell>,
    merges: &[String],
) -> String {
    let mut max_row = 1u32;
    let mut max_col = 1u32;
    let mut grid: BTreeMap<(u32, u32), &Cell> = BTreeMap::new();
    for (addr, cell) in cells {
        if let Some((c, r)) = parse_addr(addr) {
            max_row = max_row.max(r);
            max_col = max_col.max(c);
            grid.insert((r, c), cell);
        }
    }
    for m in merges {
        if let Some((_c1, _r1, c2, r2)) = parse_range(m) {
            max_row = max_row.max(r2);
            max_col = max_col.max(c2);
        }
    }

    let mut rows_xml = String::new();
    for r in 1..=max_row {
        rows_xml.push_str("<table:table-row>");
        let mut c = 1u32;
        while c <= max_col {
            if let Some((c1, r1, c2, r2)) = merge_covering(merges, c, r) {
                if c == c1 && r == r1 {
                    let span_cols = c2 - c1 + 1;
                    let span_rows = r2 - r1 + 1;
                    let cell = grid.get(&(r, c)).copied();
                    rows_xml.push_str(&cell_xml_spanned(cell, span_cols, span_rows));
                    c = c2 + 1;
                    continue;
                }
                c += 1;
                continue;
            }
            if let Some(cell) = grid.get(&(r, c)) {
                rows_xml.push_str(&cell_xml(cell));
            } else {
                rows_xml.push_str("<table:table-cell/>");
            }
            c += 1;
        }
        rows_xml.push_str("</table:table-row>");
    }

    format!(
        r#"      <table:table table:name="{name}">
{rows}
      </table:table>
"#,
        name = xml_escape(table_name),
        rows = rows_xml
    )
}

fn merge_covering(merges: &[String], col: u32, row: u32) -> Option<(u32, u32, u32, u32)> {
    for m in merges {
        if let Some((c1, r1, c2, r2)) = parse_range(m) {
            if col >= c1 && col <= c2 && row >= r1 && row <= r2 {
                return Some((c1, r1, c2, r2));
            }
        }
    }
    None
}

fn parse_range(s: &str) -> Option<(u32, u32, u32, u32)> {
    let parts: Vec<&str> = s.split(':').collect();
    if parts.len() != 2 {
        return None;
    }
    let (c1, r1) = parse_addr(parts[0])?;
    let (c2, r2) = parse_addr(parts[1])?;
    Some((c1.min(c2), r1.min(r2), c1.max(c2), r1.max(r2)))
}

fn cell_xml(cell: &Cell) -> String {
    cell_xml_spanned(Some(cell), 1, 1)
}

fn cell_xml_spanned(cell: Option<&Cell>, cols: u32, rows: u32) -> String {
    let span = if cols > 1 || rows > 1 {
        format!(
            r#" table:number-columns-spanned="{cols}" table:number-rows-spanned="{rows}""#
        )
    } else {
        String::new()
    };
    let Some(cell) = cell else {
        return format!("<table:table-cell{span}/>");
    };
    let style = style_name_for(cell);
    let display = if !cell.value.is_empty() {
        cell.value.as_str()
    } else if !cell.formula.is_empty() {
        cell.formula.as_str()
    } else {
        ""
    };
    let text = xml_escape(display);
    let formula_attr = if !cell.formula.is_empty() {
        let f = cell.formula.trim();
        let of = if f.starts_with('=') {
            format!("of:{}", &f[1..])
        } else {
            format!("of:{f}")
        };
        format!(r#" table:formula="{}""#, xml_escape(&of))
    } else {
        String::new()
    };
    if let Ok(n) = display.parse::<f64>() {
        return format!(
            r#"<table:table-cell table:style-name="{style}" office:value-type="float" office:value="{n}"{formula_attr}{span}><text:p>{text}</text:p></table:table-cell>"#
        );
    }
    format!(
        r#"<table:table-cell table:style-name="{style}" office:value-type="string"{formula_attr}{span}><text:p>{text}</text:p></table:table-cell>"#
    )
}

fn parse_ods_tables(content: &str) -> Vec<SheetTab> {
    let mut tabs = Vec::new();
    let mut search_from = 0;
    let marker = "<table:table";
    while let Some(rel) = content[search_from..].find(marker) {
        let start = search_from + rel;
        let after_tag = &content[start + marker.len()..];
        let first = after_tag.as_bytes().first().copied();
        // Skip table:table-row / table:table-cell / table:table-column / …
        if !matches!(first, Some(b' ') | Some(b'\n') | Some(b'\t') | Some(b'\r') | Some(b'>')) {
            search_from = start + marker.len();
            continue;
        }
        let Some(gt) = after_tag.find('>') else {
            break;
        };
        let attrs = &after_tag[..gt];
        let name = attr_value(attrs, "table:name")
            .or_else(|| attr_value(attrs, "name"))
            .unwrap_or_else(|| format!("Sheet{}", tabs.len() + 1));
        let body_start = start + marker.len() + gt + 1;
        let Some(end_rel) = content[body_start..].find("</table:table>") else {
            break;
        };
        let body = &content[body_start..body_start + end_rel];
        let (cells, merges) = parse_table_body(body);
        tabs.push(SheetTab {
            name: xml_unescape(&name),
            cells,
            charts: vec![],
            protected: false,
            freeze_rows: 0,
            freeze_cols: 0,
            merges,
            protected_ranges: vec![],
            filter_criteria: None,
            scenarios: None,
        });
        search_from = body_start + end_rel + "</table:table>".len();
    }
    tabs
}

fn parse_table_body(body: &str) -> (BTreeMap<String, Cell>, Vec<String>) {
    let mut cells = BTreeMap::new();
    let mut merges = Vec::new();
    let mut row: u32 = 1;
    let mut search = 0;
    let marker = "<table:table-row";
    while let Some(rel) = body[search..].find(marker) {
        let start = search + rel;
        let after = &body[start + marker.len()..];
        let Some(gt) = after.find('>') else {
            break;
        };
        let attrs = &after[..gt];
        let self_closing = attrs.ends_with('/') || after[..gt + 1].ends_with("/>");
        let row_repeat = attr_u32(attrs, "table:number-rows-repeated").unwrap_or(1);
        let row_body = if self_closing {
            ""
        } else {
            let body_start = start + marker.len() + gt + 1;
            match body[body_start..].find("</table:table-row>") {
                Some(end_rel) => &body[body_start..body_start + end_rel],
                None => "",
            }
        };
        for r_off in 0..row_repeat {
            parse_row_cells(row_body, row + r_off, &mut cells, &mut merges);
        }
        row += row_repeat;
        if self_closing {
            search = start + marker.len() + gt + 1;
        } else if let Some(end_rel) = body[start + marker.len() + gt + 1..].find("</table:table-row>")
        {
            search = start + marker.len() + gt + 1 + end_rel + "</table:table-row>".len();
        } else {
            break;
        }
    }
    (cells, merges)
}

fn parse_row_cells(
    row_xml: &str,
    row: u32,
    cells: &mut BTreeMap<String, Cell>,
    merges: &mut Vec<String>,
) {
    let mut col: u32 = 1;
    let mut search = 0;
    while search < row_xml.len() {
        let rest = &row_xml[search..];
        let cell_rel = rest.find("<table:table-cell");
        let covered_rel = rest.find("<table:covered-table-cell");
        let next = match (cell_rel, covered_rel) {
            (Some(a), Some(b)) if a <= b => (a, false),
            (Some(a), None) => (a, false),
            (None, Some(b)) => (b, true),
            (None, None) => break,
            (Some(_), Some(b)) => (b, true),
        };
        let start = search + next.0;
        let is_covered = next.1;
        let tag = if is_covered {
            "<table:covered-table-cell"
        } else {
            "<table:table-cell"
        };
        let after = &row_xml[start + tag.len()..];
        // Avoid matching table:table-cell-… if any; require space/>/>
        let first = after.as_bytes().first().copied();
        if !matches!(
            first,
            Some(b' ') | Some(b'\n') | Some(b'\t') | Some(b'\r') | Some(b'>') | Some(b'/')
        ) {
            search = start + tag.len();
            continue;
        }
        let Some(gt) = after.find('>') else {
            break;
        };
        let attrs = after[..gt].trim_end_matches('/');
        let self_closing = after[..gt].ends_with('/') || after.as_bytes().get(gt - 1) == Some(&b'/');
        let col_repeat = attr_u32(attrs, "table:number-columns-repeated").unwrap_or(1);
        let col_span = attr_u32(attrs, "table:number-columns-spanned").unwrap_or(1);
        let row_span = attr_u32(attrs, "table:number-rows-spanned").unwrap_or(1);

        let cell_inner = if self_closing || is_covered {
            ""
        } else {
            let body_start = start + tag.len() + gt + 1;
            let close = "</table:table-cell>";
            match row_xml[body_start..].find(close) {
                Some(end_rel) => &row_xml[body_start..body_start + end_rel],
                None => "",
            }
        };

        if !is_covered {
            let formula = attr_value(attrs, "table:formula").map(|f| normalize_ods_formula(&f));
            let mut value = attr_value(attrs, "office:value")
                .or_else(|| attr_value(attrs, "office:string-value"))
                .map(|v| xml_unescape(&v))
                .unwrap_or_default();
            if value.is_empty() {
                if let Some(p) = extract_tag_text(cell_inner, "text:p") {
                    value = xml_unescape(&strip_inner_tags(&p));
                }
            }
            let formula = formula.unwrap_or_default();
            if !value.is_empty() || !formula.is_empty() {
                let addr = format_addr(col, row);
                cells.insert(
                    addr,
                    Cell {
                        value,
                        formula,
                        format: String::new(),
                        ..Default::default()
                    },
                );
            }
            if col_span > 1 || row_span > 1 {
                let end = format_addr(col + col_span - 1, row + row_span - 1);
                let start_addr = format_addr(col, row);
                merges.push(format!("{start_addr}:{end}"));
            }
        }

        // Covered cells and repeated empties advance by repeat; spanned top-left
        // is followed by covered-table-cell siblings in ODF.
        col += col_repeat;

        if self_closing || is_covered {
            search = start + tag.len() + gt + 1;
        } else if let Some(end_rel) = row_xml[start + tag.len() + gt + 1..].find("</table:table-cell>")
        {
            search = start + tag.len() + gt + 1 + end_rel + "</table:table-cell>".len();
        } else {
            break;
        }
    }
}

fn normalize_ods_formula(raw: &str) -> String {
    let f = xml_unescape(raw.trim());
    let f = f.strip_prefix("of:").unwrap_or(&f).trim();
    if f.starts_with('=') {
        f.to_string()
    } else {
        format!("={f}")
    }
}

fn strip_inner_tags(s: &str) -> String {
    let mut out = String::new();
    let mut in_tag = false;
    for ch in s.chars() {
        match ch {
            '<' => in_tag = true,
            '>' => in_tag = false,
            _ if !in_tag => out.push(ch),
            _ => {}
        }
    }
    out
}

fn attr_value(chunk: &str, name: &str) -> Option<String> {
    let patterns = [
        format!(r#"{name}=""#),
        format!(r#"{name}='"#),
    ];
    for (i, pat) in patterns.iter().enumerate() {
        if let Some(idx) = chunk.find(pat.as_str()) {
            let rest = &chunk[idx + pat.len()..];
            let quote = if i == 0 { '"' } else { '\'' };
            let end = rest.find(quote)?;
            return Some(rest[..end].to_string());
        }
    }
    None
}

fn attr_u32(chunk: &str, name: &str) -> Option<u32> {
    attr_value(chunk, name)?.parse().ok()
}

fn extract_tag_text(chunk: &str, tag: &str) -> Option<String> {
    let open = format!("<{tag}");
    let idx = chunk.find(&open)?;
    let after = &chunk[idx + open.len()..];
    let gt = after.find('>')?;
    if after[..gt].ends_with('/') {
        return Some(String::new());
    }
    let rest = &after[gt + 1..];
    let close = format!("</{tag}>");
    let end = rest.find(&close)?;
    Some(rest[..end].to_string())
}

fn parse_addr(addr: &str) -> Option<(u32, u32)> {
    let addr = addr.to_uppercase();
    let mut cols = 0u32;
    let mut i = 0;
    let bytes = addr.as_bytes();
    while i < bytes.len() && bytes[i].is_ascii_alphabetic() {
        cols = cols * 26 + (bytes[i] - b'A' + 1) as u32;
        i += 1;
    }
    let row: u32 = addr[i..].parse().ok()?;
    if cols == 0 || row == 0 {
        return None;
    }
    Some((cols, row))
}

fn format_addr(col: u32, row: u32) -> String {
    let mut n = col;
    let mut s = String::new();
    while n > 0 {
        let rem = ((n - 1) % 26) as u8;
        s.insert(0, (b'A' + rem) as char);
        n = (n - 1) / 26;
    }
    format!("{s}{row}")
}

fn xml_escape(s: &str) -> String {
    s.replace('&', "&amp;")
        .replace('<', "&lt;")
        .replace('>', "&gt;")
        .replace('"', "&quot;")
}

fn xml_unescape(s: &str) -> String {
    s.replace("&quot;", "\"")
        .replace("&lt;", "<")
        .replace("&gt;", ">")
        .replace("&amp;", "&")
}

#[cfg(test)]
mod convert_ods_tests {
    use super::*;

    #[test]
    fn export_ods_contains_mimetype() {
        let mut s = EratSheet::empty();
        s.set_cell("A1", "10", "");
        s.set_cell("B1", "hello", "");
        let bytes = export_ods(&s).unwrap();
        assert!(
            bytes
                .windows(ODS_MIME.len())
                .any(|w| w == ODS_MIME.as_bytes()),
            "zip bytes should contain ODS mimetype string"
        );
        let mut archive = ZipArchive::new(Cursor::new(&bytes)).unwrap();
        let mut mime = String::new();
        archive
            .by_name("mimetype")
            .unwrap()
            .read_to_string(&mut mime)
            .unwrap();
        assert_eq!(mime, ODS_MIME);
        assert!(archive.by_name("content.xml").is_ok());
        assert!(archive.by_name("META-INF/manifest.xml").is_ok());
        assert!(archive.by_name("styles.xml").is_ok());
        assert!(archive.by_name("meta.xml").is_ok());
    }

    #[test]
    fn export_ods_writes_formula_and_second_sheet() {
        let mut s = EratSheet::empty();
        s.set_cell("A1", "2", "");
        s.set_cell("B1", "4", "=A1*2");
        s.add_sheet("Second");
        s.set_cell("A1", "x", "");
        let bytes = export_ods(&s).unwrap();
        let mut archive = ZipArchive::new(Cursor::new(&bytes)).unwrap();
        let mut content = String::new();
        archive
            .by_name("content.xml")
            .unwrap()
            .read_to_string(&mut content)
            .unwrap();
        assert!(content.contains("table:formula="));
        assert!(content.contains("table:name=\"Second\"") || content.contains("Second"));
    }

    #[test]
    fn export_ods_writes_wrap_and_border() {
        let mut s = EratSheet::empty();
        s.set_cell("A1", "wrap me", "");
        s.set_cell_style("A1", None, None, Some(true), Some(true), None, None);
        s.set_cell("B1", "sides", "");
        s.set_cell_style("B1", None, None, None, None, Some("tb".into()), None);
        let bytes = export_ods(&s).unwrap();
        let mut archive = ZipArchive::new(Cursor::new(&bytes)).unwrap();
        let mut content = String::new();
        archive
            .by_name("content.xml")
            .unwrap()
            .read_to_string(&mut content)
            .unwrap();
        assert!(content.contains(r#"fo:wrap-option="wrap""#));
        assert!(
            content.contains(r#"fo:border=""#) || content.contains("fo:border-top="),
            "expected border lite attrs"
        );
        assert!(content.contains("fo:border-top=") || content.contains("fo:border-bottom="));
    }

    #[test]
    fn export_import_ods_roundtrip_multisheet_formula() {
        let mut s = EratSheet::empty();
        s.set_cell("A1", "2", "");
        s.set_cell("A2", "2", "");
        s.set_cell("B1", "", "=SUM(A1:A2)");
        s.add_sheet("Second");
        s.set_cell("A1", "hello", "");
        s.switch_sheet(0);
        recalc(&mut s);
        s.flush_active_to_tab();

        let bytes = export_ods(&s).unwrap();
        let imported = import_ods(&bytes).unwrap();

        assert_eq!(imported.sheets.len(), 2);
        assert_eq!(imported.sheets[0].name, "Sheet1");
        assert_eq!(imported.sheets[1].name, "Second");
        assert!(imported.sheets[0].filter_criteria.is_none());
        assert!(imported.sheets[1].filter_criteria.is_none());

        let a1 = imported.sheets[0].cells.get("A1").expect("A1");
        assert_eq!(a1.value, "2");
        let b1 = imported.sheets[0].cells.get("B1").expect("B1");
        assert_eq!(b1.formula, "=SUM(A1:A2)");
        assert_eq!(b1.value, "4");

        let second_a1 = imported.sheets[1].cells.get("A1").expect("Second!A1");
        assert_eq!(second_a1.value, "hello");
    }
}
