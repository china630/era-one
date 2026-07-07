#!/usr/bin/env python3
"""CI gate: RU datasheet HTML must be valid UTF-8 (not cp1251 with utf-8 meta)."""
from __future__ import annotations

import glob
import os
import sys

ROOT = os.path.join(os.path.dirname(__file__), "..", "datasheets", "ru")
MIN_CYRILLIC = 10


def main() -> int:
    failed: list[str] = []
    for path in sorted(glob.glob(os.path.join(ROOT, "*.html"))):
        name = os.path.basename(path)
        raw = open(path, "rb").read()
        try:
            text = raw.decode("utf-8")
        except UnicodeDecodeError as exc:
            failed.append(f"{name}: invalid UTF-8 ({exc})")
            continue
        if 'lang="ru"' not in text:
            continue
        cyr = sum(1 for ch in text if "\u0400" <= ch <= "\u04ff")
        if cyr < MIN_CYRILLIC:
            failed.append(f"{name}: expected Cyrillic text, got {cyr} chars")
    if failed:
        for line in failed:
            print("FAIL", line, file=sys.stderr)
        return 1
    print(f"PASS datasheet-encoding ({len(glob.glob(os.path.join(ROOT, '*.html')))} RU files)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
