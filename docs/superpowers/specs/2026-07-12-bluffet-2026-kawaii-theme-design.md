# Bluffet 2026 kawaii theme — design

**Date:** 2026-07-12  
**Status:** Draft — awaiting user review before implementation plan  
**Feature context:** All You Can East Bluffet event timing UI  
**Source art:** `assets/allYouCanEastBluff_26_052126.avif`

## Goal

Replace the 2026 Bluffet orange logo with the new kawaii Japanese poster artwork, and apply matching **themed chrome** across Bluffet event timing surfaces while keeping race-day UI glanceable.

## Decisions

| Topic | Choice |
|---|---|
| Scope | Full immersion in Bluffet context (home featured, event/race/live, racers, PIN, station, certificates) |
| Artwork | Full poster for heroes + cropped whole-cat avatar as logo |
| Intensity | Themed chrome (brush titles, thick outlines, sushi-flag chips) — not full decorative immersion |
| Architecture | Event theme package (CSS tokens + root class), not one-off per-view styles |
| Non-Bluffet | Unchanged default timing chrome |

## Architecture

1. **Assets** under `frontend/public/images/`:
   - `bluffet-2026-poster.avif` + `bluffet-2026-poster.png` (full poster; AVIF primary, PNG fallback; long edge ~1600–2000px)
   - `bluffet-2026-logo.png` — **replace in place** with square **512×512** crop of the whole cat (helmet through sushi tray). Keeps existing seed `logo_url` stable.
2. **Theme stylesheet** `frontend/src/themes/bluffet.css` — CSS variables and shared classes (`.theme-bluffet` / `.bluffet-theme`): tan paper, Japan red, teal, thick black outlines, offset “stamp” shadows, sushi-flag chip styles.
3. **Activation** via `useBluffetTheme()` (or equivalent):
   - Active when route/event **UUID** matches Bluffet seed id, **or** event name is exactly `All You Can East Bluffet`, **or** station is armed to that event.
   - UUID wins when present; name is fallback.
   - `App.vue` toggles `theme-bluffet` on `#app`.
   - Home featured block may use a local wrapper even when global class is off.
4. **No API/schema changes.** Seed continues to point `logo_url` at `/images/bluffet-2026-logo.png`.

## UI surfaces

| Surface | Treatment |
|---|---|
| Home featured | Full poster hero (`<picture>` AVIF+PNG); outlined red/teal CTAs; themed section chrome |
| Event / race details | Logo via `EventLogo`; brush page titles; thick-outline race panels |
| Event live / live timing | Avatar in header; red active tabs; Japan-red countdown; sushi-flag category chips |
| Racers / PIN / station | Token accents on buttons, badges, panels when Bluffet-armed — no poster takeover |
| Certificates | Cropped logo; light paper/red frame; results remain primary |
| AppHeader | Inferior Timing stays primary; when `theme-bluffet` is active, show the cropped logo mark (~28px) beside timing nav |

Motifs are CSS/SVG accents (red sun dot, thick outlines), not emoji decoration.

## Visual system

### Color tokens (lock contrast in `bluffet.css`)

- Paper / background: warm tan (~`#c4a574` / lighter panel `#f7f3ea`)
- Japan red (~`#c8102e`) — primary accent, active tabs, countdown
- Teal (~`#0d9488`) — secondary accent (jersey color)
- Ink: near-black `#1a1a1a` for outlines and body text
- Tables, timers, and PIN errors must meet **WCAG AA** against panel backgrounds

### Typography

- **Display (titles only):** [Yuji Mai](https://fonts.google.com/specimen/Yuji+Mai) — brush Latin aligned with poster energy
- **UI sans:** [IBM Plex Sans](https://fonts.google.com/specimen/IBM+Plex+Sans) 400/600/700 — tables, timers, forms
- `font-display: swap`; prefer self-hosted woffs in the theme package for offline station use
- Fallbacks: `"Segoe UI", system-ui, sans-serif`

### Motion (respect `prefers-reduced-motion: reduce`)

1. Home poster: one-shot soft scale/fade-in (~400ms ease-out)
2. Sushi-flag chips / race tabs: brief stamp-in (scale 0.92→1 + opacity) on mount; active tab short settle
3. Bluffet scan popup: thick-outline panel stamp (~200ms) instead of generic fade

Do **not** animate standings rows or poll-driven chart updates.

## Data flow

```
route eventId / event name / station.event
        ↓
 useBluffetTheme() → { active, posterSrc, logoSrc }
        ↓
 #app.theme-bluffet  +  EventLogo(logo_url)  +  Home <picture> poster
```

Poster path is a frontend constant (not stored in DB).

## Error handling & fallbacks

- Missing logo/poster: no broken mandatory UI; theme tokens still apply
- Detection miss / unarmed station: default chrome (no `theme-bluffet`)
- AVIF unsupported: PNG poster via `<picture>`
- Font failure: system sans; layout remains usable
- Offline station: tokens + layout readable without webfonts/poster

## Testing (minimum; no visual snapshots)

- **Unit:** composable activates on seed UUID or exact name; inactive on mismatch; exposes expected class + poster path constants
- **Component:** themed views assert `theme-bluffet` when mocked Bluffet; logo `src` matches seed path; poster `<picture>` has AVIF + PNG; missing `logo_url` does not render a broken required image
- **E2E (1–2):** Bluffet live or home featured has theme class; non-Bluffet never does; asset URLs return OK
- **Skip:** Percy/Chromatic, CSS snapshot diffs, font-render checks

## Out of scope

- Theming unrelated events or global app chrome outside Bluffet context
- Redesigning RFID/Proxmark behavior or timing algorithms
- Full immersive decorative mode (intensity C)
- Keeping the old orange logo on any Bluffet surface

## Risks

- Tan + red/teal contrast must be validated on live tables and PIN errors
- Certificate print: logo crop sharp at ~64–96px; frame must not clip content
- Name-string drift vs UUID — both keys required; UUID preferred
- Home currently hardcodes old logo path — migrate in the same change set as asset swap
