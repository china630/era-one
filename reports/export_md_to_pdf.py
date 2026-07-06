# -*- coding: utf-8 -*-
"""Markdown -> HTML -> PDF via Chrome/Edge headless. Stdlib only; output on D: to avoid full C: drive."""
from __future__ import annotations

import html
import re
import subprocess
import sys
from pathlib import Path


def inline_md(s: str) -> str:
    s = html.escape(s)
    s = re.sub(r"\*\*(.+?)\*\*", r"<strong>\1</strong>", s)
    s = re.sub(r"`([^`]+)`", r"<code>\1</code>", s)
    s = re.sub(r"\[([^\]]+)\]\(([^)]+)\)", r'<a href="\2">\1</a>', s)
    return s


def md_to_html(md: str) -> str:
    lines = md.replace("\r\n", "\n").split("\n")
    out: list[str] = []
    i = 0
    in_ul = False
    in_ol = False
    in_code = False
    code_lines: list[str] = []

    def close_lists():
        nonlocal in_ul, in_ol
        if in_ul:
            out.append("</ul>")
            in_ul = False
        if in_ol:
            out.append("</ol>")
            in_ol = False

    while i < len(lines):
        raw = lines[i]
        line = raw.rstrip()

        if in_code:
            if line.strip() == "```":
                block = html.escape("\n".join(code_lines))
                out.append(f'<pre class="code">{block}</pre>')
                code_lines = []
                in_code = False
            else:
                code_lines.append(raw)
            i += 1
            continue

        if line.strip() == "```":
            close_lists()
            in_code = True
            i += 1
            continue

        if not line.strip():
            close_lists()
            i += 1
            continue

        if line.strip() == "---":
            close_lists()
            out.append("<hr>")
            i += 1
            continue

        if line.startswith("|") and "|" in line[1:]:
            close_lists()
            rows: list[list[str]] = []
            while i < len(lines) and lines[i].strip().startswith("|"):
                row = [c.strip() for c in lines[i].strip().strip("|").split("|")]
                rows.append(row)
                i += 1
            if len(rows) >= 2 and re.match(r"^[\s\-:|]+$", "|".join(rows[1])):
                header, _sep, body = rows[0], rows[1], rows[2:]
                out.append("<table>")
                out.append("<thead><tr>")
                for c in header:
                    out.append(f"<th>{inline_md(c)}</th>")
                out.append("</tr></thead><tbody>")
                for row in body:
                    out.append("<tr>")
                    for c in row:
                        out.append(f"<td>{inline_md(c)}</td>")
                    out.append("</tr>")
                out.append("</tbody></table>")
            else:
                out.append("<table><tbody>")
                for row in rows:
                    out.append("<tr>")
                    for c in row:
                        out.append(f"<td>{inline_md(c)}</td>")
                    out.append("</tr>")
                out.append("</tbody></table>")
            continue

        if re.match(r"^#\s+", line):
            close_lists()
            level = len(line) - len(line.lstrip("#"))
            text = line.lstrip("#").strip()
            out.append(f"<h{min(level, 6)}>{inline_md(text)}</h{min(level, 6)}>")
            i += 1
            continue

        if re.match(r"^-\s+", line):
            if not in_ul:
                close_lists()
                out.append("<ul>")
                in_ul = True
            item = re.sub(r"^-\s+", "", line)
            out.append(f"<li>{inline_md(item)}</li>")
            i += 1
            continue

        if re.match(r"^\d+\.\s+", line):
            if not in_ol:
                close_lists()
                out.append("<ol>")
                in_ol = True
            item = re.sub(r"^\d+\.\s+", "", line)
            out.append(f"<li>{inline_md(item)}</li>")
            i += 1
            continue

        close_lists()
        out.append(f"<p>{inline_md(line)}</p>")
        i += 1

    close_lists()
    body = "\n".join(out)
    return f"""<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="utf-8"/>
<title>AI-Donors Matrix Report</title>
<style>
  body {{ font-family: "Segoe UI", Arial, sans-serif; font-size: 11pt; line-height: 1.45; margin: 2cm; color: #111; }}
  h1 {{ font-size: 18pt; border-bottom: 1px solid #ccc; padding-bottom: 0.3em; }}
  h2 {{ font-size: 14pt; margin-top: 1.2em; }}
  h3 {{ font-size: 12pt; margin-top: 1em; }}
  table {{ border-collapse: collapse; width: 100%; margin: 0.8em 0; font-size: 10pt; }}
  th, td {{ border: 1px solid #999; padding: 6px 8px; vertical-align: top; }}
  th {{ background: #f0f0f0; }}
  pre.code {{ background: #f5f5f5; border: 1px solid #ddd; padding: 10px; font-size: 9pt; white-space: pre-wrap; }}
  code {{ font-family: Consolas, monospace; font-size: 0.95em; }}
  hr {{ border: none; border-top: 1px solid #ccc; margin: 1.2em 0; }}
  a {{ color: #0645ad; }}
  ul, ol {{ margin: 0.5em 0 0.5em 1.2em; }}
</style>
</head>
<body>
{body}
</body>
</html>
"""


def find_browser() -> Path | None:
    candidates = [
        Path(r"C:\Program Files\Google\Chrome\Application\chrome.exe"),
        Path(r"C:\Program Files (x86)\Google\Chrome\Application\chrome.exe"),
        Path(r"C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe"),
        Path(r"C:\Program Files\Microsoft\Edge\Application\msedge.exe"),
    ]
    for p in candidates:
        if p.is_file():
            return p
    return None


def main() -> int:
    base = Path(__file__).resolve().parent
    md_path = base / "AI-Donors-Matrix-Deep-Analysis.md"
    html_path = base / "AI-Donors-Matrix-Deep-Analysis.print.html"
    pdf_path = base / "AI-Donors-Matrix-Deep-Analysis.pdf"

    if not md_path.is_file():
        print("MD not found:", md_path, file=sys.stderr)
        return 1

    md_text = md_path.read_text(encoding="utf-8")
    html_path.write_text(md_to_html(md_text), encoding="utf-8")

    browser = find_browser()
    if not browser:
        print("Chrome/Edge not found.", file=sys.stderr)
        return 1

    url = html_path.as_uri()
    cmd = [
        str(browser),
        "--headless=new",
        "--disable-gpu",
        "--no-first-run",
        f"--print-to-pdf={pdf_path}",
        "--no-pdf-header-footer",
        url,
    ]
    r = subprocess.run(cmd, capture_output=True, text=True, timeout=120)
    if r.returncode != 0 or not pdf_path.is_file():
        print(r.stderr or r.stdout, file=sys.stderr)
        return r.returncode or 1

    print("OK:", pdf_path)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
