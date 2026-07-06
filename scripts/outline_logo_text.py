#!/usr/bin/env python3
"""Convert ERA One SVG logo text blocks into SVG paths.

Uses fontTools and a local Windows font file. This avoids a dependency on
Inkscape for the final "text as curves" production asset.
"""

from __future__ import annotations

import argparse
import os
import re
from pathlib import Path

from fontTools.pens.svgPathPen import SVGPathPen
from fontTools.ttLib import TTFont


def cmap_for(font: TTFont) -> dict[int, str]:
    cmap: dict[int, str] = {}
    for table in font["cmap"].tables:
        cmap.update(table.cmap)
    return cmap


def glyph_paths(
    font: TTFont,
    text: str,
    *,
    x: float,
    baseline: float,
    size: float,
    letter_spacing: float,
    fill: str,
    class_name: str,
) -> tuple[str, float]:
    glyph_set = font.getGlyphSet()
    cmap = cmap_for(font)
    hmtx = font["hmtx"].metrics
    units = font["head"].unitsPerEm
    scale = size / units

    cur_x = x
    parts: list[str] = [f'  <g class="{class_name}" fill="{fill}">']
    for ch in text:
        glyph_name = cmap.get(ord(ch))
        if glyph_name is None:
            glyph_name = ".notdef"

        advance, _ = hmtx.get(glyph_name, (units * 0.5, 0))
        if ch != " ":
            pen = SVGPathPen(glyph_set)
            glyph_set[glyph_name].draw(pen)
            d = pen.getCommands()
            if d:
                parts.append(
                    f'    <path d="{d}" transform="translate({cur_x:.3f} {baseline:.3f}) '
                    f'scale({scale:.6f} {-scale:.6f})"/>'
                )
        cur_x += advance * scale + letter_spacing

    parts.append("  </g>")
    return "\n".join(parts), cur_x


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("input", type=Path)
    parser.add_argument("output", type=Path)
    parser.add_argument(
        "--font",
        type=Path,
        default=Path(os.environ["WINDIR"]) / "Fonts" / "EurostileLTProUnicode.ttf",
    )
    args = parser.parse_args()

    font = TTFont(args.font)
    era, _ = glyph_paths(
        font,
        "ERA",
        x=426,
        baseline=358,
        size=91,
        letter_spacing=12,
        fill="url(#whiteStroke)",
        class_name="wordmark-era-outlined",
    )
    one, _ = glyph_paths(
        font,
        "ONE",
        x=712,
        baseline=358,
        size=91,
        letter_spacing=12,
        fill="url(#cyanHorizontal)",
        class_name="wordmark-one-outlined",
    )

    x = 425.0
    slogan_parts: list[str] = []
    for text, fill in [
        ("ONE", "#38d7d0"),
        (" AGENT. ", "#d9e5eb"),
        ("ONE", "#38d7d0"),
        (" PLATFORM. ", "#d9e5eb"),
        ("ONE", "#38d7d0"),
        (" CONTROL.", "#d9e5eb"),
    ]:
        block, x = glyph_paths(
            font,
            text,
            x=x,
            baseline=402,
            size=17.5,
            letter_spacing=3.7,
            fill=fill,
            class_name="slogan-outlined",
        )
        slogan_parts.append(block)

    replacement = f"""  <!-- Wordmark and slogan outlined from EurostileLTProUnicode.ttf; no font dependency. -->
  <g id="wordmark-outlined" filter="url(#wordGlow)">
{era}
{one}
    <circle cx="676" cy="346" r="6.5" fill="#28d7d0"/>
  </g>

  <g id="slogan-outlined">
{chr(10).join(slogan_parts)}
  </g>"""

    svg = args.input.read_text(encoding="utf-8")
    svg = re.sub(
        r"  <!-- Wordmark\..*?  </text>\s*</svg>",
        replacement + "\n</svg>",
        svg,
        flags=re.S,
    )
    args.output.write_text(svg, encoding="utf-8")


if __name__ == "__main__":
    main()
