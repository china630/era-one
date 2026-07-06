#!/usr/bin/env python3
"""Сборка публичного портала ERA One из SSOT цен.

Читает:
  - docs/distributor/pricing-data.yaml (ERA Control, per-endpoint)
  - docs/distributor/pricing-comms-data.yaml (ERA Communications, per-user)
  - docs/distributor/pricing-office-data.yaml (ERA Office, per-user)

Генерирует site/pricing-data.js (window.ERA_PRICING).

Запуск: python scripts/build_portal.py
"""
from __future__ import annotations

import json
import shutil
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[1]
DIST = ROOT / "docs" / "distributor"
SSOT_CONTROL = DIST / "pricing-data.yaml"
SSOT_COMMS = DIST / "pricing-comms-data.yaml"
SSOT_OFFICE = DIST / "pricing-office-data.yaml"
SITE = ROOT / "site"
ASSETS = SITE / "assets"
LOGO_SRC = DIST / "assets" / "era-one-logo.png"
LOGO_DST = ASSETS / "era-one-logo.png"
DATA_JS = SITE / "pricing-data.js"


def load_yaml(path: Path) -> dict:
    if not path.is_file():
        raise SystemExit(f"Нет SSOT: {path}")
    return yaml.safe_load(path.read_text(encoding="utf-8"))


def main() -> None:
    control = load_yaml(SSOT_CONTROL)
    comms = load_yaml(SSOT_COMMS)
    office = load_yaml(SSOT_OFFICE)

    control["product_lines"] = {
        "communications": comms,
        "office": office,
    }

    ASSETS.mkdir(parents=True, exist_ok=True)
    payload = json.dumps(control, ensure_ascii=False, indent=2)
    DATA_JS.write_text(
        "// АВТОГЕНЕРАЦИЯ из pricing-data.yaml + pricing-comms/office-data.yaml\n"
        "// Пересборка: python scripts/build_portal.py\n"
        f"window.ERA_PRICING = {payload};\n",
        encoding="utf-8",
    )
    print(f"OK: {DATA_JS}")

    if LOGO_SRC.is_file():
        shutil.copyfile(LOGO_SRC, LOGO_DST)
        print(f"OK: {LOGO_DST}")
    else:
        print(f"WARN: логотип не найден: {LOGO_SRC}")


if __name__ == "__main__":
    main()
