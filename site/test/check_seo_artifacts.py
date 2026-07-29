#!/usr/bin/env python3
"""Gate: SEO artifacts present in dist/site after site_seo_enrich.py.
Usage: python3 site/test/check_seo_artifacts.py [dist/site]
"""
from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]


def fail(msg: str) -> None:
    print(f"FAIL: {msg}", file=sys.stderr)
    raise SystemExit(1)


def main() -> None:
    out = Path(sys.argv[1] if len(sys.argv) > 1 else ROOT / "dist" / "site")
    if not out.is_dir():
        fail(f"dist not found: {out}")

    for name in ("robots.txt", "sitemap.xml", "favicon.svg", "404.html"):
        if not (out / name).is_file():
            fail(f"missing {name}")

    robots = (out / "robots.txt").read_text(encoding="utf-8")
    if "Sitemap:" not in robots or "Disallow: /datasheets/" not in robots:
        fail("robots.txt incomplete")

    sm = (out / "sitemap.xml").read_text(encoding="utf-8")
    for needle in (
        "https://www.era-one.solutions/",
        "control.html",
        "editions/era-core.html",
    ):
        if needle not in sm:
            fail(f"sitemap missing {needle}")

    home = (out / "index.html").read_text(encoding="utf-8")
    if 'rel="canonical"' not in home:
        fail("index.html missing canonical")
    if "application/ld+json" not in home or "Organization" not in home:
        fail("index.html missing Organization JSON-LD")
    if "<h1" not in home.lower():
        fail("index.html missing H1")

    control = (out / "control.html").read_text(encoding="utf-8")
    if 'data-prerendered="1"' not in control:
        fail("control.html not prerendered")
    if 'rel="canonical"' not in control:
        fail("control.html missing canonical")
    if re.search(r"<h1[^>]*>\s*ERA Control\s*</h1>", control, flags=re.I) is None:
        fail("control.html missing page H1")
    if "ds-loading" in control:
        fail("control.html still has Loading placeholder")

    edition = out / "editions" / "era-core.html"
    if not edition.is_file():
        fail("editions/era-core.html missing")
    ed = edition.read_text(encoding="utf-8")
    if 'rel="canonical"' not in ed or "ERA Core" not in ed:
        fail("edition page incomplete")
    if "SoftwareApplication" not in ed:
        fail("edition missing SoftwareApplication schema")

    stub = out / "control" / "index.html"
    if not stub.is_file():
        fail("clean URL stub control/index.html missing")

    sample_ds = out / "datasheets" / "en" / "01-ERA-Core.html"
    if sample_ds.is_file():
        ds = sample_ds.read_text(encoding="utf-8")
        if "noindex" not in ds:
            fail("datasheet missing noindex")

    print("OK: SEO artifacts check passed")


if __name__ == "__main__":
    main()
