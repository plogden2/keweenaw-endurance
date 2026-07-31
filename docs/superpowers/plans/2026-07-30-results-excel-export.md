# Results Excel Export Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** PIN-gated `.xlsx` download of event standings — AutoFilter tables, sheets for individual overall / team overall / each non-empty category per race.

**Architecture:** Backend `excelize` workbook built from `ResultsService` (same ranking as live). Handler `GET /api/events/:id/results.xlsx` under `adminOnly`. Event Live button downloads blob when PIN unlocked.

**Tech Stack:** Go, excelize/v2, Gin, Vue 3, Vitest

**Spec:** `docs/superpowers/specs/2026-07-30-results-excel-export-design.md`

---

### Task 1: Add excelize + BuildEventResultsWorkbook (TDD)

**Files:**
- Modify: `backend/go.mod` / `go.sum` (`github.com/xuri/excelize/v2`)
- Modify: `backend/internal/services/results_service.go` (or new `results_excel.go` next to it)
- Create/Modify: `backend/internal/services/results_excel_test.go`

- [ ] **Step 1: Failing tests** covering:
  - Bluffet-like seed (or existing test helpers): sheets include `{race} individual overall` and non-empty category sheets
  - Team sheet present only when ≥1 eligible team; omitted otherwise
  - Individual columns: Place, Racer name, Bib, Laps, Age, Gender, Team name
  - Team columns: Place, Team, Avg laps, Members
  - Voided lap does not count; karaoke does
  - Zero-lap racer included
  - Sheet names ≤ 31 chars
  - Workbook opens via excelize and has AutoFilter / table on used range

- [ ] **Step 2: Implement** `BuildEventResultsWorkbook(eventID uuid.UUID) (data []byte, filename string, err error)`:
  - Load event + races (skip cancelled)
  - For each race: overall board (enrich age/gender/team name from participants)
  - For each category with ≥1 participant on that race: category-filtered board
  - Team board if non-empty
  - Sheet name helpers with truncation/dedupe
  - Filename: slug(event.Name) + `-results-` + event date or UTC date `YYYYMMDD` + `.xlsx`

- [ ] **Step 3: Run** Docker if host cgo broken:  
  `go test ./internal/services/ -count=1 -run ResultsExcel|BuildEventResults`  
  Commit: `feat(results): build event standings excel workbook`

---

### Task 2: HTTP handler + route

**Files:**
- Modify: `backend/internal/handlers/events.go` (or new `results_export.go`)
- Modify: `backend/cmd/server/main.go` — `events.GET("/:id/results.xlsx", append(adminOnly, handlers.GetEventResultsExcel)...)`
- Modify: `backend/internal/handlers/handlers_test.go` (register route in test router)
- Modify: `specs/002-rfid-race-scanner/contracts/api-rfid-scanner.md`

- [ ] Handler: resolve event id → `BuildEventResultsWorkbook` → set headers → `c.Data(200, mime, data)`
- [ ] Tests: no auth → 401/403; with PIN JWT → 200, body starts with `PK` (zip/xlsx), Content-Disposition has `.xlsx`
- [ ] Commit: `feat(api): PIN-gated event results.xlsx endpoint`

---

### Task 3: Frontend download + EventLive button

**Files:**
- Modify: `frontend/src/services/api.ts` — `eventsApi` / new helper download with auth header (blob)
- Modify: `frontend/src/views/EventLive.vue` — Export Excel when `pinAuth.isAuthenticated`
- Modify: `frontend/src/views/EventLive.test.ts` (or create) — button visible only when authenticated; click triggers download helper
- Optionally update `docs/production-reader.md` one line under manage/export

```ts
// Suggested API helper (match existing auth interceptor / blob patterns)
export async function downloadEventResultsExcel(eventId: string): Promise<void> {
  const res = await apiClient.get(`/api/events/${eventId}/results.xlsx`, {
    responseType: 'blob',
  })
  // create object URL, <a download>, revoke
}
```

- [ ] `data-testid="export-results-excel"`
- [ ] Commit: `feat(frontend): export results excel from event live`

---

### Task 4: Verification

- [ ] Docker: `go test ./internal/services/ ./internal/handlers/ -count=1`
- [ ] `cd frontend && npm test -- --run EventLive`
- [ ] Fix failures; final commit if needed

---

## Spec coverage

| Spec item | Task |
|-----------|------|
| excelize workbook + filters | 1 |
| overall + category + team sheets | 1 |
| columns / voided / zero-lap | 1 |
| PIN endpoint | 2 |
| Event Live button | 3 |
| Dynamic non-Bluffet | 1 |
