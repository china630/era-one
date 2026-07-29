#!/usr/bin/env python3
"""Enrich dist/site/ with SEO artifacts + multi-lang prerender (en/ru/tr/ar).

EN lives at site root. Locales: /ru/, /tr/, /ar/ with hreflang alternates.
Usage: python3 scripts/site_seo_enrich.py [dist/site]
"""
from __future__ import annotations

import html as html_lib
import json
import re
import shutil
import subprocess
import sys
from pathlib import Path
from xml.sax.saxutils import escape as xml_escape

ROOT = Path(__file__).resolve().parents[1]
CANON = "https://www.era-one.solutions"
OG_IMAGE = f"{CANON}/assets/og-default.png"
LANGS = ("en", "ru", "tr", "ar")
NON_EN = ("ru", "tr", "ar")

ORG = {
    "@context": "https://schema.org",
    "@type": "Organization",
    "name": "ERA One",
    "url": CANON,
    "logo": f"{CANON}/assets/era-one-logo.png",
    "email": "sales@era-one.solutions",
    "address": {
        "@type": "PostalAddress",
        "addressLocality": "Geneva",
        "addressCountry": "CH",
    },
}

# Mirrors site/assets/products-catalog.js (datasheet basenames).
PRODUCTS = {
    "control": {
        "name": "ERA Control",
        "page": "control.html",
        "slogan": "ONE AGENT. ONE PLATFORM. ONE CONTROL.",
        "familyDs": "ERA-One-DataSheet.html",
        "i18n_h1": "pg.control.h1",
        "i18n_lead": "pg.control.lead",
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
        "i18n_h1": "pg.comms.h1",
        "i18n_lead": "pg.comms.lead",
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
        "i18n_h1": "pg.office.h1",
        "i18n_lead": "pg.office.lead",
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
    "privacy.html",
    "impressum.html",
    "partners.html",
    "careers.html",
]

# EN-relative path → (title_key, desc_key) for locale meta
STATIC_I18N_META = {
    "index.html": ("hero.tag", "hero.lead"),
    "about.html": ("pg.about.h1", "pg.about.lead"),
    "vision.html": ("pg.vision.h1", "pg.vision.lead"),
    "contacts.html": ("pg.contacts.h1", "pg.contacts.lead"),
    "downloads.html": ("dl.h1", "dl.lead"),
    "compare.html": ("compare.h1", "compare.lead"),
    "privacy.html": ("legal.privacy.h1", "legal.privacy.lead"),
    "impressum.html": ("legal.impressum.h1", "legal.impressum.lead"),
    "partners.html": ("legal.partners.h1", "legal.partners.lead"),
    "careers.html": ("legal.careers.h1", "legal.careers.lead"),
}


def read(path: Path) -> str:
    return path.read_text(encoding="utf-8")


def write(path: Path, text: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text, encoding="utf-8", newline="\n")


def page_rel(rel: str, lang: str) -> str:
    rel = rel.lstrip("/")
    if lang == "en":
        return rel
    return f"{lang}/{rel}"


def page_url(rel: str, lang: str) -> str:
    r = page_rel(rel, lang)
    if r in ("", "index.html"):
        return f"{CANON}/"
    if r.endswith("/index.html") and r.count("/") == 1:
        # ru/index.html → /ru/
        return f"{CANON}/{r.split('/')[0]}/"
    if r.endswith("index.html") and "/" in r:
        base = r[: -len("index.html")]
        return f"{CANON}/{base}"
    return f"{CANON}/{r}"


def hreflang_links(en_rel: str) -> str:
    lines: list[str] = []
    for lang in LANGS:
        lines.append(
            f'<link rel="alternate" hreflang="{lang}" href="{page_url(en_rel, lang)}" />'
        )
    lines.append(
        f'<link rel="alternate" hreflang="x-default" href="{page_url(en_rel, "en")}" />'
    )
    return "\n".join(lines) + "\n"


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


def plain_from_html(fragment: str, limit: int = 160) -> str:
    text = re.sub(r"<[^>]+>", " ", fragment)
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
    en_rel: str,
    lang: str,
    title: str,
    description: str,
    extra_ld: list | None = None,
    noindex: bool = False,
) -> str:
    url = page_url(en_rel, lang)
    esc_title = html_lib.escape(title, quote=True)
    esc_desc = html_lib.escape(description, quote=True)
    robots = '<meta name="robots" content="noindex,nofollow" />\n' if noindex else ""
    ld_bits = [ORG]
    if extra_ld:
        ld_bits.extend(extra_ld)
    scripts = "\n".join(json_ld(x) for x in ld_bits)
    return f"""{robots}<link rel="canonical" href="{url}" />
{hreflang_links(en_rel)}<link rel="icon" href="/assets/favicon.png" type="image/png" />
<meta property="og:type" content="website" />
<meta property="og:site_name" content="ERA One" />
<meta property="og:locale" content="{lang}" />
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


def strip_injected_seo(html: str) -> str:
    """Remove previously injected canonical/hreflang/og/json-ld so we can re-inject."""
    html = re.sub(
        r'\s*<link\s+rel="canonical"[^>]*>\s*',
        "\n",
        html,
        flags=re.I,
    )
    html = re.sub(
        r'\s*<link\s+rel="alternate"\s+hreflang="[^"]*"\s+href="[^"]*"\s*/?>\s*',
        "\n",
        html,
        flags=re.I,
    )
    html = re.sub(
        r'\s*<link\s+rel="icon"[^>]*>\s*',
        "\n",
        html,
        flags=re.I,
    )
    html = re.sub(
        r'\s*<meta\s+(?:property|name)="(?:og|twitter):[^"]*"[^>]*>\s*',
        "\n",
        html,
        flags=re.I,
    )
    html = re.sub(
        r'\s*<meta\s+name="robots"[^>]*>\s*',
        "\n",
        html,
        flags=re.I,
    )
    html = re.sub(
        r'\s*<script\s+type="application/ld\+json">.*?</script>\s*',
        "\n",
        html,
        flags=re.I | re.S,
    )
    return html


def inject_before_stylesheet(html: str, block: str) -> str:
    html = strip_injected_seo(html)
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


def set_html_lang(html: str, lang: str) -> str:
    dir_attr = ' dir="rtl"' if lang == "ar" else ""
    if re.search(r"<html\b", html, flags=re.I):
        return re.sub(
            r"<html\b[^>]*>",
            f'<html lang="{lang}"{dir_attr}>',
            html,
            count=1,
            flags=re.I,
        )
    return f'<html lang="{lang}"{dir_attr}>\n' + html


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


def lang_chain(lang: str) -> list[str]:
    if lang == "en":
        return ["en", "ru"]
    if lang == "ru":
        return ["ru", "en"]
    return [lang, "en", "ru"]


def try_load_datasheet(out: Path, ds_file: str, lang: str) -> str | None:
    for cand in lang_chain(lang):
        path = out / "datasheets" / cand / ds_file
        if path.is_file():
            return read(path)
    print(f"WARN: skip missing datasheet {ds_file} ({lang})", file=sys.stderr)
    return None


def load_i18n() -> dict:
    path = ROOT / "site" / "assets" / "i18n-data.js"
    code = (
        "const fs=require('fs');const vm=require('vm');"
        f"const ctx={{window:{{}}}};"
        f"vm.runInNewContext(fs.readFileSync({json.dumps(str(path))},'utf8'),ctx);"
        "process.stdout.write(JSON.stringify(ctx.window.ERA_I18N));"
    )
    r = subprocess.run(
        ["node", "-e", code],
        capture_output=True,
        text=True,
        encoding="utf-8",
        check=False,
    )
    if r.returncode != 0:
        raise RuntimeError(f"load_i18n failed: {r.stderr}")
    return json.loads(r.stdout)


def apply_data_i18n(html: str, d: dict) -> str:
    """Replace inner HTML of elements that have data-i18n (non-nested tags)."""

    def repl(m: re.Match[str]) -> str:
        key = m.group(4)
        if key not in d:
            return m.group(0)
        return f"{m.group(1)}{d[key]}{m.group(6)}"

    return re.sub(
        r"(<(\w+)([^>]*\sdata-i18n=\"([^\"]+)\"[^>]*)>)(.*?)(</\2>)",
        repl,
        html,
        flags=re.S,
    )


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


def write_sitemap(out: Path, edition_en_rels: list[str], compare_en: list[str]) -> None:
    """Emit every language version with full xhtml hreflang cluster."""
    en_rels: list[str] = []
    for p in INDEXABLE_STATIC:
        en_rels.append(p)
    en_rels.extend(edition_en_rels)
    en_rels.extend(compare_en)
    for stub in ("control", "communications", "office"):
        en_rels.append(f"{stub}/")

    lines = [
        '<?xml version="1.0" encoding="UTF-8"?>',
        '<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"',
        '        xmlns:xhtml="http://www.w3.org/1999/xhtml">',
    ]
    seen: set[str] = set()
    for en_rel in en_rels:
        for lang in LANGS:
            # clean stubs only for EN
            if en_rel.endswith("/") and lang != "en":
                continue
            loc = page_url(en_rel if en_rel != "index.html" else "index.html", lang)
            if en_rel.endswith("/"):
                loc = f"{CANON}/{en_rel}"
            if loc in seen:
                continue
            seen.add(loc)
            pri = "1.0" if en_rel == "index.html" else "0.8"
            if en_rel in ("control.html", "communications.html", "office.html"):
                pri = "0.9"
            lines.append("  <url>")
            lines.append(f"    <loc>{xml_escape(loc)}</loc>")
            for alt in LANGS:
                if en_rel.endswith("/") and alt != "en":
                    continue
                href = (
                    f"{CANON}/{en_rel}"
                    if en_rel.endswith("/")
                    else page_url(en_rel, alt)
                )
                lines.append(
                    f'    <xhtml:link rel="alternate" hreflang="{alt}" '
                    f'href="{xml_escape(href)}" />'
                )
            if not en_rel.endswith("/"):
                lines.append(
                    f'    <xhtml:link rel="alternate" hreflang="x-default" '
                    f'href="{xml_escape(page_url(en_rel, "en"))}" />'
                )
            lines.append(f"    <priority>{pri}</priority>")
            lines.append("  </url>")
    lines.append("</urlset>")
    lines.append("")
    write(out / "sitemap.xml", "\n".join(lines))


def copy_favicon_and_og(out: Path) -> None:
    assets = out / "assets"
    assets.mkdir(parents=True, exist_ok=True)
    logo_png = assets / "era-one-logo.png"
    if not logo_png.is_file():
        print("WARN: site/assets/era-one-logo.png missing", file=sys.stderr)
        return
    shutil.copy2(logo_png, assets / "favicon.png")
    shutil.copy2(logo_png, out / "favicon.png")
    shutil.copy2(logo_png, assets / "og-default.png")


def effective_og_image(out: Path) -> str:
    if (out / "assets" / "og-default.png").is_file():
        return f"{CANON}/assets/og-default.png"
    return f"{CANON}/assets/era-one-logo.png"


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


def SoftwareApplication(name: str, description: str, url: str) -> dict:
    return {
        "@context": "https://schema.org",
        "@type": "SoftwareApplication",
        "name": name,
        "description": description,
        "applicationCategory": "BusinessApplication",
        "operatingSystem": "Windows, Linux, macOS",
        "url": url,
        "publisher": {"@type": "Organization", "name": "ERA One"},
    }


def enrich_page(
    out: Path,
    en_rel: str,
    lang: str,
    *,
    extra_ld: list | None = None,
    noindex: bool = False,
    force_title: str | None = None,
    force_desc: str | None = None,
) -> None:
    rel = page_rel(en_rel, lang)
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
    title = html_lib.unescape(title)
    desc = html_lib.unescape(desc)
    if force_title or force_desc:
        html = ensure_title_desc(html, force_title, force_desc)
    html = set_html_lang(html, lang)
    block = seo_head_block(
        en_rel=en_rel,
        lang=lang,
        title=title,
        description=desc,
        extra_ld=extra_ld,
        noindex=noindex,
    )
    og = effective_og_image(out)
    block = block.replace(OG_IMAGE, og)
    html = inject_before_stylesheet(html, block)
    write(path, html)


def i18n_get(i18n: dict, lang: str, key: str, default: str = "") -> str:
    d = i18n.get(lang) or {}
    if key in d:
        return plain_from_html(d[key], 300) if "<" in str(d[key]) else d[key]
    en = i18n.get("en") or {}
    if key in en:
        v = en[key]
        return plain_from_html(v, 300) if "<" in str(v) else v
    return default


def prerender_family(out: Path, key: str, lang: str, i18n: dict) -> str | None:
    meta = PRODUCTS[key]
    en_rel = meta["page"]
    # Always start from EN source shell in dist (or site) without relying on prior body.
    src = out / en_rel
    if lang != "en":
        # Prefer clean shell from EN root file before/after — re-read source from site/
        site_src = ROOT / "site" / en_rel
        if site_src.is_file():
            html = read(site_src)
        elif src.is_file():
            html = read(src)
        else:
            print(f"WARN: skip missing family page {en_rel}", file=sys.stderr)
            return None
    else:
        if not src.is_file():
            print(f"WARN: skip missing family page {en_rel}", file=sys.stderr)
            return None
        html = read(src)

    ds = try_load_datasheet(out, meta["familyDs"], lang)
    if ds is None:
        return None
    body = demote_h1(extract_bodies(ds))
    html = set_ds_content(html, body)
    dict_lang = i18n.get(lang) or i18n.get("en") or {}
    html = apply_data_i18n(html, dict_lang)
    html = set_html_lang(html, lang)

    title = f"{i18n_get(i18n, lang, meta['i18n_h1'], meta['name'])} | ERA One"
    desc = i18n_get(i18n, lang, meta["i18n_lead"], meta["description"])
    html = ensure_title_desc(html, title, desc)

    rel = page_rel(en_rel, lang)
    write(out / rel, html)
    soft = SoftwareApplication(meta["name"], desc, page_url(en_rel, lang))
    enrich_page(
        out,
        en_rel,
        lang,
        extra_ld=[soft],
        force_title=title,
        force_desc=desc,
    )
    return rel


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
    lang: str,
    en_rel: str,
) -> str:
    soft = SoftwareApplication(name, description, page_url(en_rel, lang))
    head = seo_head_block(
        en_rel=en_rel,
        lang=lang,
        title=title,
        description=description,
        extra_ld=[soft],
    )
    fam_href = family_page if family_page.startswith("/") else f"/{family_page}"
    if lang != "en":
        fam_href = f"/{lang}{fam_href}"
    home = "/index.html" if lang == "en" else f"/{lang}/index.html"
    return f"""<!DOCTYPE html>
<html lang="{lang}"{" dir=\"rtl\"" if lang == "ar" else ""}>
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
      <a href="{home}" data-i18n="common.back">← Back to home</a>
      <span>/</span>
      <a href="{html_lib.escape(fam_href)}">{html_lib.escape(family_name)}</a>
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


def generate_editions(out: Path, lang: str, i18n: dict) -> list[str]:
    """Return EN-relative edition paths (editions/slug.html) for sitemap."""
    en_paths: list[str] = []
    og = effective_og_image(out)
    for key, meta in PRODUCTS.items():
        for name, slug, ds_file in meta["editions"]:
            ds = try_load_datasheet(out, ds_file, lang)
            if ds is None:
                continue
            body = extract_bodies(ds)
            dict_lang = i18n.get(lang) or {}
            body_wrap = f'<div id="_ed">{body}</div>'
            body_wrap = apply_data_i18n(body_wrap, dict_lang)
            body = body_wrap[len('<div id="_ed">') : -len("</div>")]
            h1 = first_h1_text(body) or name
            desc = lead_text(body) or f"{name} — sovereign edition from ERA One."
            title = f"{h1} | ERA One"
            en_rel = f"editions/{slug}.html"
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
                lang=lang,
                en_rel=en_rel,
            )
            html = html.replace(OG_IMAGE, og)
            html = html.replace(
                'data-edition-ds=""',
                f'data-edition-ds="{html_lib.escape(ds_file, quote=True)}"',
            )
            write(out / page_rel(en_rel, lang), html)
            if lang == "en":
                en_paths.append(en_rel)
    return en_paths


def localize_static_page(out: Path, en_rel: str, lang: str, i18n: dict) -> None:
    if lang == "en":
        return
    src = out / en_rel
    if not src.is_file():
        site_src = ROOT / "site" / en_rel
        if not site_src.is_file():
            return
        html = read(site_src)
    else:
        html = read(src)
    dict_lang = i18n.get(lang) or {}
    html = apply_data_i18n(html, dict_lang)
    html = set_html_lang(html, lang)
    keys = STATIC_I18N_META.get(en_rel)
    force_title = force_desc = None
    if keys:
        h1 = i18n_get(i18n, lang, keys[0])
        lead = i18n_get(i18n, lang, keys[1])
        if h1:
            force_title = f"{h1} | ERA One"
        if lead:
            force_desc = lead
    write(out / page_rel(en_rel, lang), html)
    enrich_page(
        out,
        en_rel,
        lang,
        force_title=force_title,
        force_desc=force_desc,
    )


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
<link rel="icon" href="/assets/favicon.png" type="image/png" />
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
        if src.is_file():
            shutil.copy2(src, stub_dir / "index.html")


def list_compare_en(out: Path) -> list[str]:
    d = out / "compare" / "en"
    if not d.is_dir():
        return []
    return sorted(f"compare/en/{p.name}" for p in d.glob("*.html"))


def legal_page_html(slug: str, h1_key: str, lead_key: str, body_en: str) -> str:
    return f"""<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<title>ERA One</title>
<meta name="description" content="" />
<link rel="stylesheet" href="/assets/site.css" />
</head>
<body data-page="{slug}">
<header class="top" id="site-header"></header>
<section class="hero subhero">
  <div class="wrap">
    <h1 data-i18n="{h1_key}">Title</h1>
    <p class="lead" data-i18n="{lead_key}">Lead</p>
  </div>
</section>
<section>
  <div class="wrap">
    <div class="prose">
{body_en}
    </div>
  </div>
</section>
<footer class="ftr" id="site-footer"></footer>
<script src="/assets/i18n-data.js"></script>
<script src="/assets/products-catalog.js"></script>
<script src="/assets/site.js"></script>
</body>
</html>
"""


def ensure_legal_source_pages(site_src: Path) -> None:
    pages = {
        "privacy.html": (
            "privacy",
            "legal.privacy.h1",
            "legal.privacy.lead",
            """      <p>ERA One processes personal data submitted via sales inquiries (name, work email, organization, message) solely to respond to your request.</p>
      <p>We do not sell personal data. Data is retained only as long as needed for the sales process or applicable law.</p>
      <p>Contact: <a href="mailto:sales@era-one.solutions">sales@era-one.solutions</a>. Head office: Geneva, Switzerland.</p>""",
        ),
        "impressum.html": (
            "impressum",
            "legal.impressum.h1",
            "legal.impressum.lead",
            """      <p><strong>ERA One</strong></p>
      <p>Head office: Geneva, Switzerland</p>
      <p>Email: <a href="mailto:sales@era-one.solutions">sales@era-one.solutions</a></p>
      <p>Website: <a href="https://www.era-one.solutions">www.era-one.solutions</a></p>""",
        ),
        "partners.html": (
            "partners",
            "legal.partners.h1",
            "legal.partners.lead",
            """      <p>ERA One works with distributors and system integrators for sovereign deployments of Control, Communications and Office.</p>
      <p>For partnership inquiries write to <a href="mailto:sales@era-one.solutions">sales@era-one.solutions</a>.</p>""",
        ),
        "careers.html": (
            "careers",
            "legal.careers.h1",
            "legal.careers.lead",
            """      <p>We hire engineers and operators who build air-gap-ready platforms for regulated environments.</p>
      <p>Send your profile to <a href="mailto:sales@era-one.solutions">sales@era-one.solutions</a> with subject “Careers”.</p>""",
        ),
    }
    for fname, (slug, h1, lead, body) in pages.items():
        path = site_src / fname
        if path.is_file():
            continue
        write(path, legal_page_html(slug, h1, lead, body))


def main() -> int:
    out = Path(sys.argv[1] if len(sys.argv) > 1 else ROOT / "dist" / "site")
    if not out.is_dir():
        print(f"ERROR: site dist not found: {out}", file=sys.stderr)
        return 1

    site_src = ROOT / "site"
    ensure_legal_source_pages(site_src)
    # Copy any newly created legal pages into dist
    for fname in ("privacy.html", "impressum.html", "partners.html", "careers.html"):
        src = site_src / fname
        dst = out / fname
        if src.is_file() and not dst.is_file():
            shutil.copy2(src, dst)

    i18n = load_i18n()
    copy_favicon_and_og(out)
    write_robots(out)
    write_404(out)

    # EN family prerender
    for key in PRODUCTS:
        prerender_family(out, key, "en", i18n)

    edition_paths = generate_editions(out, "en", i18n)

    website_ld = {
        "@context": "https://schema.org",
        "@type": "WebSite",
        "name": "ERA One",
        "url": CANON,
        "inLanguage": list(LANGS),
        "publisher": {"@type": "Organization", "name": "ERA One"},
    }
    enrich_page(out, "index.html", "en", extra_ld=[website_ld])

    for rel in (
        "about.html",
        "vision.html",
        "contacts.html",
        "downloads.html",
        "compare.html",
        "privacy.html",
        "impressum.html",
        "partners.html",
        "careers.html",
    ):
        enrich_page(out, rel, "en")

    for rel in ("login.html", "register.html", "edition.html", "legacy-portal.html"):
        enrich_page(out, rel, "en", noindex=True)

    # Locale trees
    for lang in NON_EN:
        # locale home
        localize_static_page(out, "index.html", lang, i18n)
        for key in PRODUCTS:
            prerender_family(out, key, lang, i18n)
        generate_editions(out, lang, i18n)
        for rel in (
            "about.html",
            "vision.html",
            "contacts.html",
            "downloads.html",
            "compare.html",
            "privacy.html",
            "impressum.html",
            "partners.html",
            "careers.html",
        ):
            localize_static_page(out, rel, lang, i18n)

    n_ds = noindex_datasheets(out)
    write_clean_stubs(out)
    compare_en = list_compare_en(out)
    # Enrich compare EN pages with canonical (no full locale matrix)
    for cp in compare_en:
        enrich_page(out, cp, "en")
    write_sitemap(out, edition_paths, compare_en)

    og = effective_og_image(out)
    if og != OG_IMAGE:
        for path in out.rglob("*.html"):
            txt = read(path)
            if OG_IMAGE in txt:
                write(path, txt.replace(OG_IMAGE, og))

    print(
        f"OK: SEO enrich {out} — editions_en={len(edition_paths)} "
        f"datasheet_noindex={n_ds} langs={','.join(LANGS)}"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
