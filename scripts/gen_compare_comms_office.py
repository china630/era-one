#!/usr/bin/env python3
"""Generate H2H compare pages for Communications and Office (EN)."""
import os

ROOT = os.path.join(os.path.dirname(__file__), "..", "compare")

PAGES = {
    "ERA-vs-Exchange.html": {
        "title": "ERA Communications vs Microsoft Exchange",
        "lead": "Exchange is the classic on-prem mail stack. ERA Communications delivers mail, chat, meetings and Air-Gap AI in one sovereign perimeter — without CAL sprawl and foreign cloud dependencies.",
        "rows": [
            ("Model", "Exchange Server + separate Skype/Teams on-prem", "ERA Mail, Chat, Conference, Comms AI — one platform"),
            ("Deployment", "On-prem; hybrid to M365 common", "On-prem / air-gap; optional Sovereign Hybrid"),
            ("Data residency", "Customer DC; hybrid sync risks", "Data never leaves perimeter in sovereign mode"),
            ("Chat / meetings", "Separate products or cloud", "ERA Chat + ERA Conference (LiveKit on-prem)"),
            ("AI", "Copilot requires cloud", "ERA Comms AI — LLM inside perimeter"),
            ("Identity", "Active Directory centric", "Unified ERA One identity with Control & Office"),
            ("Migration", "Complex coexistence", "Zero-Touch Autodiscover migration path"),
        ],
        "summary": "ERA Communications wins on unified sovereign stack (mail + chat + meetings + AI), single vendor with ERA Control, and air-gap native design. Exchange remains strong where Microsoft ecosystem maturity and existing CAL investments dominate.",
    },
    "ERA-vs-Microsoft365-Comms.html": {
        "title": "ERA Communications vs Microsoft 365 (Exchange Online)",
        "lead": "Microsoft 365 is the default cloud communications stack. ERA Communications targets regulated customers who cannot export mail and collaboration data to foreign SaaS.",
        "rows": [
            ("Model", "Cloud SaaS subscription", "On-prem / air-gap in customer perimeter"),
            ("Data location", "Microsoft global cloud", "Customer data center only"),
            ("Phone-home", "Required for service", "None in air-gap mode"),
            ("Mail / calendar", "Exchange Online", "ERA Mail Server + Client"),
            ("Teams / meetings", "Cloud Teams", "ERA Conference on-prem"),
            ("AI / Copilot", "Cloud processing", "ERA Comms AI — local LLM"),
            ("Compliance", "Depends on region & DPA", "Full data sovereignty by design"),
        ],
        "summary": "Choose Microsoft 365 for global SaaS convenience. Choose ERA Communications when sovereignty, air-gap and a single local vendor (with ERA Control and Office) are mandatory.",
    },
    "ERA-vs-IceWarp.html": {
        "title": "ERA Communications vs IceWarp",
        "lead": "IceWarp is a popular on-prem mail and collaboration suite in CIS/MENA. ERA Communications positions as a next-gen sovereign stack with Rust/Go core and native ERA One integration.",
        "rows": [
            ("Core stack", "IceWarp proprietary suite", "Rust + Go; ClickHouse audit"),
            ("Mail / calendar", "Mature on-prem", "ERA Mail Server + Client"),
            ("Meetings", "IceWarp TeamChat / Conference", "ERA Conference (LiveKit)"),
            ("XDR / SOC integration", "Limited", "Native ERA Control SIEM integration"),
            ("Air-Gap AI", "Limited", "ERA Comms AI on-prem LLM"),
            ("Identity", "IceWarp directory", "Shared ERA One identity layer"),
        ],
        "summary": "IceWarp is a proven regional alternative to Exchange. ERA Communications competes on unified ERA One ecosystem, SecOps integration, and Air-Gap AI within the same perimeter.",
    },
    "ERA-vs-CommuniGate.html": {
        "title": "ERA Communications vs CommuniGate Pro",
        "lead": "CommuniGate Pro is a long-standing sovereign mail platform. ERA Communications modernizes the stack with microservices, LiveKit meetings and ERA Control integration.",
        "rows": [
            ("Heritage", "Mature mail + groupware", "Greenfield Rust/Go platform"),
            ("Meetings", "Add-on / limited", "ERA Conference (LiveKit)"),
            ("Chat", "Available", "ERA Chat — first-class"),
            ("AI", "Limited", "ERA Comms AI"),
            ("SecOps tie-in", "Minimal", "ERA Control data lake & cases"),
            ("Office docs", "Basic / separate", "Separate ERA Office family"),
        ],
        "summary": "CommuniGate Pro wins on installed base and mail maturity. ERA Communications wins on modern architecture, meetings scale, AI-in-perimeter and one vendor for security + comms + office.",
    },
    "ERA-vs-Microsoft365-Office.html": {
        "title": "ERA Office vs Microsoft 365 (Office)",
        "lead": "Microsoft 365 dominates cloud office productivity. ERA Office delivers documents, spreadsheets and presentations with co-editing inside the isolated perimeter.",
        "rows": [
            ("Model", "Cloud SaaS (Word, Excel, PPT online)", "On-prem ERA Documents, Tables, Presentations"),
            ("Co-editing", "Cloud real-time", "On-prem co-editing in contour"),
            ("File storage", "OneDrive / SharePoint cloud", "ERA Drive in customer perimeter"),
            ("Data residency", "Microsoft cloud", "Customer DC only"),
            ("Air-gap", "Not supported", "Native offline licensing & updates"),
            ("AI", "Copilot cloud", "ERA Office AI — local LLM"),
            ("Integration", "Teams / Exchange cloud", "ERA Communications + Control on same platform"),
        ],
        "summary": "Microsoft 365 wins on feature breadth and ecosystem. ERA Office wins when documents must stay inside a sovereign air-gapped contour alongside ERA Communications and Control.",
    },
    "ERA-vs-GoogleWorkspace.html": {
        "title": "ERA Office vs Google Workspace",
        "lead": "Google Workspace is a cloud-native office and collaboration suite. ERA Office targets customers who cannot use Google cloud for regulated documents.",
        "rows": [
            ("Model", "Cloud Docs, Sheets, Slides", "On-prem ERA Office editions"),
            ("Deployment", "100% SaaS", "Customer perimeter / air-gap"),
            ("Drive", "Google Drive cloud", "ERA Drive sovereign storage"),
            ("Compliance", "Google DPA / regions", "Full local control"),
            ("Mail integration", "Gmail cloud", "Optional ERA Communications on-prem"),
        ],
        "summary": "Google Workspace excels in cloud collaboration UX. ERA Office competes on sovereignty, offline operation and unified ERA One stack.",
    },
    "ERA-vs-OnlyOffice.html": {
        "title": "ERA Office vs OnlyOffice",
        "lead": "OnlyOffice is a common self-hosted office suite. ERA Office integrates natively with ERA Drive, ERA Communications and ERA Control under one identity and admin portal.",
        "rows": [
            ("Deployment", "Self-hosted document server", "ERA Office on-prem in ERA One stack"),
            ("Editors", "Docs, Sheets, Slides — mature", "ERA Documents, Tables, Presentations (roadmap)"),
            ("Storage", "Connectors to Nextcloud etc.", "ERA Drive — native shared platform"),
            ("Identity", "External IdP integration", "Unified ERA One identity"),
            ("Security stack", "Separate from SOC/XDR", "Same vendor as ERA Control XDR"),
            ("Mail / chat", "Not included", "ERA Communications family"),
        ],
        "summary": "OnlyOffice wins on mature self-hosted editors today. ERA Office wins as part of ERA One — one admin, one identity, sovereign drive, and integration with security and communications.",
    },
}


def render(lang: str, data: dict) -> str:
    rows_html = "".join(
        f"<tr><td>{r[0]}</td><td>{r[1]}</td><td><span class=\"win\">{r[2]}</span></td></tr>"
        for r in data["rows"]
    )
    return f"""<!doctype html>
<html lang="{lang}">
<head>
<meta charset="utf-8">
<title>{data['title']}</title>
<link rel="stylesheet" href="../../datasheets/assets/datasheet-common.css">
</head>
<body>
<div class="page page-stretch">
  <div class="hdr">
    <img class="logo-full" src="../../datasheets/assets/era-one-logo-banner.png" alt="ERA One">
  </div>
  <div class="body">
    <h1>{data['title']}</h1>
    <p class="lead">{data['lead']}</p>
    <table class="cmp"><tr><th>Criteria</th><th>Competitor</th><th>ERA One</th></tr>{rows_html}</table>
    <div class="summary"><b>Summary:</b> {data['summary']}</div>
  </div>
  <div class="ftr">
    <div class="ftr-contact">
      <div class="col"><b>Head Office</b>Geneva, Switzerland</div>
      <div class="col"><b>Engineering Center</b>Warsaw, Poland</div>
      <div class="col"><b>Contact</b>
        <a href="https://www.era-one.solutions">www.era-one.solutions</a><br>
        <a href="mailto:sales@era-one.solutions">sales@era-one.solutions</a>
      </div>
    </div>
    <div class="ftr-brand">
      <img class="logo-ftr" src="../../datasheets/assets/era-one-logo-banner.png" alt="ERA One">
    </div>
  </div>
</div>
</body>
</html>
"""


def main() -> None:
    for lang in ("en", "ru"):
        out_dir = os.path.join(ROOT, lang)
        os.makedirs(out_dir, exist_ok=True)
        for fname, data in PAGES.items():
            path = os.path.join(out_dir, fname)
            with open(path, "w", encoding="utf-8") as f:
                f.write(render(lang, data))
    print(f"Wrote {len(PAGES) * 2} compare pages")


if __name__ == "__main__":
    main()
