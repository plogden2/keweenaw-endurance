# BluffetDisplay font — design

**Date:** 2026-07-25  
**Status:** Approved for implementation  
**Deliverable:** Full uppercase A–Z `.ttf`

## Goal

Recreate the ALL YOU CAN / EAST BLUFFET banner lettering as an installable display font. Prior free “chop suey” faces failed on the capital **A** (pointed apex). This face uses the banner’s flat-top, spurred **A** as the lock glyph.

## Style rules

- Blocky faux–East Asian / chop-suey display capitals
- Heavy geometric strokes (not brushy / pointed wedges)
- Recurring left-stem rectangular spur on many letters
- Flat feet; left feet may hook outward
- Solid fills only — white inline / outer outline are design effects, not glyph geometry
- Lock **A**: flat horizontal top cap, two left spurs, small rectangular counter, flat feet

## Scope

- Glyphs: A–Z + space
- Format: TrueType (`.ttf`) via fontTools
- Output: `assets/fonts/BluffetDisplay-Regular.ttf`
- Previews: alphabet sheet + banner phrase PNG

## Out of scope (v1)

- Lowercase, punctuation set, kerning pairs, variable axes
- Built-in inline/outline variants
