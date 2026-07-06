#!/usr/bin/env python3
"""Vectorize a raster logo into an SVG mosaic.

This is intentionally dependency-light (Pillow + stdlib) for air-gapped use.
It does not embed the raster image: the output SVG is made from vector rects
merged into horizontal runs after palette quantization.
"""

from __future__ import annotations

import argparse
import html
from pathlib import Path

from PIL import Image


def hex_rgb(rgb: tuple[int, int, int]) -> str:
    return "#{:02x}{:02x}{:02x}".format(*rgb)


def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser()
    p.add_argument("input", type=Path)
    p.add_argument("output", type=Path)
    p.add_argument("--cell", type=int, default=2, help="source pixels per vector cell")
    p.add_argument("--colors", type=int, default=96, help="palette size, 2..256")
    return p.parse_args()


def main() -> None:
    args = parse_args()
    if args.cell < 1:
        raise SystemExit("--cell must be >= 1")
    if not (2 <= args.colors <= 256):
        raise SystemExit("--colors must be between 2 and 256")

    src = Image.open(args.input).convert("RGB")
    full_w, full_h = src.size
    small_w = max(1, full_w // args.cell)
    small_h = max(1, full_h // args.cell)

    # Downsample gently, then reduce colors. This keeps the original composition
    # while producing a manageable amount of vector geometry.
    small = src.resize((small_w, small_h), Image.Resampling.LANCZOS)
    quant = small.quantize(colors=args.colors, method=Image.Quantize.MEDIANCUT)
    palette_raw = quant.getpalette() or []
    palette = [
        tuple(palette_raw[i : i + 3])  # type: ignore[arg-type]
        for i in range(0, min(len(palette_raw), args.colors * 3), 3)
    ]
    idx = quant.load()
    bg_idx = idx[0, 0]
    bg_color = hex_rgb(palette[bg_idx])

    parts: list[str] = []
    parts.append('<?xml version="1.0" encoding="UTF-8"?>')
    parts.append(
        f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {full_w} {full_h}" '
        f'width="{full_w}" height="{full_h}" role="img" '
        f'aria-label="{html.escape(args.input.stem)} vectorized logo">'
    )
    parts.append("  <metadata>")
    parts.append(
        f"    Vectorized from {html.escape(args.input.name)}; "
        f"cell={args.cell}; colors={args.colors}; no embedded raster."
    )
    parts.append("  </metadata>")
    parts.append(f'  <rect width="{full_w}" height="{full_h}" fill="{bg_color}"/>')
    parts.append("  <g shape-rendering=\"geometricPrecision\">")

    for y in range(small_h):
        x = 0
        while x < small_w:
            color_idx = idx[x, y]
            run_start = x
            x += 1
            while x < small_w and idx[x, y] == color_idx:
                x += 1

            if color_idx == bg_idx:
                continue

            run_w = x - run_start
            fill = hex_rgb(palette[color_idx])
            parts.append(
                f'    <rect x="{run_start * args.cell}" y="{y * args.cell}" '
                f'width="{run_w * args.cell}" height="{args.cell}" fill="{fill}"/>'
            )

    parts.append("  </g>")
    parts.append("</svg>")
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text("\n".join(parts) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()
