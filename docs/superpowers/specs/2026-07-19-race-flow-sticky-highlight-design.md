# Race Flow Chart Sticky Highlight — Design

## Goal

Make `RaceFlowChart` lines sticky-selectable by clicking a line/point or legend name. The highlight persists until cleared by re-clicking the selection, clicking empty plot area, or clicking outside all race-flow charts. Hover remains a temporary preview only when nothing is sticky-selected.

## Architecture

Controlled `v-model:highlightParticipantId`: parents own the sticky id; each chart keeps `hoveredParticipantId` local for preview. Emit `update:highlightParticipantId` only (`string` to select, `undefined` to clear).

In `EventLive`, all charts already share one `highlightParticipantId` ref — selecting in any chart highlights across all bound charts.

## Effective highlight

```ts
return props.highlightParticipantId ?? hoveredParticipantId.value ?? undefined
```

Sticky wins over hover (flips today's `hover ?? sticky`).

## Interactions

| Action | Result |
|--------|--------|
| Click line/point (`intersect: true`) | Emit participant id (or `undefined` if already selected → toggle off) |
| Click empty plot area | Emit `undefined` |
| Click outside **all** `.race-flow-chart` | Emit `undefined` |
| Click legend name button | Same sticky select/toggle as plot |
| Legend checkbox | Visibility only (independent) |
| Hover with no sticky | Preview highlight + `cursor: pointer` over lines |
| Hover with sticky active | No visual change from hover |
| Celebration / compare-in-flow sets prop | Overwrites manual sticky (shared ref, last-writer-wins) |

## Legend markup

Replace wrapping `<label>` with a `<div class="legend-item">`:
- Checkbox with its own `aria-label` (visibility only)
- `<button type="button">` wrapping swatch + name for sticky select
- Keep row `mouseenter`/`mousemove`/`mouseleave` for hover tooltip

## Parent wiring

- `EventLive.vue` / `RaceDetails.vue`: bind `v-model:highlight-participant-id`
- `EventLive`: `watch(highlightParticipantId, v => { if (!v) focusParticipantId.value = undefined })` so clearing sticky also clears leaderboard focus

## Scope

- In: `RaceFlowChart`, `EventLive`, `RaceDetails`, their tests
- Out: `ParticipantFlowChart`, multi-select, canvas keyboard a11y, persistence across routes

## Acceptance criteria

1. Click line/point sticky-selects; persists after mouse leave
2. Sticky active → hover other lines does not change styling
3. No sticky → hover previews; leave restores neutral
4. Re-click selected line/legend clears
5. Empty plot click clears
6. Outside all charts clears; sibling chart click does not spuriously clear
7. Legend name selects/toggles; checkbox only toggles visibility
8. Selecting a hidden participant makes it visible
9. Cursor `pointer` over line, `default` over empty
10. EventLive: select in one chart highlights all bound charts; clear clears all + `focusParticipantId`
11. Celebration overwrites manual sticky
12. `ParticipantFlowChart` unchanged

## Advisor decisions

See advisor DDR (A–L): intersect click hit-testing; outside = outside ALL charts; toggle-off on re-click; no extra emit name; force-visible via existing prop watcher.
