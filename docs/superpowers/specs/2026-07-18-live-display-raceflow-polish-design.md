# Live display + race-flow polish — design

**Date:** 2026-07-18  
**Status:** Approved — awaiting implementation plan  
**Related:** `docs/superpowers/specs/2026-07-12-bluffet-2026-kawaii-theme-design.md`, Event Live / RaceFlowChart  
**Scope:** Frontend-only fixes for home navigation, Bluffet live chrome, race-flow x-axis, legend contrast, and TV readability.

## Goal

Make Bluffet live pages usable on laptop and TV: home stays reachable, full event styling applies, race-flow charts stay constrained to the race window, filters stay readable, and type is large enough for display.

## Decisions (locked with user)

| Topic | Choice |
|---|---|
| Home during live race | No auto-redirect; keep **View live timing** as the most prominent top card |
| Display font | Drop Yuji Mai; use IBM Plex Sans everywhere (including titles) |
| Bluffet on live | Full chrome — cream/paper background, red accents, outlined panels |
| TV type | Baseline bump on Event Live + extra display scale in Fullscreen rotate |
| Race-flow x-axis | Cap to race `duration_minutes`; clamp “now” + extrapolations to that window |
| Legend / filters | Do not paint `.legend-item` solid Bluffet red; keep dark ink on light surfaces |

## Root cause — broken race-flow plot

While a race is `active`, RaceFlowChart sets “now” to wall-clock elapsed since `start_time` with **no duration cap**. That value becomes:

1. Horizontal dashed projection endpoints (`buildExtrapolationPoint`)
2. Chart.js x-axis `max` (`maxElapsedMinutes * 1.05`)
3. The red current-time vertical line

If the race has been `active` for many hours (or `start_time` is far in the past), elapsed can reach ~2,500+ minutes. Real taps near t≈0 collapse into a left-edge sliver; the rest of the chart is empty dashed runway. A 12h race must top out at **720** minutes; a 6h at **360**; a 90m at **90**.

## §1 — Navigation & home

- Remove `router.beforeEach` auto-redirect from `/` → `event-live` when any race is active.
- Home always renders. When live timing is available, the existing featured **View live timing** card remains the top, most prominent CTA (no new banner system).
- Update router/home tests that currently expect forced redirect.

## §2 — Bluffet live chrome & fonts

- Ensure `theme-bluffet` activates on Event Live for Bluffet by event id **or** name (already partially supported; verify live payload / store so name match works when id differs from seed constant).
- Extend `bluffet.css` so live surfaces use paper/cream background, red accents, outlined panels/shadows (parity with home featured card).
- Remove Yuji Mai usage (CSS + `@fontsource/yuji-mai` import if unused). Titles use IBM Plex Sans.
- **Legend/filters:** remove `#app.theme-bluffet .legend-item { background: var(--bluffet-red) }`. Selected/hover states use light red tint or red outline; participant labels stay dark ink. Category color dots unchanged.

## §3 — Race-flow time window

- Pass `duration_minutes` into RaceFlowChart (and ParticipantFlowChart if it shares the same live “now” path).
- Compute `axisMaxMinutes = duration_minutes` when present; otherwise fall back to max recorded (not unbounded wall-clock).
- Clamp live elapsed used for now-line and extrapolations: `min(wallClockElapsed, axisMaxMinutes)`.
- Overlap views (e.g. 12h + 6h): use the longest included duration as axis max.
- After the race ends / status not active: keep axis at duration; do not extend past finish for wall-clock drift.
- Add unit tests: active race with wall-clock elapsed ≫ duration still renders `x.max === duration` and extrapolation endpoint ≤ duration.

## §4 — TV readability

- **Baseline (Event Live):** larger page title, race tabs, category chips, chart titles/ticks, legend search/filters/items.
- **Fullscreen rotate:** additional CSS scale (or root font-size bump) on the rotator surface so chart + chrome remain legible across a room.
- Chart.js `scales` / `plugins.title` font sizes follow the same scales (not only DOM chrome).

## Out of scope

- Changing race `start_time` / activation semantics on the backend.
- New marketing layouts beyond the existing home featured card.
- Non-Bluffet event themes.

## Verification

- Home loads at `/` during an active Bluffet race; featured live card is top CTA.
- Event Live shows full Bluffet chrome; no Yuji Mai; legend text readable on light backgrounds.
- 6h / 12h / 90m charts: x-axis max equals duration; early-race data is spread across the window, not crushed left.
- Fullscreen rotate text/axes clearly larger than baseline Event Live.
- Existing RaceFlowChart / EventLive / Home / theme tests updated and green.
