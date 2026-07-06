#!/usr/bin/env python3
"""HTML -> PDF через headless Edge/Chrome (Windows).

Использование:
  python scripts/html_to_pdf.py <input.html> [output.pdf]
  python scripts/html_to_pdf.py --all   # все датащиты в docs/distributor/

Air-gap: без сетевых вызовов.
"""
from __future__ import annotations

import argparse
import os
import shutil
import subprocess
import sys

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
DIST = os.path.join(ROOT, "docs", "distributor")


def find_edge() -> str:
    candidates = [
        r"C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe",
        r"C:\Program Files\Microsoft\Edge\Application\msedge.exe",
    ]
    for c in candidates:
        if os.path.exists(c):
            return c
    found = shutil.which("msedge") or shutil.which("chrome") or shutil.which("chromium")
    if found:
        return found
    raise SystemExit("Не найден Microsoft Edge/Chrome для печати в PDF")


def html_to_pdf(html_path: str, pdf_path: str) -> None:
    html_path = os.path.abspath(html_path)
    pdf_path = os.path.abspath(pdf_path)
    os.makedirs(os.path.dirname(pdf_path), exist_ok=True)
    tmp_path = pdf_path + ".tmp.pdf"
    if os.path.exists(tmp_path):
        os.remove(tmp_path)
    mtime = int(os.path.getmtime(html_path))
    uri = "file:///" + html_path.replace("\\", "/") + f"?v={mtime}"
    edge = find_edge()
    subprocess.run(
        [
            edge,
            "--headless",
            "--disable-gpu",
            "--no-pdf-header-footer",
            "--run-all-compositor-stages-before-draw",
            f"--print-to-pdf={tmp_path}",
            uri,
        ],
        check=True,
        timeout=120,
    )
    if not os.path.exists(tmp_path):
        raise SystemExit(f"PDF не создан: {tmp_path}")
    try:
        if os.path.exists(pdf_path):
            os.remove(pdf_path)
        os.replace(tmp_path, pdf_path)
    except OSError as e:
        fallback = pdf_path.replace(".pdf", "-new.pdf")
        os.replace(tmp_path, fallback)
        print(f"WARN: {pdf_path} занят — сохранено как {fallback} ({e})")
        pdf_path = fallback
    print(f"OK: {pdf_path} ({os.path.getsize(pdf_path)} bytes)")


def collect_html_files() -> list[str]:
    out: list[str] = []
    main = os.path.join(DIST, "ERA-One-DataSheet.html")
    if os.path.isfile(main):
        out.append(main)
    pl = os.path.join(DIST, "ERA-Product-Line.html")
    if os.path.isfile(pl):
        out.append(pl)
    hyb = os.path.join(DIST, "ERA-Sovereign-Hybrid-Internal.html")
    if os.path.isfile(hyb):
        out.append(hyb)
    for sub in ("datasheets", "head-to-head"):
        d = os.path.join(DIST, sub)
        if not os.path.isdir(d):
            continue
        for name in sorted(os.listdir(d)):
            if name.endswith(".html"):
                out.append(os.path.join(d, name))
    return out


def main() -> None:
    parser = argparse.ArgumentParser(description="HTML to PDF (Edge headless)")
    parser.add_argument("input", nargs="?", help="input.html")
    parser.add_argument("output", nargs="?", help="output.pdf")
    parser.add_argument("--all", action="store_true", help="собрать все датащиты")
    args = parser.parse_args()

    if args.all:
        files = collect_html_files()
        if not files:
            raise SystemExit("HTML-файлы не найдены в docs/distributor/")
        for html in files:
            pdf = os.path.splitext(html)[0] + ".pdf"
            html_to_pdf(html, pdf)
        print(f"Собрано PDF: {len(files)}")
        return

    if not args.input:
        parser.print_help()
        raise SystemExit(1)

    pdf = args.output or os.path.splitext(args.input)[0] + ".pdf"
    html_to_pdf(args.input, pdf)


if __name__ == "__main__":
    main()
