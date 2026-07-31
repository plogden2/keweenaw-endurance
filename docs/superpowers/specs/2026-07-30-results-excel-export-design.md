# Results Excel Export — Design

**Date**: 2026-07-30  
**Status**: Approved (advisor defaults; user chose overall + category sheets)  
**Branch context**: `main` / race-day ops

## Problem

Organizers need a readable Excel workbook of standings for printing/sharing: sortable tables with racer demographics, one sheet per board (overall, category, team), matching live scoring rules.

## Goals

1. Download `.xlsx` for an event with AutoFilter tables (sortable by every column).
2. Individual sheets: Place, Racer name, Bib, Laps, Age, Gender, Team name.
3. Team sheets: Place, Team, Avg laps, Members (one row per team).
4. Sheets: per race — individual overall, team overall (if any), each non-empty category.
5. Same lap/voided/tie-break rules as live boards.
6. PIN-gated; button on Event Live when unlocked.

## Non-goals

- Spectator/public Excel download
- Changing live CSV recovery format
- Per-racer roster dump on team sheets
- Excel charts or fancy formatting beyond Table + AutoFilter + header row

## Decisions (advisor)

| Topic | Decision |
|-------|----------|
| Approach | Server-side `excelize` |
| Auth | `adminOnly` (PIN JWT), like live-csv |
| Team rows | One row per team (not expanded racers) |
| Empty sheets | Skip empty category / empty team boards |
| Zero-lap | Include (match live) |
| Voided | Exclude from lap counts |
| Kids cats | Emit Men/Women when non-empty |
| Events | Dynamic for any event (not Bluffet-hardcoded) |
| UI | Event Live “Export Excel” when PIN unlocked |

## Sheet naming (≤31 chars)

Pattern: `{race short} individual overall` | `{race short} team overall` | `{race short} {category}`

Bluffet examples: `12 hour individual overall`, `12 hour Intermediate Men`, `6 hour team overall`, `90 min kids Women`.

Race short: lowercase race name, trim obvious suffix words as needed to fit (`90-Minute Kids` → `90 min kids`). Truncate/dedupe if custom names exceed limit.

## API

```
GET /api/events/:id/results.xlsx   → adminOnly
```

- `Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
- `Content-Disposition: attachment; filename="{event-slug}-results-{YYYYMMDD}.xlsx"`
- Body: workbook bytes

## Data rules

Reuse `ResultsService` ranking:
- Individual: non-voided `rfid_lap` + `karaoke_bonus`; sort laps desc, earliest last RFID lap, 0-lap by bib.
- Category sheets: same board filtered to that category (participant `category_id` / existing filter semantics used by live).
- Team: `BuildTeamLeaderboard` (≥2 members, avg laps).

Enrich rows from participants (+ team name) for age/gender.

## UI

On `EventLive.vue`, when `pinAuth.isAuthenticated`: **Export Excel** button → authenticated GET → browser download blob. Keep CSV recovery separate.

## Testing

- Unit: sheet inventory + places/laps vs ResultsService (voided excluded; karaoke counted; skip empty team sheets).
- Handler: 401 without PIN; 200 + xlsx signature with PIN.
- Optional light e2e: unlock → export downloads.

## Implementation sketch

1. Add `github.com/xuri/excelize/v2`
2. `ResultsService.BuildEventResultsWorkbook(eventID) ([]byte, filename string, error)`
3. Handler + route
4. Frontend API helper + EventLive button
5. Tests
