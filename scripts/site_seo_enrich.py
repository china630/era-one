#!/usr/bin/env python3
"""Enrich dist/site/ with SEO artifacts: robots, sitemap, canonical/OG/schema,
prerendered family & edition pages, noindex datasheets, 404, clean-URL stubs.
Usage: python3 scripts/site_seo_enrich.py [dist/site]
"""
from __future__ import annotations

import html as html_lib
import json
import re
import shutil
import sys
from pathlib import Path
from xml.sax.saxutils import escape as xml_escape

ROOT = Path(__file__).resolve().parents[1]
CANON = "https://www.era-one.solutions"
OG_IMAGE = f"{CANON}/assets/og-default.png"
ORG = {
    "@context": "https://schema.org",
    "@type": "Organization",
    "name": "ERA One",
    "url": CANON,
    "logo": f"{CANON}/assets/era-one-logo.svg",
    "email": "sales@era-one.solutions",
}

# Mirrors site/assets/products-catalog.js (EN datasheets).
PRODUCTS = {
    "control": {
        "name": "ERA Control",
        "page": "control.html",
        "slogan": "ONE AGENT. ONE PLATFORM. ONE CONTROL.",
        "familyDs": "ERA-One-DataSheet.html",
        "description": (
            "ERA Control unifies XDR, vulnerability management, UEM, ITSM, PAM "
            "and network monitoring — one lightweight agent, modular licensing, air-gap ready."
        ),
        "editions": [
            ("ERA Core", "era-core", "01-ERA-Core.html"),
            ("ERA Control AI", "era-control-ai", "02-ERA-Control-AI.html"),
            ("ERA Response", "era-response", "03-ERA-Response.html"),
            ("ERA Vuln", "era-vuln", "04-ERA-Vuln.html"),
            ("ERA Federated / National", "era-federated-national", "05-ERA-Federated-National.html"),
            ("ERA Workbench", "era-workbench", "06-ERA-Workbench.html"),
            ("ERA Exposure", "era-exposure", "07-ERA-Exposure.html"),
            ("ERA BYO-EDR Hub", "era-byo-edr", "08-ERA-BYO-EDR.html"),
            ("ERA Manage", "era-manage", "09-ERA-Manage.html"),
            ("ERA Service", "era-service", "10-ERA-Service.html"),
            ("ERA Provision", "era-provision", "11-ERA-Provision.html"),
            ("ERA PAM", "era-pam", "12-ERA-PAM.html"),
            ("ERA Observe", "era-observe", "13-ERA-Observe.html"),
            ("ERA Perimeter", "era-perimeter", "15-ERA-Perimeter.html"),
            ("ERA Resolve", "era-resolve", "16-ERA-Resolve.html"),
            ("ERA Sovereign Hybrid", "era-sovereign-hybrid", "14-ERA-Sovereign-Hybrid.html"),
        ],
    },
    "communications": {
        "name": "ERA Communications",
        "page": "communications.html",
        "slogan": "ONE IDENTITY. ONE PLATFORM. ONE CONVERSATION.",
        "familyDs": "ERA-Communications-DataSheet.html",
        "description": (
            "ERA Communications delivers sovereign corporate mail, chat and video meetings "
            "with Outlook parity inside your perimeter."
        ),
        "editions": [
            ("ERA Mail Server", "era-mail-server", "comms-01-ERA-Mail-Server.html"),
            ("ERA Mail Client", "era-mail-client", "comms-02-ERA-Mail-Client.html"),
            ("ERA Conference", "era-conference", "comms-03-ERA-Conference.html"),
            ("ERA Chat", "era-chat", "comms-04-ERA-Chat.html"),
            ("ERA Comms AI", "era-comms-ai", "comms-05-ERA-Comms-AI.html"),
            ("ERA Mail Connect", "era-mail-connect", "comms-06-ERA-Mail-Connect.html"),
        ],
    },
    "office": {
        "name": "ERA Office",
        "page": "office.html",
        "slogan": "ONE WORKSPACE. ONE PLATFORM. ONE TEAM.",
        "familyDs": "ERA-Office-DataSheet.html",
        "description": (
            "ERA Office brings documents, tables, presentations and projects with co-editing "
            "into the isolated contour — without cloud dependencies."
        ),
        "editions": [
            ("ERA Drive", "era-drive", "office-00-ERA-Drive.html"),
            ("ERA Documents", "era-documents", "office-01-ERA-Documents.html"),
            ("ERA Tables", "era-tables", "office-02-ERA-Tables.html"),
            ("ERA Presentations", "era-presentations", "office-03-ERA-Presentations.html"),
            ("ERA Projects", "era-projects", "office-04-ERA-Projects.html"),
            ("ERA Office AI", "era-office-ai", "office-05-ERA-Office-AI.html"),
        ],
    },
}

INDEXABLE_STATIC = [
    "index.html",
    "control.html",
    "communications.html",
    "office.html",
    "about.html",
    "vision.html",
    "contacts.html",
    "downloads.html",
    "compare.html",
]


def read(path: Path) -> str:
    return path.read_text(encoding="utf-8")


def write(path: Path, text: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text, encoding="utf-8", newline="\n")


def extract_bodies(ds_html: str) -> str:
    """Extract inner HTML of all div.body blocks (handles nested divs)."""
    out: list[str] = []
    for m in re.finditer(r'<div\s+class="body"[^>]*>', ds_html, flags=re.I):
        start = m.end()
        depth = 1
        i = start
        lower = ds_html.lower()
        while i < len(ds_html) and depth:
            next_open = lower.find("<div", i)
            next_close = lower.find("</div>", i)
            if next_close < 0:
                break
            if next_open >= 0 and next_open < next_close:
                depth += 1
                i = next_open + 4
            else:
                depth -= 1
                if depth == 0:
                    out.append(ds_html[start:next_close])
                    break
                i = next_close + 6
    return "".join(out) if out else "<p>Datasheet content not found.</p>"


def demote_h1(fragment: str) -> str:
    return re.sub(r"<h1(\s[^>]*)?>", r"<h2\1>", fragment, count=1, flags=re.I).replace(
        "</h1>", "</h2>", 1
    )


def first_h1_text(fragment: str) -> str:
    m = re.search(r"<h1[^>]*>(.*?)</h1>", fragment, flags=re.I | re.S)
    if not m:
        return ""
    text = re.sub(r"<[^>]+>", " ", m.group(1))
    return re.sub(r"\s+", " ", text).strip()


def lead_text(fragment: str, limit: int = 160) -> str:
    m = re.search(r'<p\s+class="lead"[^>]*>(.*?)</p>', fragment, flags=re.I | re.S)
    if not m:
        m = re.search(r"<p[^>]*>(.*?)</p>", fragment, flags=re.I | re.S)
    if not m:
        return ""
    text = re.sub(r"<[^>]+>", " ", m.group(1))
    text = re.sub(r"\s+", " ", text).strip()
    if len(text) > limit:
        text = text[: limit - 1].rstrip() + "…"
    return text


def json_ld(obj: dict | list) -> str:
    return (
        '<script type="application/ld+json">\n'
        + json.dumps(obj, ensure_ascii=False, indent=2)
        + "\n</script>"
    )


def seo_head_block(
    *,
    canonical_path: str,
    title: str,
    description: str,
    extra_ld: list | None = None,
    noindex: bool = False,
) -> str:
    url = f"{CANON}/{canonical_path.lstrip('/')}" if canonical_path != "index.html" else f"{CANON}/"
    if canonical_path == "" or canonical_path == "/":
        url = f"{CANON}/"
    esc_title = html_lib.escape(title, quote=True)
    esc_desc = html_lib.escape(description, quote=True)
    robots = '<meta name="robots" content="noindex,nofollow" />\n' if noindex else ""
    ld_bits = [ORG]
    if extra_ld:
        ld_bits.extend(extra_ld)
    # Organization always; if list of graphs, emit one script per or combine
    scripts = "\n".join(json_ld(x) for x in ld_bits)
    return f"""{robots}<link rel="canonical" href="{url}" />
<link rel="icon" href="/favicon.svg" type="image/svg+xml" />
<meta property="og:type" content="website" />
<meta property="og:site_name" content="ERA One" />
<meta property="og:title" content="{esc_title}" />
<meta property="og:description" content="{esc_desc}" />
<meta property="og:url" content="{url}" />
<meta property="og:image" content="{OG_IMAGE}" />
<meta name="twitter:card" content="summary_large_image" />
<meta name="twitter:title" content="{esc_title}" />
<meta name="twitter:description" content="{esc_desc}" />
<meta name="twitter:image" content="{OG_IMAGE}" />
{scripts}
"""


def inject_before_stylesheet(html: str, block: str) -> str:
    # Avoid double-inject
    if 'rel="canonical"' in html:
        return html
    m = re.search(r'<link\s+rel="stylesheet"', html, flags=re.I)
    if m:
        return html[: m.start()] + block + html[m.start() :]
    m = re.search(r"</head>", html, flags=re.I)
    if m:
        return html[: m.start()] + block + html[m.start() :]
    return block + html


def ensure_title_desc(html: str, title: str | None, description: str | None) -> str:
    if title:
        html = re.sub(
            r"<title>[^<]*</title>",
            f"<title>{html_lib.escape(title)}</title>",
            html,
            count=1,
            flags=re.I,
        )
    if description:
        esc = html_lib.escape(description, quote=True)
        if re.search(r'<meta\s+name="description"', html, flags=re.I):
            html = re.sub(
                r'<meta\s+name="description"\s+content="[^"]*"\s*/?>',
                f'<meta name="description" content="{esc}" />',
                html,
                count=1,
                flags=re.I,
            )
        else:
            html = inject_before_stylesheet(
                html, f'<meta name="description" content="{esc}" />\n'
            )
    return html


def set_ds_content(html: str, body: str, *, prerendered: bool = True) -> str:
    attr = ' data-prerendered="1"' if prerendered else ""
    repl = f'<div id="ds-content" class="ds-content"{attr}>\n{body}\n    </div>'
    new_html, n = re.subn(
        r'<div\s+id="ds-content"[^>]*>.*?</div>',
        repl,
        html,
        count=1,
        flags=re.I | re.S,
    )
    if n == 0:
        raise RuntimeError("ds-content not found")
    return new_html


def load_en_datasheet(out: Path, ds_file: str) -> str:
    path = out / "datasheets" / "en" / ds_file
    if not path.is_file():
        # fallback ru
        path = out / "datasheets" / "ru" / ds_file
    if not path.is_file():
        raise FileNotFoundError(ds_file)
    return read(path)


def write_robots(out: Path) -> None:
    write(
        out / "robots.txt",
        f"""User-agent: *
Allow: /
Disallow: /legacy-portal.html
Disallow: /login.html
Disallow: /register.html
Disallow: /edition.html
Disallow: /datasheets/

Sitemap: {CANON}/sitemap.xml
""",
    )


def write_sitemap(out: Path, edition_paths: list[str], compare_en: list[str]) -> None:
    urls: list[tuple[str, str]] = []
    urls.append((f"{CANON}/", "1.0"))
    for p in INDEXABLE_STATIC:
        if p == "index.html":
            continue
        urls.append((f"{CANON}/{p}", "0.9" if p.endswith("control.html") or "communications" in p or p == "office.html" else "0.7"))
    for ep in edition_paths:
        urls.append((f"{CANON}/{ep}", "0.8"))
    for cp in compare_en:
        urls.append((f"{CANON}/{cp}", "0.6"))
    # clean URL stubs
    for stub in ("control", "communications", "office"):
        urls.append((f"{CANON}/{stub}/", "0.5"))

    lines = [
        '<?xml version="1.0" encoding="UTF-8"?>',
        '<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">',
    ]
    seen = set()
    for loc, pri in urls:
        if loc in seen:
            continue
        seen.add(loc)
        lines.append("  <url>")
        lines.append(f"    <loc>{xml_escape(loc)}</loc>")
        lines.append(f"    <priority>{pri}</priority>")
        lines.append("  </url>")
    lines.append("</urlset>")
    lines.append("")
    write(out / "sitemap.xml", "\n".join(lines))


def copy_favicon_and_og(out: Path) -> None:
    logo = out / "assets" / "era-one-logo.svg"
    if logo.is_file():
        shutil.copy2(logo, out / "favicon.svg")
    # OG image: prefer banner if present in distributor assets copied elsewhere
    banner_candidates = [
        out / "datasheets" / "assets" / "era-one-logo-banner.png",
        ROOT / "docs" / "distributor" / "assets" / "era-one-logo-banner.png",
        ROOT / "site" / "assets" / "era-one-logo-banner.png",
    ]
    dest = out / "assets" / "og-default.png"
    for c in banner_candidates:
        if c.is_file():
            shutil.copy2(c, dest)
            break
    else:
        # Fallback: copy svg as og-default.svg and point... plan wants png.
        # If no banner, keep referencing logo svg via updating OG_IMAGE usage —
        # write a tiny note by copying svg to og-default.svg and leave png missing;
        # inject will use svg if png absent.
        if logo.is_file():
            shutil.copy2(logo, out / "assets" / "og-default.svg")


def effective_og_image(out: Path) -> str:
    if (out / "assets" / "og-default.png").is_file():
        return f"{CANON}/assets/og-default.png"
    return f"{CANON}/assets/era-one-logo.svg"


def noindex_datasheets(out: Path) -> int:
    n = 0
    for path in (out / "datasheets").rglob("*.html"):
        html = read(path)
        if 'name="robots"' in html:
            continue
        html = inject_before_stylesheet(
            html, '<meta name="robots" content="noindex,follow" />\n'
        )
        write(path, html)
        n += 1
    return n


def enrich_static_page(
    out: Path,
    rel: str,
    *,
    extra_ld: list | None = None,
    noindex: bool = False,
    force_title: str | None = None,
    force_desc: str | None = None,
) -> None:
    path = out / rel
    if not path.is_file():
        return
    html = read(path)
    title_m = re.search(r"<title>([^<]*)</title>", html, flags=re.I)
    desc_m = re.search(
        r'<meta\s+name="description"\s+content="([^"]*)"', html, flags=re.I
    )
    title = force_title or (title_m.group(1) if title_m else "ERA One")
    desc = force_desc or (desc_m.group(1) if desc_m else "ERA One")
    # unescape for OG re-escape
    title = html_lib.unescape(title)
    desc = html_lib.unescape(desc)
    if force_title or force_desc:
        html = ensure_title_desc(html, force_title, force_desc)
    canon = "" if rel == "index.html" else rel
    block = seo_head_block(
        canonical_path=canon,
        title=title,
        description=desc,
        extra_ld=extra_ld,
        noindex=noindex,
    )
    # Patch OG image dynamically
    og = effective_og_image(out)
    block = block.replace(OG_IMAGE, og)
    html = inject_before_stylesheet(html, block)
    write(path, html)


def prerender_family(out: Path, key: str) -> None:
    meta = PRODUCTS[key]
    page = out / meta["page"]
    html = read(page)
    ds = load_en_datasheet(out, meta["familyDs"])
    body = demote_h1(extract_bodies(ds))
    html = set_ds_content(html, body)
    soft = SoftwareApplication(meta["name"], meta["description"])
    # head inject done later in enrich_static_page
    write(page, html)
    enrich_static_page(out, meta["page"], extra_ld=[soft])


def SoftwareApplication(name: str, description: str) -> dict:
    return {
        "@context": "https://schema.org",
        "@type": "SoftwareApplication",
        "name": name,
        "description": description,
        "applicationCategory": "BusinessApplication",
        "operatingSystem": "Windows, Linux, macOS",
        "url": CANON,
        "publisher": {"@type": "Organization", "name": "ERA One"},
    }


def edition_template(
    *,
    name: str,
    slug: str,
    family_key: str,
    family_name: str,
    family_page: str,
    slogan: str,
    body: str,
    title: str,
    description: str,
) -> str:
    soft = SoftwareApplication(name, description)
    head = seo_head_block(
        canonical_path=f"editions/{slug}.html",
        title=title,
        description=description,
        extra_ld=[soft],
    )
    # og image patched by caller if needed — use global; main() rewrites file
    return f"""<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<title>{html_lib.escape(title)}</title>
<meta name="description" content="{html_lib.escape(description, quote=True)}" />
{head}<link rel="stylesheet" href="/assets/site.css" />
</head>
<body data-page="products" data-family="{html_lib.escape(family_key)}" data-edition-slug="{html_lib.escape(slug)}" data-edition-ds="">
<header class="top" id="site-header"></header>
<section class="hero subhero">
  <div class="wrap">
    <nav class="ds-breadcrumb">
      <a href="/index.html" data-i18n="common.back">← Back to home</a>
      <span>/</span>
      <a href="{html_lib.escape(family_page)}">{html_lib.escape(family_name)}</a>
      <span>/</span>
      <span>{html_lib.escape(name)}</span>
    </nav>
    <div class="slogan">{html_lib.escape(slogan)}</div>
  </div>
</section>
<section class="ds-section">
  <div class="wrap">
    <div class="ds-toolbar">
      <button type="button" class="pdf-btn" id="ds-pdf-btn">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M12 3v12m0 0l-4-4m4 4l4-4M5 21h14"/></svg>
        <span data-i18n="common.download">Download PDF</span>
      </button>
    </div>
    <div id="ds-content" class="ds-content" data-prerendered="1">
{body}
    </div>
  </div>
</section>
<footer class="ftr" id="site-footer"></footer>
<script src="/assets/i18n-data.js"></script>
<script src="/assets/products-catalog.js"></script>
<script src="/assets/site.js"></script>
<script src="/assets/datasheet-view.js"></script>
</body>
</html>
"""


def generate_editions(out: Path) -> list[str]:
    paths: list[str] = []
    editions_dir = out / "editions"
    editions_dir.mkdir(exist_ok=True)
    og = effective_og_image(out)
    for key, meta in PRODUCTS.items():
        for name, slug, ds_file in meta["editions"]:
            ds = load_en_datasheet(out, ds_file)
            body = extract_bodies(ds)
            h1 = first_h1_text(body) or name
            desc = lead_text(body) or f"{name} — sovereign edition from ERA One."
            title = f"{h1} | ERA One"
            html = edition_template(
                name=name,
                slug=slug,
                family_key=key,
                family_name=meta["name"],
                family_page=meta["page"] if meta["page"].startswith("/") else f"/{meta['page']}",
                slogan=meta["slogan"],
                body=body,
                title=title,
                description=desc,
            )
            html = html.replace(OG_IMAGE, og)
            # embed ds filename for PDF / lang reload
            html = html.replace(
                'data-edition-ds=""',
                f'data-edition-ds="{html_lib.escape(ds_file, quote=True)}"',
            )
            rel = f"editions/{slug}.html"
            write(out / rel, html)
            paths.append(rel)
    return paths


def write_404(out: Path) -> None:
    write(
        out / "404.html",
        """<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<title>Page not found | ERA One</title>
<meta name="description" content="The requested page was not found on era-one.solutions." />
<meta name="robots" content="noindex,follow" />
<link rel="icon" href="/favicon.svg" type="image/svg+xml" />
<link rel="stylesheet" href="/assets/site.css" />
</head>
<body data-page="home">
<header class="top" id="site-header"></header>
<section class="hero subhero">
  <div class="wrap">
    <h1>Page not found</h1>
    <p class="lead">The page you requested does not exist or has moved.</p>
    <div class="actions">
      <a class="btn-lg btn-primary" href="/index.html">Back to home</a>
      <a class="btn-lg btn-ghost" href="/contacts.html">Contact us</a>
    </div>
  </div>
</section>
<footer class="ftr" id="site-footer"></footer>
<script src="/assets/i18n-data.js"></script>
<script src="/assets/products-catalog.js"></script>
<script src="/assets/site.js"></script>
</body>
</html>
""",
    )


def write_clean_stubs(out: Path) -> None:
    for key, meta in PRODUCTS.items():
        stub_dir = out / key
        stub_dir.mkdir(exist_ok=True)
        src = out / meta["page"]
        # Copy enriched family page; keep canonical on *.html (avoid duplicate signals).
        shutil.copy2(src, stub_dir / "index.html")


def list_compare_en(out: Path) -> list[str]:
    d = out / "compare" / "en"
    if not d.is_dir():
        return []
    return sorted(f"compare/en/{p.name}" for p in d.glob("*.html"))


def main() -> int:
    out = Path(sys.argv[1] if len(sys.argv) > 1 else ROOT / "dist" / "site")
    if not out.is_dir():
        print(f"ERROR: site dist not found: {out}", file=sys.stderr)
        return 1

    copy_favicon_and_og(out)
    write_robots(out)
    write_404(out)

    # Prerender families first (mutates ds-content)
    for key in PRODUCTS:
        prerender_family(out, key)

    edition_paths = generate_editions(out)

    # Home + company pages
    website_ld = {
        "@context": "https://schema.org",
        "@type": "WebSite",
        "name": "ERA One",
        "url": CANON,
        "publisher": {"@type": "Organization", "name": "ERA One"},
    }
    enrich_static_page(out, "index.html", extra_ld=[website_ld])
    for rel in (
        "about.html",
        "vision.html",
        "contacts.html",
        "downloads.html",
        "compare.html",
    ):
        enrich_static_page(out, rel)

    for rel in ("login.html", "register.html", "edition.html", "legacy-portal.html"):
        enrich_static_page(out, rel, noindex=True)

    n_ds = noindex_datasheets(out)
    write_clean_stubs(out)
    compare_en = list_compare_en(out)
    write_sitemap(out, edition_paths, compare_en)

    # Patch OG image in already-written SEO blocks if png appeared late
    og = effective_og_image(out)
    if og != OG_IMAGE:
        for path in out.rglob("*.html"):
            txt = read(path)
            if OG_IMAGE in txt:
                write(path, txt.replace(OG_IMAGE, og))

    print(
        f"OK: SEO enrich {out} — editions={len(edition_paths)} "
        f"datasheet_noindex={n_ds} compare_en={len(compare_en)}"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
