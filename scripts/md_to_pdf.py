#!/usr/bin/env python3
"""Конвертер Markdown -> PDF для distributor-документов ERA.

Подход без внешних бинарей кроме штатного Microsoft Edge (Windows):
  Markdown --(python-markdown)--> styled HTML --(Edge headless)--> PDF.

Использование:
  python scripts/md_to_pdf.py <input.md> [output.pdf]

Air-gap: не делает сетевых вызовов; использует локальный Edge и локальные шрифты.
"""
from __future__ import annotations

import os
import shutil
import subprocess
import sys
import tempfile

import markdown

CSS = """
@page { size: A4; margin: 16mm 14mm; }
* { box-sizing: border-box; }
body {
  font-family: "Segoe UI", "Segoe UI Emoji", Arial, sans-serif;
  font-size: 10.5pt; line-height: 1.45; color: #1f2933; margin: 0;
}
h1 { font-size: 20pt; color: #0b3d5c; border-bottom: 3px solid #0b3d5c;
     padding-bottom: 6px; margin: 0 0 10px; }
h2 { font-size: 14pt; color: #0b3d5c; margin: 18px 0 8px;
     border-left: 4px solid #2e8bc0; padding-left: 8px; }
h3 { font-size: 11.5pt; color: #145374; margin: 12px 0 4px; }
p, li { margin: 4px 0; }
code { background: #eef2f5; padding: 1px 4px; border-radius: 3px;
       font-family: "Cascadia Code", Consolas, monospace; font-size: 9.5pt; }
pre { background: #0f1b2b; color: #e6edf3; padding: 10px 12px; border-radius: 6px;
      overflow-x: auto; font-size: 9pt; }
pre code { background: transparent; color: inherit; padding: 0; }
blockquote { margin: 8px 0; padding: 8px 12px; background: #f3f8fb;
             border-left: 4px solid #2e8bc0; color: #29475a; }
table { border-collapse: collapse; width: 100%; margin: 8px 0; font-size: 9.5pt; }
th, td { border: 1px solid #c7d2da; padding: 5px 8px; text-align: left;
         vertical-align: top; }
th { background: #0b3d5c; color: #fff; font-weight: 600; }
tr:nth-child(even) td { background: #f4f8fb; }
hr { border: none; border-top: 1px solid #d4dde3; margin: 14px 0; }
strong { color: #0b3d5c; }
a { color: #1c6ea4; text-decoration: none; }
"""

HTML_TMPL = """<!doctype html>
<html lang="ru"><head><meta charset="utf-8">
<title>{title}</title><style>{css}</style></head>
<body>{body}</body></html>"""


def find_edge() -> str:
    candidates = [
        r"C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe",
        r"C:\Program Files\Microsoft\Edge\Application\msedge.exe",
    ]
    for c in candidates:
        if os.path.exists(c):
            return c
    found = shutil.which("msedge") or shutil.which("chrome")
    if found:
        return found
    raise SystemExit("Не найден Microsoft Edge/Chrome для печати в PDF")


def render(md_path: str, pdf_path: str) -> None:
    with open(md_path, encoding="utf-8") as f:
        text = f.read()
    body = markdown.markdown(
        text,
        extensions=["tables", "fenced_code", "sane_lists", "toc"],
    )
    title = os.path.splitext(os.path.basename(md_path))[0]
    html = HTML_TMPL.format(title=title, css=CSS, body=body)

    with tempfile.NamedTemporaryFile(
        "w", suffix=".html", delete=False, encoding="utf-8"
    ) as tmp:
        tmp.write(html)
        html_path = tmp.name

    try:
        edge = find_edge()
        uri = "file:///" + html_path.replace("\\", "/")
        subprocess.run(
            [
                edge,
                "--headless",
                "--disable-gpu",
                "--no-pdf-header-footer",
                f"--print-to-pdf={pdf_path}",
                uri,
            ],
            check=True,
            timeout=120,
        )
    finally:
        try:
            os.remove(html_path)
        except OSError:
            pass

    if not os.path.exists(pdf_path):
        raise SystemExit("PDF не создан — проверьте Edge headless")
    print(f"OK: {pdf_path} ({os.path.getsize(pdf_path)} bytes)")


def main() -> None:
    if len(sys.argv) < 2:
        raise SystemExit("usage: python scripts/md_to_pdf.py <input.md> [output.pdf]")
    md_path = sys.argv[1]
    if len(sys.argv) >= 3:
        pdf_path = sys.argv[2]
    else:
        pdf_path = os.path.splitext(md_path)[0] + ".pdf"
    render(md_path, os.path.abspath(pdf_path))


if __name__ == "__main__":
    main()
