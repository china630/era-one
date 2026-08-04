use crate::model::EratSheet;

fn col_row(addr: &str) -> Option<(u32, u32)> {
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

fn cell_num(sheet: &EratSheet, addr: &str) -> f64 {
    sheet
        .cells
        .get(&addr.to_uppercase())
        .and_then(|c| c.value.parse().ok())
        .unwrap_or(0.0)
}

fn cell_raw(sheet: &EratSheet, addr: &str) -> String {
    sheet
        .cells
        .get(&addr.to_uppercase())
        .map(|c| c.value.clone())
        .unwrap_or_default()
}

fn expand_range(range: &str) -> Vec<String> {
    let parts: Vec<_> = range.split(':').collect();
    if parts.len() == 1 {
        return vec![parts[0].to_uppercase()];
    }
    let (c1, r1) = match col_row(parts[0]) {
        Some(v) => v,
        None => return vec![],
    };
    let (c2, r2) = match col_row(parts[1]) {
        Some(v) => v,
        None => return vec![],
    };
    let mut out = Vec::new();
    for c in c1.min(c2)..=c1.max(c2) {
        for r in r1.min(r2)..=r1.max(r2) {
            out.push(addr_of(c, r));
        }
    }
    out
}

/// Split top-level commas (respect nested parentheses).
fn split_args(s: &str) -> Vec<String> {
    let mut out = Vec::new();
    let mut cur = String::new();
    let mut depth = 0i32;
    let mut in_str = false;
    for ch in s.chars() {
        match ch {
            '"' if !in_str => in_str = true,
            '"' if in_str => in_str = false,
            '(' if !in_str => {
                depth += 1;
                cur.push(ch);
            }
            ')' if !in_str => {
                depth -= 1;
                cur.push(ch);
            }
            ',' if !in_str && depth == 0 => {
                out.push(cur.trim().to_string());
                cur.clear();
            }
            _ => cur.push(ch),
        }
    }
    if !cur.trim().is_empty() {
        out.push(cur.trim().to_string());
    }
    out
}

fn eval_atom(sheet: &EratSheet, atom: &str) -> String {
    let a = atom.trim();
    if a.is_empty() {
        return String::new();
    }
    if (a.starts_with('"') && a.ends_with('"')) || (a.starts_with('\'') && a.ends_with('\'')) {
        return a[1..a.len() - 1].to_string();
    }
    if let Ok(n) = a.parse::<f64>() {
        return format_num(n);
    }
    if col_row(a).is_some() {
        return cell_raw(sheet, a);
    }
    a.to_string()
}

fn eval_cond(sheet: &EratSheet, cond: &str) -> bool {
    let c = cond.trim();
    for op in [">=", "<=", "<>", "!=", "=", ">", "<"] {
        if let Some(i) = c.find(op) {
            let left = eval_atom(sheet, &c[..i]);
            let right = eval_atom(sheet, &c[i + op.len()..]);
            let ln = left.parse::<f64>().ok();
            let rn = right.parse::<f64>().ok();
            return match (ln, rn, op) {
                (Some(l), Some(r), ">") => l > r,
                (Some(l), Some(r), "<") => l < r,
                (Some(l), Some(r), ">=") => l >= r,
                (Some(l), Some(r), "<=") => l <= r,
                (Some(l), Some(r), "=") => (l - r).abs() < 1e-9,
                (Some(l), Some(r), "<>") | (Some(l), Some(r), "!=") => (l - r).abs() >= 1e-9,
                (_, _, "=") => left == right,
                (_, _, "<>") | (_, _, "!=") => left != right,
                _ => false,
            };
        }
    }
    let v = eval_atom(sheet, c);
    if let Ok(n) = v.parse::<f64>() {
        return n != 0.0;
    }
    !v.is_empty()
}

fn format_num(v: f64) -> String {
    if (v - v.round()).abs() < 1e-9 {
        format!("{}", v as i64)
    } else {
        format!("{v}")
    }
}

fn countif_match(value: &str, criteria: &str) -> bool {
    let c = criteria.trim();
    for op in [">=", "<=", "<>", "!=", "=", ">", "<"] {
        if let Some(rest) = c.strip_prefix(op) {
            let right = rest.trim().trim_matches('"').trim_matches('\'');
            let ln = value.parse::<f64>().ok();
            let rn = right.parse::<f64>().ok();
            return match (ln, rn, op) {
                (Some(l), Some(r), ">") => l > r,
                (Some(l), Some(r), "<") => l < r,
                (Some(l), Some(r), ">=") => l >= r,
                (Some(l), Some(r), "<=") => l <= r,
                (Some(l), Some(r), "=") => (l - r).abs() < 1e-9,
                (Some(l), Some(r), "<>") | (Some(l), Some(r), "!=") => (l - r).abs() >= 1e-9,
                (_, _, "=") => value == right,
                (_, _, "<>") | (_, _, "!=") => value != right,
                _ => false,
            };
        }
    }
    value == c || value.eq_ignore_ascii_case(c)
}

fn eval_formula(sheet: &EratSheet, formula: &str) -> String {
    let f = formula.trim().trim_start_matches('=');
    let fu = f.to_uppercase();
    let (name, args_raw) = match fu.find('(') {
        Some(i) => (&fu[..i], f[i + 1..].trim_end_matches(')')),
        None => return String::new(),
    };
    let name = name.trim();

    if name == "IF" {
        let args = split_args(args_raw);
        if args.len() < 2 {
            return String::new();
        }
        let then_v = args.get(1).map(|s| s.as_str()).unwrap_or("");
        let else_v = args.get(2).map(|s| s.as_str()).unwrap_or("");
        return if eval_cond(sheet, &args[0]) {
            eval_atom(sheet, then_v)
        } else {
            eval_atom(sheet, else_v)
        };
    }

    if name == "COUNT" {
        let addrs = expand_range(args_raw.trim());
        let n = addrs
            .iter()
            .filter(|a| {
                let raw = cell_raw(sheet, a);
                !raw.is_empty() && raw.parse::<f64>().is_ok()
            })
            .count();
        return format_num(n as f64);
    }

    if name == "COUNTIF" {
        let args = split_args(args_raw);
        if args.len() < 2 {
            return String::new();
        }
        let addrs = expand_range(args[0].trim());
        let crit = args[1].trim().trim_matches('"').trim_matches('\'');
        let n = addrs
            .iter()
            .filter(|a| countif_match(&cell_raw(sheet, a), crit))
            .count();
        return format_num(n as f64);
    }

    // O-FMT-2: ROUND(number, digits)
    if name == "ROUND" {
        let args = split_args(args_raw);
        if args.is_empty() {
            return String::new();
        }
        let n = cell_num(sheet, args[0].trim());
        let digits = args
            .get(1)
            .and_then(|s| s.trim().parse::<i32>().ok())
            .unwrap_or(0)
            .clamp(0, 10) as i32;
        let factor = 10f64.powi(digits);
        return format_num((n * factor).round() / factor);
    }

    let addrs = expand_range(args_raw.trim());
    let nums: Vec<f64> = addrs.iter().map(|a| cell_num(sheet, a)).collect();
    if nums.is_empty() {
        return "0".into();
    }
    let v = match name {
        "SUM" => nums.iter().sum(),
        "AVERAGE" | "AVG" => nums.iter().sum::<f64>() / nums.len() as f64,
        "MIN" => nums.iter().cloned().fold(f64::INFINITY, f64::min),
        "MAX" => nums.iter().cloned().fold(f64::NEG_INFINITY, f64::max),
        _ => return String::new(),
    };
    format_num(v)
}

/// Recalculate all formula cells (one pass; MVP dependency order = map order).
pub fn recalc(sheet: &mut EratSheet) {
    let formulas: Vec<(String, String)> = sheet
        .cells
        .iter()
        .filter(|(_, c)| !c.formula.is_empty())
        .map(|(a, c)| (a.clone(), c.formula.clone()))
        .collect();
    for (addr, formula) in formulas {
        let val = eval_formula(sheet, &formula);
        if let Some(cell) = sheet.cells.get_mut(&addr) {
            cell.value = val;
        }
    }
}

#[cfg(test)]
mod calc_tests {
    use super::*;

    #[test]
    fn calc_sum_range_recalc() {
        let mut s = EratSheet::empty();
        s.set_cell("A1", "10", "");
        s.set_cell("A2", "20", "");
        s.set_cell("A3", "", "=SUM(A1:A2)");
        recalc(&mut s);
        assert_eq!(s.cells["A3"].value, "30");
    }

    #[test]
    fn calc_average_min_max() {
        let mut s = EratSheet::empty();
        s.set_cell("B1", "2", "");
        s.set_cell("B2", "4", "");
        s.set_cell("B3", "", "=AVERAGE(B1:B2)");
        s.set_cell("B4", "", "=MIN(B1:B2)");
        s.set_cell("B5", "", "=MAX(B1:B2)");
        recalc(&mut s);
        assert_eq!(s.cells["B3"].value, "3");
        assert_eq!(s.cells["B4"].value, "2");
        assert_eq!(s.cells["B5"].value, "4");
    }

    #[test]
    fn avg_min_max_round() {
        let mut s = EratSheet::empty();
        s.set_cell("C1", "2", "");
        s.set_cell("C2", "4", "");
        s.set_cell("C3", "", "=AVERAGE(C1:C2)");
        s.set_cell("C4", "", "=MIN(C1:C2)");
        s.set_cell("C5", "", "=MAX(C1:C2)");
        s.set_cell("C6", "3.14159", "");
        s.set_cell("C7", "", "=ROUND(C6,2)");
        recalc(&mut s);
        assert_eq!(s.cells["C3"].value, "3");
        assert_eq!(s.cells["C4"].value, "2");
        assert_eq!(s.cells["C5"].value, "4");
        assert_eq!(s.cells["C7"].value, "3.14");
    }

    #[test]
    fn calc_count_and_if() {
        let mut s = EratSheet::empty();
        s.set_cell("A1", "10", "");
        s.set_cell("A2", "x", "");
        s.set_cell("A3", "20", "");
        s.set_cell("B1", "", "=COUNT(A1:A3)");
        s.set_cell("B2", "", "=IF(A1>5,\"yes\",\"no\")");
        s.set_cell("B3", "", "=IF(A1<5,\"yes\",\"no\")");
        recalc(&mut s);
        assert_eq!(s.cells["B1"].value, "2");
        assert_eq!(s.cells["B2"].value, "yes");
        assert_eq!(s.cells["B3"].value, "no");
    }

    #[test]
    fn calc_countif() {
        let mut s = EratSheet::empty();
        s.set_cell("A1", "10", "");
        s.set_cell("A2", "20", "");
        s.set_cell("A3", "10", "");
        s.set_cell("B1", "", "=COUNTIF(A1:A3,10)");
        s.set_cell("B2", "", "=COUNTIF(A1:A3,\">15\")");
        recalc(&mut s);
        assert_eq!(s.cells["B1"].value, "2");
        assert_eq!(s.cells["B2"].value, "1");
    }
}
