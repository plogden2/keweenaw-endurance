"""
BluffetDisplay — perfect logo letter replicas.

1) Black-ink masks → vector outlines (OpenCV drawContours IoU == 1.0)
2) Exact RGBA PNGs of each logo letter → sbix bitmap strikes
3) Previews composited from those PNGs (pixel-identical to banner letters)
"""

from __future__ import annotations

import io
from pathlib import Path

import cv2
import numpy as np
from fontTools.fontBuilder import FontBuilder
from fontTools.pens.ttGlyphPen import TTGlyphPen
from fontTools.ttLib import TTFont, newTable
from fontTools.ttLib.tables.sbixGlyph import Glyph as SbixGlyph
from fontTools.ttLib.tables.sbixStrike import Strike
from PIL import Image, ImageDraw, ImageFont

OUT_DIR = Path(__file__).resolve().parent
TTF_PATH = OUT_DIR / "BluffetDisplay-Regular.ttf"
PREVIEW_DIR = OUT_DIR / "preview"
GLYPH_PNG_DIR = OUT_DIR / "glyphs"
BANNER = Path(
    r"C:\Users\gener\.cursor\projects\c-Users-gener-Documents-keweenaw-endurance"
    r"\assets\c__Users_gener_AppData_Roaming_Cursor_User_workspaceStorage_"
    r"de2a706eef9815289cb6cbbb59ec2956_images_image-24993862-37bc-48ff-80b1-5b41222af2a8.png"
)

PX = 16
UPM = 2048
ASCENDER = 1600
DESCENDER = -400
CAP = 1472
LOGO_CHARS = "ABCEFLNOSTUY"


def load_banner() -> np.ndarray:
    return np.array(Image.open(BANNER).convert("RGB"))


def ink_mask(rgb: np.ndarray) -> np.ndarray:
    return (
        (rgb[:, :, 0] < 80) & (rgb[:, :, 1] < 80) & (rgb[:, :, 2] < 80)
    ).astype(np.uint8) * 255


def solid_letter_mask(rgb_crop: np.ndarray) -> np.ndarray:
    """Black-core mask — no morph-close fattening."""
    ink = ink_mask(rgb_crop)
    n, labels, stats, _ = cv2.connectedComponentsWithStats(ink, 8)
    if n <= 1:
        return ink
    largest = 1 + int(np.argmax(stats[1:, cv2.CC_STAT_AREA]))
    solid = np.where(labels == largest, 255, 0).astype(np.uint8)
    inv = cv2.bitwise_not(solid)
    n2, lab2, st2, _ = cv2.connectedComponentsWithStats(inv, 8)
    for i in range(1, n2):
        if st2[i, cv2.CC_STAT_AREA] < 8:
            solid[lab2 == i] = 255
    return solid


def letter_rgba(rgb_crop: np.ndarray, solid: np.ndarray) -> np.ndarray:
    """
    Exact logo letter appearance as RGBA.

    Keeps black ink + the thin light inline ring around it (from the crop),
    punches background to transparent. This is what gets embedded in sbix.
    """
    h, w = solid.shape
    # Grow slightly to catch white inline / outer hairline antialias
    ring = cv2.dilate(solid, np.ones((3, 3), np.uint8), iterations=2)
    rgba = np.zeros((h, w, 4), dtype=np.uint8)
    rgba[:, :, :3] = rgb_crop[:h, :w]
    rgba[:, :, 3] = np.where(ring > 0, 255, 0)
    # Force truly transparent outside
    rgba[ring == 0] = 0
    return rgba


def discover_first_boxes(rgb: np.ndarray) -> dict[str, tuple[int, int, int, int]]:
    black = ink_mask(rgb)
    k = cv2.getStructuringElement(cv2.MORPH_RECT, (2, 2))
    seg = cv2.morphologyEx(black, cv2.MORPH_CLOSE, k, iterations=1)
    specs = [("ALLYOUCAN", 16, 122), ("EASTBLUFFET", 148, 260)]
    first: dict[str, tuple[int, int, int, int]] = {}

    for text, y0, y1 in specs:
        band = seg[y0:y1, :]
        col = (band > 0).any(axis=0)
        runs: list[tuple[int, int]] = []
        in_run = False
        start = 0
        for i, v in enumerate(col):
            if v and not in_run:
                in_run = True
                start = i
            elif not v and in_run:
                in_run = False
                if i - start >= 8:
                    runs.append((start, i))
        if in_run and len(col) - start >= 8:
            runs.append((start, len(col)))

        guard = 0
        while len(runs) < len(text) and guard < 24:
            guard += 1
            widths = sorted(((b - a, i) for i, (a, b) in enumerate(runs)), reverse=True)
            split_done = False
            for _, idx in widths:
                a, b = runs[idx]
                if b - a < 36:
                    continue
                profile = band[:, a:b].sum(axis=0).astype(np.float64)
                c0, c1 = int((b - a) * 0.28), int((b - a) * 0.72)
                rel = c0 + int(np.argmin(profile[c0:c1]))
                split = a + rel
                if split <= a + 10 or split >= b - 10:
                    continue
                runs = runs[:idx] + [(a, split), (split, b)] + runs[idx + 1 :]
                split_done = True
                break
            if not split_done:
                break

        print(f"{text}: {len(runs)} boxes (want {len(text)})")
        vis = cv2.cvtColor(band, cv2.COLOR_GRAY2BGR)
        for i, (x0, x1) in enumerate(runs):
            cv2.rectangle(vis, (x0, 0), (x1 - 1, y1 - y0 - 1), (0, 255, 0), 1)
            if i >= len(text):
                continue
            ch = text[i]
            cv2.putText(vis, ch, (x0, 16), cv2.FONT_HERSHEY_SIMPLEX, 0.5, (0, 0, 255), 1)
            if ch in first:
                continue
            ink_band = black[y0:y1, x0:x1]
            rows = np.where(ink_band.any(axis=1))[0]
            cols = np.where(ink_band.any(axis=0))[0]
            if len(rows) == 0 or len(cols) == 0:
                continue
            yy0, yy1 = int(rows[0]), int(rows[-1]) + 1
            xx0, xx1 = int(cols[0]), int(cols[-1]) + 1
            pad = 1
            first[ch] = (
                max(0, x0 + xx0 - pad),
                max(0, y0 + yy0 - pad),
                min(rgb.shape[1], x0 + xx1 + pad),
                min(rgb.shape[0], y0 + yy1 + pad),
            )
        PREVIEW_DIR.mkdir(parents=True, exist_ok=True)
        Image.fromarray(vis).save(PREVIEW_DIR / f"seg_{text}.png")
    return first


def iou(a: np.ndarray, b: np.ndarray) -> float:
    aa, bb = a > 0, b > 0
    inter = np.logical_and(aa, bb).sum()
    union = np.logical_or(aa, bb).sum()
    return float(inter) / float(union) if union else 0.0


def signed_area(pts) -> float:
    arr = np.asarray(pts, dtype=np.float64)
    x, y = arr[:, 0], arr[:, 1]
    return 0.5 * float(np.dot(x, np.roll(y, -1)) - np.dot(y, np.roll(x, -1)))


def extract_contours(mask: np.ndarray):
    cnts, hier = cv2.findContours(mask, cv2.RETR_CCOMP, cv2.CHAIN_APPROX_NONE)
    if hier is None or not cnts:
        raise RuntimeError("no contours")
    check = np.zeros_like(mask)
    cv2.drawContours(check, cnts, -1, 255, thickness=cv2.FILLED, hierarchy=hier)
    score = iou(mask, check)
    if score < 1.0 - 1e-9:
        raise RuntimeError(f"contour refill IoU {score} != 1.0")

    hier = hier[0]
    out = []
    for i, cnt in enumerate(cnts):
        parent = hier[i][3]
        if parent != -1 and hier[parent][3] != -1:
            continue
        if abs(cv2.contourArea(cnt)) < 2:
            continue
        pts = cnt.reshape(-1, 2).astype(np.float64)
        out.append((pts, parent != -1))

    outers = [(p, h) for p, h in out if not h]
    outers.sort(key=lambda t: abs(cv2.contourArea(t[0].astype(np.float32))), reverse=True)
    main = outers[0][0]
    ob = (main[:, 0].min(), main[:, 1].min(), main[:, 0].max(), main[:, 1].max())
    kept = [outers[0]]
    for p, h in out:
        if not h:
            continue
        cx, cy = p[:, 0].mean(), p[:, 1].mean()
        if ob[0] <= cx <= ob[2] and ob[1] <= cy <= ob[3]:
            kept.append((p, True))
    return kept, score


def to_font_units(contours, pad_px: int = 1):
    xs = np.concatenate([c[0][:, 0] for c in contours])
    ys = np.concatenate([c[0][:, 1] for c in contours])
    min_x, max_x = float(xs.min()), float(xs.max())
    min_y, max_y = float(ys.min()), float(ys.max())
    font_contours = []
    for pts, is_hole in contours:
        scaled = []
        for x, y in pts:
            fx = int(round((x - min_x + pad_px) * PX))
            fy = int(round((max_y - y) * PX))
            scaled.append((fx, fy))
        area = signed_area(scaled)
        if not is_hole and area > 0:
            scaled = list(reversed(scaled))
        if is_hole and area < 0:
            scaled = list(reversed(scaled))
        clean = [scaled[0]]
        for p in scaled[1:]:
            if p != clean[-1]:
                clean.append(p)
        if len(clean) >= 2 and clean[0] == clean[-1]:
            clean = clean[:-1]
        if len(clean) >= 3:
            font_contours.append(clean)
    width = int(round((max_x - min_x + pad_px * 2) * PX))
    return font_contours, max(width, 8 * PX)


def contours_to_glyph(font_contours):
    pen = TTGlyphPen(None)
    for pts in font_contours:
        pen.moveTo(pts[0])
        for p in pts[1:]:
            pen.lineTo(p)
        pen.closePath()
    return pen.glyph()


def placeholder_glyph():
    pen = TTGlyphPen(None)
    pen.moveTo((80, 0))
    pen.lineTo((80, CAP))
    pen.lineTo((480, CAP))
    pen.lineTo((480, 0))
    pen.closePath()
    return pen.glyph()


def png_bytes(rgba: np.ndarray) -> bytes:
    buf = io.BytesIO()
    Image.fromarray(rgba, "RGBA").save(buf, format="PNG")
    return buf.getvalue()


def extract_logo_glyphs(rgb: np.ndarray):
    boxes = discover_first_boxes(rgb)
    print("boxes:", {k: boxes[k] for k in sorted(boxes)})
    glyphs = {}
    pngs = {}
    masks = {}
    ious = {}
    PREVIEW_DIR.mkdir(parents=True, exist_ok=True)
    GLYPH_PNG_DIR.mkdir(parents=True, exist_ok=True)

    for ch, (x0, y0, x1, y1) in boxes.items():
        crop = rgb[y0:y1, x0:x1]
        solid = solid_letter_mask(crop)
        # Trim empty padding so advance/height match real ink (fixes tall F box)
        rows = np.where(solid.any(axis=1))[0]
        cols = np.where(solid.any(axis=0))[0]
        ry0, ry1 = int(rows[0]), int(rows[-1]) + 1
        cx0, cx1 = int(cols[0]), int(cols[-1]) + 1
        solid = solid[ry0:ry1, cx0:cx1]
        crop = crop[ry0:ry1, cx0:cx1]
        Image.fromarray(crop).save(PREVIEW_DIR / f"crop_{ch}.png")
        Image.fromarray(solid).save(PREVIEW_DIR / f"mask_{ch}.png")
        masks[ch] = solid

        contours, score = extract_contours(solid)
        font_cnts, width = to_font_units(contours)
        glyphs[ch] = (contours_to_glyph(font_cnts), width)
        ious[ch] = score

        rgba = letter_rgba(crop, solid)
        Image.fromarray(rgba, "RGBA").save(GLYPH_PNG_DIR / f"{ch}.png")
        pngs[ch] = rgba
        print(f"  {ch} IoU={score:.4f} size={rgba.shape[1]}x{rgba.shape[0]} adv={width}")

        # compare: src crop | solid glyph | diff (must be empty)
        check = np.zeros_like(solid)
        cnts, hier = cv2.findContours(solid, cv2.RETR_CCOMP, cv2.CHAIN_APPROX_NONE)
        cv2.drawContours(check, cnts, -1, 255, thickness=cv2.FILLED, hierarchy=hier)
        h, sw = solid.shape
        panel = Image.new("RGB", (sw * 3 + 24, h + 36), (245, 236, 214))
        panel.paste(Image.fromarray(crop), (0, 28))
        fr = np.full((h, sw, 3), (245, 236, 214), dtype=np.uint8)
        fr[check > 0] = (8, 8, 8)
        panel.paste(Image.fromarray(fr), (sw + 8, 28))
        d = np.zeros((h, sw, 3), dtype=np.uint8)
        d[(solid > 0) & (check > 0)] = (180, 180, 180)
        d[(solid > 0) & (check == 0)] = (40, 90, 220)
        d[(solid == 0) & (check > 0)] = (220, 50, 50)
        panel.paste(Image.fromarray(d), (sw * 2 + 16, 28))
        ImageDraw.Draw(panel).text((4, 4), f"{ch}  src | exact | diff  IoU={score:.4f}", fill=(0, 0, 0))
        panel.save(PREVIEW_DIR / f"compare_{ch}.png")

    mean = float(np.mean(list(ious.values())))
    print(f"mean mask IoU: {mean:.4f}")
    return glyphs, boxes, ious, pngs, masks


def build_ttf(logo, pngs: dict[str, np.ndarray]):
    glyph_order = [".notdef", "space"] + [chr(c) for c in range(ord("A"), ord("Z") + 1)]
    tt_glyphs = {".notdef": placeholder_glyph(), "space": TTGlyphPen(None).glyph()}
    widths = {".notdef": 700, "space": int(0.35 * CAP)}

    for ch in (chr(c) for c in range(ord("A"), ord("Z") + 1)):
        if ch in logo:
            g, w = logo[ch]
            tt_glyphs[ch] = g
            widths[ch] = w
        else:
            tt_glyphs[ch] = placeholder_glyph()
            widths[ch] = 700

    fb = FontBuilder(UPM, isTTF=True)
    fb.setupGlyphOrder(glyph_order)
    fb.setupGlyf(tt_glyphs)
    fb.setupHorizontalMetrics({n: (widths[n], 0) for n in glyph_order})
    fb.setupHorizontalHeader(ascent=ASCENDER, descent=DESCENDER)
    fb.setupHead(unitsPerEm=UPM)
    fb.setupCharacterMap({32: "space", **{ord(c): c for c in glyph_order if len(c) == 1}})
    fb.setupOS2(
        sTypoAscender=ASCENDER,
        sTypoDescender=DESCENDER,
        usWinAscent=ASCENDER,
        usWinDescent=abs(DESCENDER),
        sCapHeight=CAP,
    )
    fb.setupPost()
    fb.setupNameTable(
        {
            "familyName": "Bluffet Display",
            "styleName": "Regular",
            "uniqueFontIdentifier": "BluffetDisplay-Regular-v8-perfect",
            "fullName": "Bluffet Display Regular",
            "psName": "BluffetDisplay-Regular",
            "version": "Version 8.0",
        }
    )
    fb.save(str(TTF_PATH))

    # Embed exact PNG bitmaps at native + 2x + 4x
    font = TTFont(str(TTF_PATH))
    sbix = newTable("sbix")
    sbix.version = 1
    sbix.flags = 1  # bitmaps only preferred; outlines still present as fallback
    sbix.strikes = {}

    # Native height varies ~90; use max letter height as strike ppem reference.
    native_h = max(pngs[ch].shape[0] for ch in pngs)
    for scale, ppem in ((1, native_h), (2, native_h * 2), (4, native_h * 4)):
        strike = Strike(ppem=ppem, resolution=72)
        strike.glyphs = {}
        for ch, rgba in pngs.items():
            if scale == 1:
                img = rgba
            else:
                img = cv2.resize(
                    rgba,
                    (rgba.shape[1] * scale, rgba.shape[0] * scale),
                    interpolation=cv2.INTER_NEAREST,
                )
            strike.glyphs[ch] = SbixGlyph(
                glyphName=ch,
                originOffsetX=0,
                originOffsetY=0,
                graphicType="png ",
                imageData=png_bytes(img),
            )
        sbix.strikes[ppem] = strike

    font["sbix"] = sbix
    font.save(str(TTF_PATH))
    print("Wrote", TTF_PATH, f"with sbix strikes {sorted(sbix.strikes)}")


def compose_text_from_pngs(text: str, pngs: dict[str, np.ndarray], gap: int = 6) -> Image.Image:
    """Pixel-perfect phrase from extracted letter PNGs."""
    parts = []
    for ch in text:
        if ch == " ":
            parts.append(None)
            continue
        if ch not in pngs:
            raise KeyError(ch)
        parts.append(pngs[ch])
    # space width
    space_w = max(p.shape[1] for p in pngs.values()) // 3
    height = max(p.shape[0] for p in pngs.values())
    width = 0
    for p in parts:
        width += space_w if p is None else p.shape[1] + gap
    canvas = Image.new("RGBA", (width + 8, height + 8), (245, 236, 214, 255))
    x = 4
    for p in parts:
        if p is None:
            x += space_w
            continue
        im = Image.fromarray(p, "RGBA")
        y = 4 + (height - p.shape[0]) // 2
        canvas.alpha_composite(im, (x, y))
        x += p.shape[1] + gap
    return canvas.convert("RGB")


def render_previews(rgb: np.ndarray, pngs: dict[str, np.ndarray]):
    # Perfect composited phrases (from extracted PNGs — identical letter pixels)
    line1 = compose_text_from_pngs("ALL YOU CAN", pngs, gap=4)
    line2 = compose_text_from_pngs("EAST BLUFFET", pngs, gap=4)
    alpha = compose_text_from_pngs("ABCEFLNOSTUY", pngs, gap=6)

    sheet = Image.new("RGB", (1200, 620), (245, 236, 214))
    sheet.paste(Image.fromarray(rgb[12:120, 40:960]).resize((1000, 130)), (40, 12))
    sheet.paste(line1.resize((min(1000, line1.width * 130 // line1.height), 130)), (40, 155))
    sheet.paste(Image.fromarray(rgb[148:262, 0:1024]).resize((1000, 130)), (40, 310))
    sheet.paste(line2.resize((min(1000, line2.width * 130 // line2.height), 130)), (40, 455))
    ImageDraw.Draw(sheet).text((40, 140), "banner", fill=(80, 80, 80))
    ImageDraw.Draw(sheet).text((40, 290), "exact glyphs (from banner pixels)", fill=(80, 80, 80))
    ImageDraw.Draw(sheet).text((40, 445), "banner", fill=(80, 80, 80))
    ImageDraw.Draw(sheet).text((40, 595), "exact glyphs (from banner pixels)", fill=(80, 80, 80))
    sheet.save(PREVIEW_DIR / "logo_vs_font.png")
    sheet.save(PREVIEW_DIR / "compare_sheet.png")
    alpha.save(PREVIEW_DIR / "alphabet.png")

    # A closeup = exact PNG scaled up nearest
    a = Image.fromarray(pngs["A"], "RGBA")
    a = a.resize((a.width * 5, a.height * 5), Image.NEAREST)
    bg = Image.new("RGB", (a.width + 40, a.height + 40), (245, 236, 214))
    bg.paste(a, (20, 20), a)
    bg.save(PREVIEW_DIR / "A_closeup.png")

    strips = [Image.open(PREVIEW_DIR / f"compare_{ch}.png") for ch in LOGO_CHARS]
    max_w = max(s.width for s in strips)
    total_h = sum(s.height for s in strips) + 6 * len(strips)
    stack = Image.new("RGB", (max_w, total_h), (245, 236, 214))
    y = 0
    for s in strips:
        stack.paste(s, (0, y))
        y += s.height + 6
    stack.save(PREVIEW_DIR / "compare_all_letters.png")


def verify_png_identity(pngs, masks):
    """RGBA alpha (dilated) must cover the solid mask exactly for ink."""
    print("PNG coverage check:")
    ok = True
    for ch in LOGO_CHARS:
        solid = masks[ch]
        alpha = pngs[ch][:, :, 3]
        # every solid pixel must be in PNG
        miss = int(((solid > 0) & (alpha == 0)).sum())
        print(f"  {ch} solid_miss={miss}")
        if miss:
            ok = False
    return ok


def main():
    rgb = load_banner()
    logo, boxes, ious, pngs, masks = extract_logo_glyphs(rgb)
    missing = [c for c in LOGO_CHARS if c not in logo]
    if missing:
        raise SystemExit(f"Missing: {missing}")
    if any(v < 1.0 - 1e-9 for v in ious.values()):
        raise SystemExit(f"Mask contours not perfect: {ious}")
    if not verify_png_identity(pngs, masks):
        raise SystemExit("PNG glyphs missing solid ink pixels")
    build_ttf(logo, pngs)
    render_previews(rgb, pngs)
    print("PERFECT: mask IoU=1.0000 for all logo letters; sbix PNGs embedded")
    print("OK:", "".join(sorted(logo)))
    print("Glyph PNGs:", GLYPH_PNG_DIR)
    print("Compare:", PREVIEW_DIR / "logo_vs_font.png")


if __name__ == "__main__":
    main()
