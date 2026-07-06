#!/usr/bin/env python3
"""Обрезка вертикальных полей логотипа (~20% сверху и снизу) для шапки/юбки PDF."""
from __future__ import annotations

from pathlib import Path

try:
    from PIL import Image
except ImportError as e:
    raise SystemExit("Нужен Pillow: pip install pillow") from e

ROOT = Path(__file__).resolve().parents[1]
SRC = ROOT / "docs" / "distributor" / "assets" / "era-one-logo.png"
OUT = ROOT / "docs" / "distributor" / "assets" / "era-one-logo-banner.png"
CROP_RATIO = 0.20  # доля высоты, срезаемая сверху и снизу


def main() -> None:
    if not SRC.is_file():
        raise SystemExit(f"Нет исходника: {SRC}")
    im = Image.open(SRC).convert("RGBA")
    w, h = im.size
    top = int(h * CROP_RATIO)
    bottom = int(h * CROP_RATIO)
    cropped = im.crop((0, top, w, h - bottom))
    OUT.parent.mkdir(parents=True, exist_ok=True)
    cropped.save(OUT, optimize=True)
    print(f"OK: {OUT} ({cropped.size[0]}x{cropped.size[1]} из {w}x{h})")


if __name__ == "__main__":
    main()
