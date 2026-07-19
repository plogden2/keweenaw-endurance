# Fullscreen rotator split handle — design

**Date:** 2026-07-19  
**Status:** Approved  
**Scope:** Frontend only — fullscreen rotate in Event Live

## Goal

Let operators drag a handle between the race-flow plot and leaderboard in Fullscreen rotate to adjust relative pane widths.

## Decisions

| Topic | Choice |
|---|---|
| Where | Fullscreen rotator only (not tabbed Event Live layout) |
| Mechanism | CSS grid + pointer drag handle |
| Default | Keep current ~`1.1fr / 1fr` feel (~52% flow / 48% leaderboard) |
| Clamp | Neither pane below ~25% of grid width |
| Persist | `sessionStorage` key for tab-session recall |
| Chart | Rely on Chart.js responsive host resize |

## Behavior

1. Vertical drag handle between Race flow and Leaderboard (`data-testid="rotator-split-handle"`).
2. Pointer down → move → up updates left (flow) width percent; CSS variable `--fs-flow-width` drives `grid-template-columns`.
3. Hit target ~8–12px; `cursor: col-resize`.
4. On open, restore prior ratio from `sessionStorage` if valid; else default ~52.
5. Keyboard optional: not required for v1 (pointer/TV grab is primary).

## Out of scope

- Vertical (row) split
- Non-fullscreen layouts
- New dependencies

## Verification

- Handle present only when fullscreen rotator is open.
- Drag changes relative widths; neither pane collapses past clamp.
- Refresh within same tab restores ratio.
- Plot remains usable after resize (canvas host still sized correctly).
