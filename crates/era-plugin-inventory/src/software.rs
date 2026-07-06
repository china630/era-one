//! Сбор установленного ПО (platform-specific, ADR-0011 §5a).

use era_plugin_sdk::SoftwareEntry;
use std::path::Path;

pub fn collect_software(platform: &str) -> Vec<SoftwareEntry> {
    match platform {
        "linux" => collect_linux(),
        "macos" => collect_macos(),
        "windows" => collect_windows(),
        _ => Vec::new(),
    }
}

fn collect_linux() -> Vec<SoftwareEntry> {
    let mut out = Vec::new();
    out.extend(parse_dpkg_status(Path::new("/var/lib/dpkg/status")));
    if out.is_empty() {
        out.extend(parse_rpm_query());
    }
    out
}

fn parse_dpkg_status(path: &Path) -> Vec<SoftwareEntry> {
    let Ok(text) = std::fs::read_to_string(path) else {
        return Vec::new();
    };
    let mut out = Vec::new();
    let mut name = String::new();
    let mut version = String::new();
    for line in text.lines() {
        if line.is_empty() {
            if !name.is_empty() {
                out.push(SoftwareEntry {
                    name: name.clone(),
                    version: version.clone(),
                    vendor: String::new(),
                    source: "dpkg".into(),
                });
            }
            name.clear();
            version.clear();
            continue;
        }
        if let Some(v) = line.strip_prefix("Package: ") {
            name = v.trim().to_string();
        } else if let Some(v) = line.strip_prefix("Version: ") {
            version = v.trim().to_string();
        }
    }
    out
}

fn parse_rpm_query() -> Vec<SoftwareEntry> {
    use std::process::Command;
    let output = Command::new("rpm").args(["-qa", "--qf", "%{NAME}\t%{VERSION}\n"]).output();
    let Ok(output) = output else {
        return Vec::new();
    };
    if !output.status.success() {
        return Vec::new();
    }
    let mut out = Vec::new();
    for line in String::from_utf8_lossy(&output.stdout).lines() {
        let Some((name, version)) = line.split_once('\t') else {
            continue;
        };
        out.push(SoftwareEntry {
            name: name.to_string(),
            version: version.to_string(),
            vendor: String::new(),
            source: "rpm".into(),
        });
    }
    out
}

fn collect_macos() -> Vec<SoftwareEntry> {
    use std::process::Command;
    let output = Command::new("brew").args(["list", "--versions"]).output();
    let Ok(output) = output else {
        return Vec::new();
    };
    if !output.status.success() {
        return Vec::new();
    }
    let mut out = Vec::new();
    for line in String::from_utf8_lossy(&output.stdout).lines() {
        let mut parts = line.split_whitespace();
        let Some(name) = parts.next() else { continue };
        let version = parts.next().unwrap_or("").to_string();
        out.push(SoftwareEntry {
            name: name.to_string(),
            version,
            vendor: String::new(),
            source: "brew".into(),
        });
    }
    out
}

fn collect_windows() -> Vec<SoftwareEntry> {
  // Registry scan — stub для CI; полный сбор в field-deploy.
  Vec::new()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn dpkg_parser_reads_fixture() {
        let path = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
            .join("testdata/dpkg_status_sample");
        let entries = parse_dpkg_status(&path);
        assert!(!entries.is_empty());
        assert_eq!(entries[0].source, "dpkg");
    }
}
