# Leaderboard class / gender filters

**Date:** 2026-08-01  
**Status:** Approved for planning  
**Surfaces:** Event Live (individuals) + Race Details leaderboard

## Goal

Add clearer filters for **Expert / Intermediate** and **Men / Women** on individual leaderboards so spectators and organizers can view category podiums without leaving the board. Fullscreen live rotator stays overall-only.

## Decisions (locked)

| Topic | Choice |
|-------|--------|
| Interaction | Two-axis chips: Class + Gender |
| Selection | Single-select per axis (radio semantics) |
| Places when filtered | Renumber **1…n** within the filtered set |
| Scope | Event Live individuals + Race Details |
| Exclude | Fullscreen rotator, Teams mode, race-flow chart, backend |
| Implementation | Shared **client-side** filter on existing board data |
| Cross-tab (Event Live) | Same filter selection persists when switching 12h / 6h / 90min |

## Chip UI

Two rows above the individuals table (reuse Individuals/Teams pressed-chip styling):

1. **Class:** All | Expert | Intermediate  
2. **Gender:** All | Men | Women  

Defaults: both **All** (overall board).

### Matching

- Entries carry `category_key` (e.g. `expert_men`, `intermediate_women`, kids `men` / `women`).
- Filter is **AND** across axes:
  - Class All → any skill (or keys with no skill facet)
  - Gender All → any gender
  - Class Expert + Gender Women → keys that are expert **and** women
- **90-Minute Kids:** show **Gender** row only (no Class chips).
- Derive which chips to offer from keys on the current board and/or `category_legend`. Prefer hiding Class when no expert/intermediate keys are present.
- Unknown keys: still match gender via `_men` / `_women` suffix when present; otherwise only appear under All/All.

### Visibility

- Event Live: show filters only when `leaderboardMode === 'individuals'` and rotator is **not** open (do not mount filters inside fullscreen rotator panels).
- Race Details: show above the individual leaderboard.
- Empty filtered set: empty table + short “No racers match” message.

## Places

1. Start from the existing overall-ordered list.  
2. Keep relative order of matching rows.  
3. Assign `place` 1…n in the filtered list.  
4. Do not show overall place numbers while a non-All filter is active.

## Architecture

| Piece | Role |
|-------|------|
| `frontend/src/utils/leaderboardCategoryFilter.ts` | Parse `category_key` → `{ skill?, gender? }`; `matchesFilter`; `filterLeaderboard(entries, filter)` with renumbered places |
| `frontend/src/components/LeaderboardCategoryFilters.vue` | Presentational chips; props for available facets + selected values; emit updates; `data-testid` hooks |
| `EventLive.vue` | Hold filter state; apply util to active race `leaderboard_overall` for non-fullscreen individuals table |
| `RaceDetails.vue` | Same component + util on local leaderboard rows |
| `types/models.ts` | Add optional `category_key?: string` on `LeaderboardEntry` (backend already returns it; frontend type is missing it today) |

No API / `category_id` query changes for this feature. The timing leaderboard payload already includes `category_key` when the participant has a category.

**Place field:** Event Live uses `place`; Race Details uses `position`. The shared util renumbers whichever numeric rank field the caller provides (or returns a parallel `filteredPlace` / mapped list so each view keeps its existing field name).

### Filter state shape

```ts
type SkillFilter = 'all' | 'expert' | 'intermediate'
type GenderFilter = 'all' | 'men' | 'women'

interface LeaderboardCategoryFilter {
  skill: SkillFilter
  gender: GenderFilter
}
```

## Testing

- Unit: parse facets; AND match; renumber; kids gender-only keys; All/All passthrough.
- Component: chip press updates selection; All resets axis.
- EventLive: filtering changes visible individuals rows; teams mode and fullscreen rotator paths remain unfiltered.
- RaceDetails: filtering changes visible rows.

## Out of scope

- Multi-select within an axis  
- Server-side `category_id` filtering  
- Filtering team leaderboards  
- Changing category legend into the filter control (legend may remain display-only)  
- Persisting filters across browser sessions (in-memory / page session only is fine)

## Success criteria

- On Event Live (not fullscreen) and Race Details, organizers can isolate Expert/Intermediate × Men/Women with two chip rows.
- Filtered places read as category standings (1…n).
- Fullscreen rotator still shows the full overall board with no filter UI.
