use std::collections::BTreeMap;
use std::io::{Cursor, Read, Write};

use anyhow::{bail, Context, Result};
use zip::ZipArchive;

use crate::calc::recalc;
use crate::model::{Cell, EratSheet, SheetTab};

/// Multi-sheet xlsx import: workbook sheet list + rels, shared strings, formulas, values.
pub fn import_xlsx(data: &[u8]) -> Result<EratSheet> {
    let mut archive = ZipArchive::new(Cursor::new(data)).context("open xlsx")?;

    let shared = load_shared_strings(&mut archive);

    let mut workbook_xml = String::new();
    archive
        .by_name("xl/workbook.xml")
        .context("workbook.xml")?
        .read_to_string(&mut workbook_xml)?;

    let mut rels_xml = String::new();
    if let Ok(mut f) = archive.by_name("xl/_rels/workbook.xml.rels") {
        f.read_to_string(&mut rels_xml)?;
    }
    let rel_targets = parse_workbook_rels(&rels_xml);
    let sheet_entries = parse_workbook_sheets(&workbook_xml);

    let mut tabs: Vec<SheetTab> = Vec::new();
    if sheet_entries.is_empty() {
        // Fallback: single sheet1 without workbook metadata.
        if let Ok(mut f) = archive.by_name("xl/worksheets/sheet1.xml") {
            let mut xml = String::new();
            f.read_to_string(&mut xml)?;
            tabs.push(SheetTab {
                name: "Sheet1".into(),
                cells: parse_sheet_cells(&xml, &shared),
                charts: vec![],
                protected: false,
                freeze_rows: 0,
                freeze_cols: 0,
                merges: vec![],
                protected_ranges: vec![],
                filter_criteria: None,
                scenarios: None,
            });
        } else {
            bail!("no worksheets found in xlsx");
        }
    } else {
        for (name, rid) in sheet_entries {
            let target = rel_targets
                .get(&rid)
                .cloned()
                .unwrap_or_else(|| format!("worksheets/sheet{}.xml", tabs.len() + 1));
            let path = worksheet_zip_path(&target);
            let mut xml = String::new();
            archive
                .by_name(&path)
                .with_context(|| format!("worksheet {path}"))?
                .read_to_string(&mut xml)?;
            tabs.push(SheetTab {
                name,
                cells: parse_sheet_cells(&xml, &shared),
                charts: vec![],
                protected: false,
                freeze_rows: 0,
                freeze_cols: 0,
                merges: vec![],
                protected_ranges: vec![],
                filter_criteria: None,
                scenarios: None,
            });
        }
    }

    let mut sheet = EratSheet::empty();
    sheet.sheets = tabs;
    sheet.active_sheet = 0;
    sheet.normalize_tabs();
    sheet.sync_active_from_tab();

    // Recalc every tab (formulas → cached values), leave active on sheet0.
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

/// Export all tabs (or active cells if tabs empty) with formulas as `<f>` + cached `<v>`.
pub fn export_xlsx(sheet: &EratSheet) -> Result<Vec<u8>> {
    let mut work = sheet.clone();
    work.normalize_tabs();
    work.flush_active_to_tab();

    let tabs: Vec<(String, BTreeMap<String, Cell>)> = if work.sheets.is_empty() {
        vec![("Sheet1".into(), work.cells.clone())]
    } else {
        work.sheets
            .iter()
            .map(|t| (t.name.clone(), t.cells.clone()))
            .collect()
    };

    let mut sheet_parts: Vec<(String, String)> = Vec::new(); // (filename, xml)
    let mut wb_sheets = String::new();
    let mut wb_rels = String::new();
    let mut content_overrides = String::new();

    for (i, (name, cells)) in tabs.iter().enumerate() {
        let idx = i + 1;
        let rid = format!("rId{idx}");
        let filename = format!("sheet{idx}.xml");
        let part_path = format!("/xl/worksheets/{filename}");
        let sheet_xml = build_worksheet_xml(cells);
        sheet_parts.push((filename.clone(), sheet_xml));

        wb_sheets.push_str(&format!(
            r#"<sheet name="{name}" sheetId="{idx}" r:id="{rid}"/>"#,
            name = xml_escape(name),
            idx = idx,
            rid = rid
        ));
        wb_rels.push_str(&format!(
            r#"<Relationship Id="{rid}" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/{filename}"/>"#
        ));
        content_overrides.push_str(&format!(
            r#"<Override PartName="{part_path}" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>"#
        ));
    }

    let workbook = format!(
        r#"<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>{wb_sheets}</sheets></workbook>"#
    );
    let wb_rels_xml = format!(
        r#"<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">{wb_rels}</Relationships>"#
    );
    let content_types = format!(
        r#"<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
{content_overrides}
</Types>"#
    );
    let rels = r#"<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>"#;

    let mut buf = Cursor::new(Vec::new());
    {
        let mut zip = zip::ZipWriter::new(&mut buf);
        let opts = zip::write::SimpleFileOptions::default();
        zip.start_file("[Content_Types].xml", opts)?;
        zip.write_all(content_types.as_bytes())?;
        zip.start_file("_rels/.rels", opts)?;
        zip.write_all(rels.as_bytes())?;
        zip.start_file("xl/workbook.xml", opts)?;
        zip.write_all(workbook.as_bytes())?;
        zip.start_file("xl/_rels/workbook.xml.rels", opts)?;
        zip.write_all(wb_rels_xml.as_bytes())?;
        for (filename, xml) in &sheet_parts {
            zip.start_file(format!("xl/worksheets/{filename}"), opts)?;
            zip.write_all(xml.as_bytes())?;
        }
        zip.finish()?;
    }
    Ok(buf.into_inner())
}

fn load_shared_strings(archive: &mut ZipArchive<Cursor<&[u8]>>) -> Vec<String> {
    let mut shared: Vec<String> = Vec::new();
    if let Ok(mut f) = archive.by_name("xl/sharedStrings.xml") {
        let mut ss = String::new();
        if f.read_to_string(&mut ss).is_err() {
            return shared;
        }
        for part in ss.split("<t").skip(1) {
            if let Some(rest) = part.strip_prefix('>') {
                if let Some(end) = rest.find("</t>") {
                    shared.push(xml_unescape(&rest[..end]));
                }
            } else if let Some(gt) = part.find('>') {
                let rest = &part[gt + 1..];
                if let Some(end) = rest.find("</t>") {
                    shared.push(xml_unescape(&rest[..end]));
                }
            }
        }
    }
    shared
}

fn parse_workbook_sheets(workbook_xml: &str) -> Vec<(String, String)> {
    let mut out = Vec::new();
    for chunk in workbook_xml.split("<sheet ").skip(1) {
        let name = attr_value(chunk, "name").unwrap_or_else(|| format!("Sheet{}", out.len() + 1));
        let rid = attr_value(chunk, "r:id")
            .or_else(|| attr_value(chunk, "id"))
            .unwrap_or_default();
        if rid.is_empty() {
            continue;
        }
        out.push((xml_unescape(&name), rid));
    }
    out
}

fn parse_workbook_rels(rels_xml: &str) -> BTreeMap<String, String> {
    let mut map = BTreeMap::new();
    for chunk in rels_xml.split("<Relationship ").skip(1) {
        let Some(id) = attr_value(chunk, "Id") else {
            continue;
        };
        let Some(target) = attr_value(chunk, "Target") else {
            continue;
        };
        let ty = attr_value(chunk, "Type").unwrap_or_default();
        if ty.contains("worksheet") || target.contains("worksheets/") {
            map.insert(id, target);
        }
    }
    map
}

fn worksheet_zip_path(target: &str) -> String {
    let t = target.trim_start_matches('/');
    if t.starts_with("xl/") {
        t.to_string()
    } else {
        format!("xl/{t}")
    }
}

fn parse_sheet_cells(sheet_xml: &str, shared: &[String]) -> BTreeMap<String, Cell> {
    let mut cells = BTreeMap::new();
    for chunk in sheet_xml.split("<c ").skip(1) {
        let Some(addr) = attr_value(chunk, "r") else {
            continue;
        };
        let addr = addr.to_uppercase();
        let is_shared = chunk.contains("t=\"s\"");
        let formula = extract_tag_text(chunk, "f").map(|f| {
            let f = f.trim();
            if f.starts_with('=') {
                f.to_string()
            } else {
                format!("={f}")
            }
        });
        let value = if let Some(raw) = extract_tag_text(chunk, "v") {
            if is_shared {
                if let Ok(idx) = raw.parse::<usize>() {
                    shared.get(idx).cloned().unwrap_or_default()
                } else {
                    xml_unescape(&raw)
                }
            } else {
                xml_unescape(&raw)
            }
        } else {
            String::new()
        };
        let formula = formula.unwrap_or_default();
        if value.is_empty() && formula.is_empty() {
            continue;
        }
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
    cells
}

fn build_worksheet_xml(cells: &BTreeMap<String, Cell>) -> String {
    let mut row_map: BTreeMap<u32, Vec<(String, &Cell)>> = BTreeMap::new();
    for (addr, cell) in cells {
        if let Some((_, row)) = parse_addr(addr) {
            row_map.entry(row).or_default().push((addr.clone(), cell));
        }
    }
    let mut rows_xml = String::new();
    for (row, cells) in row_map {
        rows_xml.push_str(&format!("<row r=\"{row}\">"));
        for (addr, cell) in cells {
            rows_xml.push_str(&cell_xml(&addr, cell));
        }
        rows_xml.push_str("</row>");
    }
    format!(
        r#"<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>{rows_xml}</sheetData></worksheet>"#
    )
}

fn cell_xml(addr: &str, cell: &Cell) -> String {
    let v = xml_escape(&cell.value);
    if cell.formula.is_empty() {
        format!("<c r=\"{addr}\"><v>{v}</v></c>")
    } else {
        let f = cell.formula.trim_start_matches('=');
        let f = xml_escape(f);
        format!("<c r=\"{addr}\"><f>{f}</f><v>{v}</v></c>")
    }
}

fn attr_value(chunk: &str, name: &str) -> Option<String> {
    let needle = format!("{name}=\"");
    let start = chunk.find(&needle)? + needle.len();
    let rest = &chunk[start..];
    let end = rest.find('"')?;
    Some(rest[..end].to_string())
}

fn extract_tag_text(chunk: &str, tag: &str) -> Option<String> {
    // `<f>...</f>` or `<f attr="..">...</f>`
    let open = format!("<{tag}");
    let idx = chunk.find(&open)?;
    let after = &chunk[idx + open.len()..];
    let gt = after.find('>')?;
    // Self-closing
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

#[allow(dead_code)]
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

pub fn fixture_sum_xlsx() -> Result<Vec<u8>> {
    let mut s = EratSheet::empty();
    s.set_cell("A1", "10", "");
    s.set_cell("A2", "20", "");
    s.set_cell("A3", "30", "=SUM(A1:A2)");
    recalc(&mut s);
    export_xlsx(&s)
}

#[cfg(test)]
mod convert_tests {
    use super::*;

    #[test]
    fn golden_xlsx_sum_roundtrip() {
        let xlsx = fixture_sum_xlsx().unwrap();
        let imported = import_xlsx(&xlsx).unwrap();
        let golden_path = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
            .join("testdata/az_sum.golden.txt");
        if std::env::var("UPDATE_GOLDEN").ok().as_deref() == Some("1") {
            std::fs::create_dir_all(golden_path.parent().unwrap()).ok();
            std::fs::write(&golden_path, imported.dump_plain()).unwrap();
        }
        assert!(golden_path.exists(), "missing golden");
        let want = std::fs::read_to_string(&golden_path)
            .unwrap()
            .replace("\r\n", "\n")
            .trim_end()
            .to_string();
        assert_eq!(imported.dump_plain(), want);
        let exported = export_xlsx(&imported).unwrap();
        let again = import_xlsx(&exported).unwrap();
        assert_eq!(imported.dump_plain(), again.dump_plain());
    }

    #[test]
    fn import_export_multi_sheet_roundtrip() {
        let mut s = EratSheet::empty();
        s.rename_sheet(0, "Alpha");
        s.set_cell("A1", "1", "");
        s.add_sheet("Beta");
        s.set_cell("B2", "42", "");
        s.set_cell("C3", "", "=SUM(B2:B2)");
        recalc(&mut s);
        s.flush_active_to_tab();

        let xlsx = export_xlsx(&s).unwrap();
        let imported = import_xlsx(&xlsx).unwrap();

        assert_eq!(imported.sheets.len(), 2);
        assert_eq!(imported.sheets[0].name, "Alpha");
        assert_eq!(imported.sheets[1].name, "Beta");
        assert_eq!(
            imported.sheets[0]
                .cells
                .get("A1")
                .map(|c| c.value.as_str()),
            Some("1")
        );
        assert_eq!(
            imported.sheets[1]
                .cells
                .get("B2")
                .map(|c| c.value.as_str()),
            Some("42")
        );
        assert_eq!(
            imported.sheets[1]
                .cells
                .get("C3")
                .map(|c| c.value.as_str()),
            Some("42")
        );
        assert_eq!(
            imported.sheets[1]
                .cells
                .get("C3")
                .map(|c| c.formula.as_str()),
            Some("=SUM(B2:B2)")
        );
        // Active sheet0 cells synced.
        assert_eq!(imported.active_sheet, 0);
        assert_eq!(imported.cells.get("A1").map(|c| c.value.as_str()), Some("1"));
    }

    #[test]
    fn fuzz_xlsx_smoke_random_bytes() {
        for i in 0..32 {
            let data: Vec<u8> = (0..i).map(|b| (b * 13) as u8).collect();
            let _ = import_xlsx(&data);
        }
    }
}
