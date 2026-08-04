use std::collections::BTreeMap;

use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Default, Serialize, Deserialize, PartialEq)]
pub struct Cell {
    #[serde(default)]
    pub value: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub formula: String,
    /// Display format: "", "0.00", "%", "date"
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub format: String,
    /// O-FMT-2: cell chrome (Excel-class lite).
    #[serde(default, skip_serializing_if = "std::ops::Not::not")]
    pub bold: bool,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub align: String, // left | center | right
    #[serde(default, skip_serializing_if = "std::ops::Not::not")]
    pub wrap: bool,
    #[serde(default, skip_serializing_if = "std::ops::Not::not")]
    pub border: bool,
    /// T-LITE: per-side borders as subset of "tblr" (top/right/bottom/left).
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub border_sides: String,
    /// O-UX: cell note / comment lite (shown as tooltip + Notes rail).
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub note: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct ChartSpec {
    #[serde(default = "default_chart_type")]
    pub chart_type: String,
    pub range: String,
    #[serde(default)]
    pub title: String,
}

fn default_chart_type() -> String {
    "bar".into()
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct SheetTab {
    pub name: String,
    #[serde(default)]
    pub cells: BTreeMap<String, Cell>,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub charts: Vec<ChartSpec>,
    /// Wave W2: sheet protect lite (blocks cell edits when true).
    #[serde(default, skip_serializing_if = "std::ops::Not::not")]
    pub protected: bool,
    /// ERA+: freeze first N data rows (sticky).
    #[serde(default, skip_serializing_if = "is_zero_u32")]
    pub freeze_rows: u32,
    /// ERA+: freeze first N columns (sticky).
    #[serde(default, skip_serializing_if = "is_zero_u32")]
    pub freeze_cols: u32,
    /// Lite merge ranges as A1:B1 strings (client colspan).
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub merges: Vec<String>,
    /// Lite protected ranges as A1:B2 strings (block edits even if sheet unprotected).
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub protected_ranges: Vec<String>,
    /// T-LITE: client filter criteria JSON (optional persist in .erat).
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub filter_criteria: Option<serde_json::Value>,
    /// O-STUB Lite: named column scenarios `{ name: { ADDR: value } }`.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub scenarios: Option<serde_json::Value>,
}

fn is_zero_u32(v: &u32) -> bool {
    *v == 0
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct EratSheet {
    #[serde(default)]
    pub id: String,
    #[serde(default)]
    pub tenant_id: String,
    #[serde(default)]
    pub drive_object_id: String,
    #[serde(default = "default_format")]
    pub format: String,
    #[serde(default)]
    pub name: String,
    pub rows: u32,
    pub cols: u32,
    /// Active sheet cells (mirrored from sheets[active_sheet]).
    #[serde(default)]
    pub cells: BTreeMap<String, Cell>,
    #[serde(default)]
    pub sheets: Vec<SheetTab>,
    #[serde(default)]
    pub active_sheet: usize,
}

fn default_format() -> String {
    "erat".into()
}

impl EratSheet {
    pub fn empty() -> Self {
        let mut s = Self {
            id: String::new(),
            tenant_id: String::new(),
            drive_object_id: String::new(),
            format: "erat".into(),
            name: "sheet.erat".into(),
            rows: 10_000,
            cols: 621, // A–WW
            cells: BTreeMap::new(),
            sheets: vec![SheetTab {
                name: "Sheet1".into(),
                cells: BTreeMap::new(),
                charts: vec![],
                protected: false,
                freeze_rows: 0,
                freeze_cols: 0,
                merges: vec![],
                protected_ranges: vec![],
                filter_criteria: None,
                scenarios: None,
            }],
            active_sheet: 0,
        };
        s.sync_active_from_tab();
        s
    }

    pub fn normalize_tabs(&mut self) {
        if self.sheets.is_empty() {
            self.sheets.push(SheetTab {
                name: "Sheet1".into(),
                cells: self.cells.clone(),
                charts: vec![],
                protected: false,
                freeze_rows: 0,
                freeze_cols: 0,
                merges: vec![],
                protected_ranges: vec![],
                filter_criteria: None,
                scenarios: None,
            });
            self.active_sheet = 0;
        }
        if self.active_sheet >= self.sheets.len() {
            self.active_sheet = 0;
        }
        // Prefer sheets[active] as source of truth when tabs exist with data,
        // else seed tab from cells (legacy single-sheet docs).
        if self.sheets[self.active_sheet].cells.is_empty() && !self.cells.is_empty() {
            self.sheets[self.active_sheet].cells = self.cells.clone();
        }
        self.sync_active_from_tab();
    }

    pub fn sync_active_from_tab(&mut self) {
        if self.sheets.is_empty() {
            return;
        }
        if self.active_sheet >= self.sheets.len() {
            self.active_sheet = 0;
        }
        self.cells = self.sheets[self.active_sheet].cells.clone();
    }

    pub fn flush_active_to_tab(&mut self) {
        if self.sheets.is_empty() {
            return;
        }
        if self.active_sheet >= self.sheets.len() {
            self.active_sheet = 0;
        }
        self.sheets[self.active_sheet].cells = self.cells.clone();
    }

    pub fn set_cell(&mut self, addr: &str, value: impl Into<String>, formula: impl Into<String>) {
        self.normalize_tabs();
        let key = addr.to_uppercase();
        let prev = self.cells.get(&key).cloned().unwrap_or_default();
        let cell = Cell {
            value: value.into(),
            formula: formula.into(),
            format: prev.format,
            bold: prev.bold,
            align: prev.align,
            wrap: prev.wrap,
            border: prev.border,
            border_sides: prev.border_sides,
            note: prev.note,
        };
        self.cells.insert(key.clone(), cell.clone());
        self.sheets[self.active_sheet].cells.insert(key, cell);
    }

    pub fn set_cell_style(
        &mut self,
        addr: &str,
        bold: Option<bool>,
        align: Option<String>,
        wrap: Option<bool>,
        border: Option<bool>,
        border_sides: Option<String>,
        note: Option<String>,
    ) {
        self.normalize_tabs();
        let key = addr.to_uppercase();
        let cell = self.cells.entry(key.clone()).or_default();
        if let Some(b) = bold {
            cell.bold = b;
        }
        if let Some(a) = align {
            cell.align = a;
        }
        if let Some(w) = wrap {
            cell.wrap = w;
        }
        if let Some(b) = border {
            cell.border = b;
            if b && cell.border_sides.is_empty() {
                cell.border_sides = "tblr".into();
            }
            if !b {
                cell.border_sides.clear();
            }
        }
        if let Some(sides) = border_sides {
            let cleaned: String = sides
                .chars()
                .filter(|c| "tblr".contains(*c))
                .collect();
            cell.border_sides = cleaned.clone();
            cell.border = !cleaned.is_empty();
        }
        if let Some(n) = note {
            cell.note = n;
        }
        self.sheets[self.active_sheet]
            .cells
            .insert(key, cell.clone());
    }

    pub fn set_cell_format(&mut self, addr: &str, format: impl Into<String>) {
        self.normalize_tabs();
        let key = addr.to_uppercase();
        let cell = self.cells.entry(key.clone()).or_default();
        cell.format = format.into();
        self.sheets[self.active_sheet]
            .cells
            .insert(key, cell.clone());
    }

    pub fn clear_cell(&mut self, addr: &str) {
        self.normalize_tabs();
        self.cells.remove(&addr.to_uppercase());
        self.sheets[self.active_sheet]
            .cells
            .remove(&addr.to_uppercase());
    }

    pub fn switch_sheet(&mut self, index: usize) {
        self.normalize_tabs();
        if index >= self.sheets.len() {
            return;
        }
        self.flush_active_to_tab();
        self.active_sheet = index;
        self.sync_active_from_tab();
    }

    pub fn add_sheet(&mut self, name: impl Into<String>) {
        self.normalize_tabs();
        self.flush_active_to_tab();
        self.sheets.push(SheetTab {
            name: name.into(),
            cells: BTreeMap::new(),
            charts: vec![],
            protected: false,
            freeze_rows: 0,
            freeze_cols: 0,
            merges: vec![],
            protected_ranges: vec![],
            filter_criteria: None,
            scenarios: None,
        });
        self.active_sheet = self.sheets.len() - 1;
        self.sync_active_from_tab();
    }

    pub fn set_sheet_protected(&mut self, protected: bool) {
        self.normalize_tabs();
        if let Some(tab) = self.sheets.get_mut(self.active_sheet) {
            tab.protected = protected;
        }
    }

    pub fn is_active_protected(&self) -> bool {
        self.sheets
            .get(self.active_sheet)
            .map(|t| t.protected)
            .unwrap_or(false)
    }

    pub fn set_merges(&mut self, merges: Vec<String>) {
        self.normalize_tabs();
        if let Some(tab) = self.sheets.get_mut(self.active_sheet) {
            tab.merges = merges;
        }
    }

    pub fn active_merges(&self) -> Vec<String> {
        self.sheets
            .get(self.active_sheet)
            .map(|t| t.merges.clone())
            .unwrap_or_default()
    }

    pub fn set_protected_ranges(&mut self, ranges: Vec<String>) {
        self.normalize_tabs();
        if let Some(tab) = self.sheets.get_mut(self.active_sheet) {
            tab.protected_ranges = ranges;
        }
    }

    pub fn active_protected_ranges(&self) -> Vec<String> {
        self.sheets
            .get(self.active_sheet)
            .map(|t| t.protected_ranges.clone())
            .unwrap_or_default()
    }

    /// True if `addr` falls inside any active protected_ranges (A1:B2 lite).
    pub fn addr_in_protected_range(&self, addr: &str) -> bool {
        let Some((c, r)) = parse_addr(addr) else {
            return false;
        };
        for range in self.active_protected_ranges() {
            if let Some(((c1, r1), (c2, r2))) = parse_a1_range(&range) {
                let (min_c, max_c) = (c1.min(c2), c1.max(c2));
                let (min_r, max_r) = (r1.min(r2), r1.max(r2));
                if c >= min_c && c <= max_c && r >= min_r && r <= max_r {
                    return true;
                }
            }
        }
        false
    }

    pub fn set_freeze_panes(&mut self, rows: u32, cols: u32) {
        self.normalize_tabs();
        if let Some(tab) = self.sheets.get_mut(self.active_sheet) {
            tab.freeze_rows = rows;
            tab.freeze_cols = cols;
        }
    }

    pub fn active_freeze(&self) -> (u32, u32) {
        self.sheets
            .get(self.active_sheet)
            .map(|t| (t.freeze_rows, t.freeze_cols))
            .unwrap_or((0, 0))
    }

    pub fn rename_sheet(&mut self, index: usize, name: impl Into<String>) {
        self.normalize_tabs();
        if index < self.sheets.len() {
            self.sheets[index].name = name.into();
        }
    }

    pub fn reorder_sheet(&mut self, from: usize, to: usize) {
        self.normalize_tabs();
        if from >= self.sheets.len() || to >= self.sheets.len() || from == to {
            return;
        }
        self.flush_active_to_tab();
        let tab = self.sheets.remove(from);
        self.sheets.insert(to, tab);
        if self.active_sheet == from {
            self.active_sheet = to;
        } else if from < self.active_sheet && to >= self.active_sheet {
            self.active_sheet -= 1;
        } else if from > self.active_sheet && to <= self.active_sheet {
            self.active_sheet += 1;
        }
        self.sync_active_from_tab();
    }

    /// Insert a row at 1-based `row`, shifting cells down. Adjusts simple formulas A1 refs.
    pub fn insert_row(&mut self, row: u32) {
        self.normalize_tabs();
        if row == 0 {
            return;
        }
        let mut next = BTreeMap::new();
        for (addr, mut cell) in std::mem::take(&mut self.cells) {
            if let Some((c, r)) = parse_addr(&addr) {
                let new_r = if r >= row { r + 1 } else { r };
                if !cell.formula.is_empty() {
                    cell.formula = shift_formula_rows(&cell.formula, row, 1);
                }
                next.insert(addr_of(c, new_r), cell);
            } else {
                next.insert(addr, cell);
            }
        }
        self.cells = next;
        self.flush_active_to_tab();
        self.rows = self.rows.saturating_add(1);
    }

    pub fn delete_row(&mut self, row: u32) {
        self.normalize_tabs();
        if row == 0 {
            return;
        }
        let mut next = BTreeMap::new();
        for (addr, mut cell) in std::mem::take(&mut self.cells) {
            if let Some((c, r)) = parse_addr(&addr) {
                if r == row {
                    continue;
                }
                let new_r = if r > row { r - 1 } else { r };
                if !cell.formula.is_empty() {
                    cell.formula = shift_formula_rows(&cell.formula, row, -1);
                }
                next.insert(addr_of(c, new_r), cell);
            } else {
                next.insert(addr, cell);
            }
        }
        self.cells = next;
        self.flush_active_to_tab();
    }

    pub fn insert_col(&mut self, col: u32) {
        self.normalize_tabs();
        if col == 0 {
            return;
        }
        let mut next = BTreeMap::new();
        for (addr, cell) in std::mem::take(&mut self.cells) {
            if let Some((c, r)) = parse_addr(&addr) {
                let new_c = if c >= col { c + 1 } else { c };
                next.insert(addr_of(new_c, r), cell);
            } else {
                next.insert(addr, cell);
            }
        }
        self.cells = next;
        self.flush_active_to_tab();
        self.cols = self.cols.saturating_add(1);
    }

    pub fn delete_col(&mut self, col: u32) {
        self.normalize_tabs();
        if col == 0 {
            return;
        }
        let mut next = BTreeMap::new();
        for (addr, cell) in std::mem::take(&mut self.cells) {
            if let Some((c, r)) = parse_addr(&addr) {
                if c == col {
                    continue;
                }
                let new_c = if c > col { c - 1 } else { c };
                next.insert(addr_of(new_c, r), cell);
            } else {
                next.insert(addr, cell);
            }
        }
        self.cells = next;
        self.flush_active_to_tab();
    }

    /// Row-aware sort: reorder whole rows by key column (1-based). Moves cell units
    /// (value+formula+style); relative formula refs are not rewritten (MVP limitation).
    pub fn sort_rows_by_col(&mut self, col: u32, ascending: bool) {
        self.normalize_tabs();
        if col == 0 {
            return;
        }
        let mut max_row = 0u32;
        for addr in self.cells.keys() {
            if let Some((_, r)) = parse_addr(addr) {
                max_row = max_row.max(r);
            }
        }
        if max_row == 0 {
            return;
        }
        let mut rows: Vec<(u32, BTreeMap<u32, Cell>)> = Vec::new();
        for r in 1..=max_row {
            let mut row_cells = BTreeMap::new();
            for (addr, cell) in &self.cells {
                if let Some((c, rr)) = parse_addr(addr) {
                    if rr == r {
                        row_cells.insert(c, cell.clone());
                    }
                }
            }
            rows.push((r, row_cells));
        }
        rows.sort_by(|a, b| {
            let key = |m: &BTreeMap<u32, Cell>| {
                m.get(&col)
                    .map(|c| {
                        if !c.formula.is_empty() {
                            c.formula.clone()
                        } else {
                            c.value.clone()
                        }
                    })
                    .unwrap_or_default()
                    .to_lowercase()
            };
            let cmp = key(&a.1).cmp(&key(&b.1));
            if ascending {
                cmp
            } else {
                cmp.reverse()
            }
        });
        // Drop sorted rows then rewrite at sequential positions 1..n
        let mut next = BTreeMap::new();
        for (addr, cell) in self.cells.iter() {
            if let Some((_, r)) = parse_addr(addr) {
                if r > max_row {
                    next.insert(addr.clone(), cell.clone());
                }
            }
        }
        for (new_r, (_, cells)) in (1u32..).zip(rows.into_iter()) {
            for (c, cell) in cells {
                next.insert(addr_of(c, new_r), cell);
            }
        }
        self.cells = next;
        self.flush_active_to_tab();
    }

    pub fn dump_plain(&self) -> String {
        self.cells
            .iter()
            .map(|(a, c)| {
                if c.formula.is_empty() {
                    format!("{a}={}", c.value)
                } else {
                    format!("{a}={} ({})", c.value, c.formula)
                }
            })
            .collect::<Vec<_>>()
            .join("\n")
    }
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

/// Parse A1 or A1:B2 into ((col,row), (col,row)); col/row are 1-based.
fn parse_a1_range(range: &str) -> Option<((u32, u32), (u32, u32))> {
    let s = range.trim().to_uppercase();
    let (a, b) = match s.split_once(':') {
        Some((left, right)) => (left.trim(), right.trim()),
        None => (s.as_str(), s.as_str()),
    };
    let start = parse_addr(a)?;
    let end = parse_addr(b)?;
    Some((start, end))
}

fn addr_of(col: u32, row: u32) -> String {
    let mut n = col;
    let mut s = String::new();
    while n > 0 {
        let rem = ((n - 1) % 26) as u8;
        s.insert(0, (b'A' + rem) as char);
        n = (n - 1) / 26;
    }
    format!("{s}{row}")
}

fn shift_formula_rows(formula: &str, at_row: u32, delta: i32) -> String {
    let mut out = String::new();
    let mut i = 0;
    let bytes = formula.as_bytes();
    while i < bytes.len() {
        if bytes[i].is_ascii_alphabetic() {
            let start = i;
            while i < bytes.len() && bytes[i].is_ascii_alphabetic() {
                i += 1;
            }
            let col = &formula[start..i];
            let num_start = i;
            while i < bytes.len() && bytes[i].is_ascii_digit() {
                i += 1;
            }
            if num_start < i {
                if let Ok(r) = formula[num_start..i].parse::<u32>() {
                    let new_r = if delta > 0 && r >= at_row {
                        r + delta as u32
                    } else if delta < 0 && r > at_row {
                        r - ((-delta) as u32)
                    } else if delta < 0 && r == at_row {
                        r
                    } else {
                        r
                    };
                    out.push_str(col);
                    out.push_str(&new_r.to_string());
                    continue;
                }
            }
            out.push_str(&formula[start..i]);
        } else {
            out.push(formula.as_bytes()[i] as char);
            i += 1;
        }
    }
    out
}

/// Format a numeric/string cell value for display.
pub fn format_display(value: &str, format: &str) -> String {
    if format.is_empty() {
        return value.to_string();
    }
    let n = value.parse::<f64>();
    match format {
        "0.00" => n
            .map(|v| format!("{v:.2}"))
            .unwrap_or_else(|_| value.to_string()),
        "%" => n
            .map(|v| format!("{:.0}%", v * 100.0))
            .unwrap_or_else(|_| value.to_string()),
        "date" => {
            // Lite: treat number as days since 1899-12-30 Excel epoch → YYYY-MM-DD approx via unix
            if let Ok(days) = value.parse::<f64>() {
                let secs = ((days - 25569.0) * 86400.0) as i64;
                format!("day:{secs}")
            } else {
                value.to_string()
            }
        }
        _ => value.to_string(),
    }
}

#[cfg(test)]
mod model_tests {
    use super::*;

    #[test]
    fn insert_row_shifts_cells_and_refs() {
        let mut s = EratSheet::empty();
        s.set_cell("A1", "10", "");
        s.set_cell("A2", "20", "");
        s.set_cell("B1", "", "=A1");
        s.insert_row(1);
        assert!(!s.cells.contains_key("A1") || s.cells.get("A1").map(|c| c.value.as_str()) != Some("10"));
        assert_eq!(s.cells["A2"].value, "10");
        assert_eq!(s.cells["A3"].value, "20");
        assert!(s.cells["B2"].formula.contains("A2") || s.cells["B2"].formula == "=A2");
    }

    #[test]
    fn format_display_percent_and_fixed() {
        assert_eq!(format_display("0.5", "%"), "50%");
        assert_eq!(format_display("1.5", "0.00"), "1.50");
    }

    #[test]
    fn rename_and_reorder_sheets() {
        let mut s = EratSheet::empty();
        s.add_sheet("Sheet2");
        s.rename_sheet(0, "Alpha");
        assert_eq!(s.sheets[0].name, "Alpha");
        s.reorder_sheet(0, 1);
        assert_eq!(s.sheets[1].name, "Alpha");
    }
}
