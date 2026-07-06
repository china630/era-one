#!/usr/bin/env python3
"""Markdown прайса -> HTML (дизайн датащитов) -> PDF (Edge headless).

Использует тот же shell/логотип/CSS, что и датащиты (generate_distributor_docs.shell +
assets/datasheet-common.css), чтобы прайс визуально совпадал с остальной линейкой.

Air-gap: без сети; локальный Edge + локальные CSS/логотип.
Использование:
  python scripts/pricing_to_pdf.py                          # ERA-Pricing.md (внутренний)
  python scripts/pricing_to_pdf.py docs/distributor/ERA-Pricing-Client.md
"""
from __future__ import annotations

import sys
from pathlib import Path

import markdown

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "scripts"))

from generate_distributor_docs import shell, LOGO_MAIN  # noqa: E402
from html_to_pdf import html_to_pdf  # noqa: E402

DIST = ROOT / "docs" / "distributor"
DEFAULT_MD = DIST / "ERA-Pricing.md"

# Датащит-CSS стилизует table.cmp/h1/h2/h3/note-int, но не generic p/ul/code —
# добавляем недостающее в том же масштабе, чтобы прайс смотрелся как датащит.
STYLE_EXTRA = """<style>
.body p { font-size: 9.8pt; }
.body ul, .body ol { margin: 4px 0 8px 18px; font-size: 9.6pt; }
.body li { margin: 3px 0; }
.body a { color: var(--teal-d); text-decoration: none; }
.body code {
  background: #eef2f5; padding: 1px 4px; border-radius: 3px;
  font-family: Consolas, "Cascadia Code", monospace; font-size: 8.6pt; color: #0b2a3a;
}
.body table.cmp td:first-child { white-space: normal; }
.body hr { border: none; border-top: 1px solid var(--line); margin: 12px 0; }
</style>"""


def md_to_body(md_text: str) -> str:
    html = markdown.markdown(
        md_text,
        extensions=["tables", "fenced_code", "sane_lists"],
    )
    html = html.replace("<table>", '<table class="cmp">')
    html = html.replace("<blockquote>", '<div class="note-int">')
    html = html.replace("</blockquote>", "</div>")
    return f'  <div class="body">\n{STYLE_EXTRA}\n{html}\n  </div>'


def build(md_path: Path) -> None:
    if not md_path.is_file():
        raise SystemExit(f"Нет файла: {md_path}")
    html_path = md_path.with_suffix(".html")
    pdf_path = md_path.with_suffix(".pdf")
    # Заголовок = первый H1 markdown (если есть), иначе имя файла.
    first_line = md_path.read_text(encoding="utf-8").splitlines()[0].lstrip("# ").strip()
    title = first_line or f"ERA One — {md_path.stem}"
    body = md_to_body(md_path.read_text(encoding="utf-8"))
    full = shell(title, body, LOGO_MAIN, "assets/datasheet-common.css")
    html_path.write_text(full, encoding="utf-8")
    print(f"OK: {html_path}")
    html_to_pdf(str(html_path), str(pdf_path))


def main() -> None:
    md = Path(sys.argv[1]) if len(sys.argv) > 1 else DEFAULT_MD
    build(md)


if __name__ == "__main__":
    main()
