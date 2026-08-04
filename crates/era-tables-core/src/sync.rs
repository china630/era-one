use serde::{Deserialize, Serialize};

use crate::calc::recalc;
use crate::model::EratSheet;

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(tag = "type", rename_all = "snake_case")]
pub enum SheetOp {
    SetCell {
        addr: String,
        value: String,
        #[serde(default)]
        formula: String,
    },
    ClearCell {
        addr: String,
    },
    SetFormat {
        addr: String,
        format: String,
    },
    /// O-FMT-2: bold / align / wrap / border (+ O-UX note).
    SetCellStyle {
        addr: String,
        #[serde(default)]
        bold: Option<bool>,
        #[serde(default)]
        align: Option<String>,
        #[serde(default)]
        wrap: Option<bool>,
        #[serde(default)]
        border: Option<bool>,
        #[serde(default)]
        border_sides: Option<String>,
        #[serde(default)]
        note: Option<String>,
    },
    /// T-LITE: row-aware sort by 1-based column index.
    SortRange {
        col: u32,
        #[serde(default = "default_true")]
        ascending: bool,
    },
    /// Persist AutoFilter criteria on active tab (JSON).
    SetFilterCriteria {
        #[serde(default)]
        criteria: Option<serde_json::Value>,
    },
    SwitchSheet {
        index: usize,
    },
    AddSheet {
        #[serde(default)]
        name: String,
    },
    RenameSheet {
        index: usize,
        name: String,
    },
    ReorderSheet {
        from: usize,
        to: usize,
    },
    InsertRow {
        row: u32,
    },
    DeleteRow {
        row: u32,
    },
    InsertCol {
        col: u32,
    },
    DeleteCol {
        col: u32,
    },
    ProtectSheet {
        protected: bool,
    },
    /// ERA+: freeze first `rows` data rows and first `cols` columns (not blocked by protect).
    FreezePanes {
        #[serde(default)]
        rows: u32,
        #[serde(default)]
        cols: u32,
    },
    /// Lite: set merge ranges (A1:B1 strings) on active tab.
    SetMerges {
        #[serde(default)]
        merges: Vec<String>,
    },
    /// Lite: set protected ranges (A1:B2 strings) on active tab.
    SetProtectedRanges {
        #[serde(default)]
        ranges: Vec<String>,
    },
    /// Persist chart specs on active tab.
    SetCharts {
        #[serde(default)]
        charts: Vec<crate::model::ChartSpec>,
    },
    /// O-STUB Lite: persist named scenarios on active tab.
    SetScenarios {
        #[serde(default)]
        scenarios: Option<serde_json::Value>,
    },
}

fn default_true() -> bool {
    true
}

#[derive(Debug, Default, Clone)]
pub struct OpLog {
    pub version: u64,
    pub ops: Vec<SheetOp>,
}

impl OpLog {
    pub fn append(&mut self, op: SheetOp) {
        self.version += 1;
        self.ops.push(op);
    }
}

pub fn apply_op(sheet: &mut EratSheet, op: &SheetOp) {
    sheet.normalize_tabs();
    match op {
        SheetOp::SetCell {
            addr,
            value,
            formula,
        } => {
            sheet.set_cell(addr, value, formula);
            recalc(sheet);
            sheet.flush_active_to_tab();
        }
        SheetOp::ClearCell { addr } => {
            sheet.clear_cell(addr);
            recalc(sheet);
            sheet.flush_active_to_tab();
        }
        SheetOp::SwitchSheet { index } => {
            sheet.switch_sheet(*index);
            recalc(sheet);
        }
        SheetOp::AddSheet { name } => {
            let n = if name.trim().is_empty() {
                format!("Sheet{}", sheet.sheets.len() + 1)
            } else {
                name.clone()
            };
            sheet.add_sheet(n);
            recalc(sheet);
        }
        SheetOp::SetFormat { addr, format } => {
            sheet.set_cell_format(addr, format);
            sheet.flush_active_to_tab();
        }
        SheetOp::SetCellStyle {
            addr,
            bold,
            align,
            wrap,
            border,
            border_sides,
            note,
        } => {
            sheet.set_cell_style(
                addr,
                *bold,
                align.clone(),
                *wrap,
                *border,
                border_sides.clone(),
                note.clone(),
            );
            sheet.flush_active_to_tab();
        }
        SheetOp::SortRange { col, ascending } => {
            sheet.sort_rows_by_col(*col, *ascending);
            recalc(sheet);
            sheet.flush_active_to_tab();
        }
        SheetOp::SetFilterCriteria { criteria } => {
            sheet.normalize_tabs();
            if let Some(tab) = sheet.sheets.get_mut(sheet.active_sheet) {
                tab.filter_criteria = criteria.clone();
            }
        }
        SheetOp::SetCharts { charts } => {
            sheet.normalize_tabs();
            if let Some(tab) = sheet.sheets.get_mut(sheet.active_sheet) {
                tab.charts = charts.clone();
            }
        }
        SheetOp::SetScenarios { scenarios } => {
            sheet.normalize_tabs();
            if let Some(tab) = sheet.sheets.get_mut(sheet.active_sheet) {
                tab.scenarios = scenarios.clone();
            }
        }
        SheetOp::RenameSheet { index, name } => {
            sheet.rename_sheet(*index, name);
        }
        SheetOp::ReorderSheet { from, to } => {
            sheet.reorder_sheet(*from, *to);
            recalc(sheet);
        }
        SheetOp::InsertRow { row } => {
            sheet.insert_row(*row);
            recalc(sheet);
            sheet.flush_active_to_tab();
        }
        SheetOp::DeleteRow { row } => {
            sheet.delete_row(*row);
            recalc(sheet);
            sheet.flush_active_to_tab();
        }
        SheetOp::InsertCol { col } => {
            sheet.insert_col(*col);
            recalc(sheet);
            sheet.flush_active_to_tab();
        }
        SheetOp::DeleteCol { col } => {
            sheet.delete_col(*col);
            recalc(sheet);
            sheet.flush_active_to_tab();
        }
        SheetOp::ProtectSheet { protected } => {
            sheet.set_sheet_protected(*protected);
        }
        SheetOp::FreezePanes { rows, cols } => {
            sheet.set_freeze_panes(*rows, *cols);
        }
        SheetOp::SetMerges { merges } => {
            sheet.set_merges(merges.clone());
        }
        SheetOp::SetProtectedRanges { ranges } => {
            sheet.set_protected_ranges(ranges.clone());
        }
    }
}

/// Returns true if the op mutates cell data and is blocked by sheet or range protect.
/// Freeze panes / sheet switch / protect toggle / merges / range setup are allowed.
pub fn blocked_by_protect(sheet: &EratSheet, op: &SheetOp) -> bool {
    if sheet.is_active_protected() {
        return matches!(
            op,
            SheetOp::SetCell { .. }
                | SheetOp::ClearCell { .. }
                | SheetOp::SetFormat { .. }
                | SheetOp::SetCellStyle { .. }
                | SheetOp::SortRange { .. }
                | SheetOp::InsertRow { .. }
                | SheetOp::DeleteRow { .. }
                | SheetOp::InsertCol { .. }
                | SheetOp::DeleteCol { .. }
        );
    }
    match op {
        SheetOp::SetCell { addr, .. }
        | SheetOp::ClearCell { addr }
        | SheetOp::SetFormat { addr, .. }
        | SheetOp::SetCellStyle { addr, .. } => sheet.addr_in_protected_range(addr),
        _ => false,
    }
}

#[cfg(test)]
mod sync_tests {
    use super::*;

    #[test]
    fn sync_two_cell_ops() {
        let mut s = EratSheet::empty();
        apply_op(
            &mut s,
            &SheetOp::SetCell {
                addr: "A1".into(),
                value: "1".into(),
                formula: String::new(),
            },
        );
        apply_op(
            &mut s,
            &SheetOp::SetCell {
                addr: "B1".into(),
                value: "2".into(),
                formula: String::new(),
            },
        );
        assert_eq!(s.cells["A1"].value, "1");
        assert_eq!(s.cells["B1"].value, "2");
    }

    #[test]
    fn multi_sheet_switch_isolates_cells() {
        let mut s = EratSheet::empty();
        apply_op(
            &mut s,
            &SheetOp::SetCell {
                addr: "A1".into(),
                value: "on1".into(),
                formula: String::new(),
            },
        );
        apply_op(
            &mut s,
            &SheetOp::AddSheet {
                name: "Sheet2".into(),
            },
        );
        apply_op(
            &mut s,
            &SheetOp::SetCell {
                addr: "A1".into(),
                value: "on2".into(),
                formula: String::new(),
            },
        );
        assert_eq!(s.cells["A1"].value, "on2");
        apply_op(&mut s, &SheetOp::SwitchSheet { index: 0 });
        assert_eq!(s.cells["A1"].value, "on1");
    }

    #[test]
    fn protect_blocks_set_cell() {
        let mut s = EratSheet::empty();
        apply_op(
            &mut s,
            &SheetOp::ProtectSheet {
                protected: true,
            },
        );
        assert!(s.is_active_protected());
        let op = SheetOp::SetCell {
            addr: "A1".into(),
            value: "x".into(),
            formula: String::new(),
        };
        assert!(blocked_by_protect(&s, &op));
        apply_op(
            &mut s,
            &SheetOp::ProtectSheet {
                protected: false,
            },
        );
        assert!(!blocked_by_protect(&s, &op));
    }

    #[test]
    fn freeze_panes_allowed_when_protected() {
        let mut s = EratSheet::empty();
        apply_op(
            &mut s,
            &SheetOp::ProtectSheet {
                protected: true,
            },
        );
        let op = SheetOp::FreezePanes { rows: 2, cols: 1 };
        assert!(!blocked_by_protect(&s, &op));
        apply_op(&mut s, &op);
        assert_eq!(s.active_freeze(), (2, 1));
    }

    #[test]
    fn protected_range_blocks_cell_when_sheet_open() {
        let mut s = EratSheet::empty();
        apply_op(
            &mut s,
            &SheetOp::SetProtectedRanges {
                ranges: vec!["A1:B2".into()],
            },
        );
        assert!(!s.is_active_protected());
        assert!(blocked_by_protect(
            &s,
            &SheetOp::SetCell {
                addr: "A1".into(),
                value: "x".into(),
                formula: String::new(),
            }
        ));
        assert!(!blocked_by_protect(
            &s,
            &SheetOp::SetCell {
                addr: "C3".into(),
                value: "ok".into(),
                formula: String::new(),
            }
        ));
    }

    #[test]
    fn set_merges_persists_on_tab() {
        let mut s = EratSheet::empty();
        apply_op(
            &mut s,
            &SheetOp::SetMerges {
                merges: vec!["A1:B1".into()],
            },
        );
        assert_eq!(s.active_merges(), vec!["A1:B1".to_string()]);
    }

    #[test]
    fn sort_range_moves_whole_rows() {
        let mut s = EratSheet::empty();
        apply_op(
            &mut s,
            &SheetOp::SetCell {
                addr: "A1".into(),
                value: "b".into(),
                formula: String::new(),
            },
        );
        apply_op(
            &mut s,
            &SheetOp::SetCell {
                addr: "B1".into(),
                value: "row-b".into(),
                formula: String::new(),
            },
        );
        apply_op(
            &mut s,
            &SheetOp::SetCell {
                addr: "A2".into(),
                value: "a".into(),
                formula: String::new(),
            },
        );
        apply_op(
            &mut s,
            &SheetOp::SetCell {
                addr: "B2".into(),
                value: "row-a".into(),
                formula: String::new(),
            },
        );
        apply_op(
            &mut s,
            &SheetOp::SortRange {
                col: 1,
                ascending: true,
            },
        );
        assert_eq!(s.cells["A1"].value, "a");
        assert_eq!(s.cells["B1"].value, "row-a");
        assert_eq!(s.cells["A2"].value, "b");
        assert_eq!(s.cells["B2"].value, "row-b");
    }

    #[test]
    fn set_filter_criteria_and_charts_persist_on_tab() {
        let mut s = EratSheet::empty();
        apply_op(
            &mut s,
            &SheetOp::SetFilterCriteria {
                criteria: Some(serde_json::json!({"col":0,"mode":"contains","value":"x"})),
            },
        );
        apply_op(
            &mut s,
            &SheetOp::SetCharts {
                charts: vec![crate::model::ChartSpec {
                    chart_type: "bar".into(),
                    range: "A1:A3".into(),
                    title: String::new(),
                }],
            },
        );
        let tab = &s.sheets[s.active_sheet];
        assert!(tab.filter_criteria.is_some());
        assert_eq!(tab.charts.len(), 1);
        assert_eq!(tab.charts[0].range, "A1:A3");
    }
}
