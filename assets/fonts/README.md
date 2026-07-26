# Bluffet Display

Pixel-exact replicas of the ALL YOU CAN / EAST BLUFFET logo letters.

**Logo letters:** `A B C E F L N O S T U Y`  
(other A–Z slots are placeholders)

## What’s “perfect”

1. **Masks** — black ink from the banner, no morph fattening  
2. **Vectors** — OpenCV contours with refill IoU **1.000** vs those masks  
3. **Bitmaps** — exact RGBA PNGs embedded as `sbix` strikes (native / 2× / 4×)  
4. **Previews** — phrases composited from those PNGs (same pixels as the banner)

## Files

| Path | Purpose |
|------|---------|
| `BluffetDisplay-Regular.ttf` | Vectors + sbix bitmaps |
| `glyphs/*.png` | Exact per-letter PNGs |
| `preview/logo_vs_font.png` | Banner vs exact glyph composite |
| `preview/compare_*.png` | Per-letter IoU proof |
| `build_bluffet_display.py` | Rebuild |

## Rebuild

```bash
pip install opencv-python-headless fonttools pillow numpy
python assets/fonts/build_bluffet_display.py
```

Apps that prefer `sbix` (e.g. many macOS text paths) will show the bitmap strikes at matching ppem; others fall back to the vector outlines.
