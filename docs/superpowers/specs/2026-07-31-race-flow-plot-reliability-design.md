# Race-flow plot reliability — design

**Date:** 2026-07-31  
**Status:** Approved via advisor (user directed: consult advisor, do not stop until green)

## Problem

Users experience broken “hover pan/zoom,” janky plots, and suspected mobile crashes. Investigation found Chart.js race-flow charts with **no pan/zoom plugin**; touch/hover ran expensive polyline hit-tests and frequent `chart.update`, and `ParticipantFlowChart` still destroy/recreated on live ticks (iOS OOM risk).

## Decisions

1. **Keep Chart.js 4.5.1** — pain is interaction/perf, not the library.
2. **No `chartjs-plugin-zoom` / freehand pan.** Ship **toolbar X zoom** (Zoom in/out, Last 60 min, Reset).
3. **Desktop:** hover preview + click sticky. **Mobile/coarse:** tap sticky only; no hover dimming.
4. **Perf:** RAF-coalesce hit-test; cache polylines; update only when highlight id changes; soft-cap 80 visible datasets; `animation: false`; DPR ≤ 2.
5. **Crash prevention:** in-place live updates on both charts; keep single-tab mount patterns.

## Acceptance

Sticky highlight (legend + canvas) works desktop + mobile; toolbar zoom reliable; hover not janky; no destroy on live ticks; Playwright e2e green for legend/canvas sticky, zoom, overlap, participant chart, mobile viewport.

## Implementation notes

- Chart.js instances use `shallowRef` + `markRaw` — deep Vue proxies caused `chart.update()` stack overflows (broken sticky/hover).
- Toolbar X zoom only (no freehand pan plugin).
- RAF-coalesced hover hit-test + polyline cache; coarse pointer skips hover dimming.
- Soft-cap 80 painted datasets; `animation: false`; DPR ≤ 2.
- E2E: `frontend/e2e/race-flow-chart.spec.ts` (API-mocked).
