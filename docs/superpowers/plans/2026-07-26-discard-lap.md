# Discard / Remove Lap Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** PIN-unlocked volunteers can soft-void (and restore) scored timing records from the scan popup and Recent Records, with cascade karaoke, cooldown/leaderboard/CSV/live side effects.

**Architecture:** Add `voided_at` on `TimingRecord`. New `TimingService.VoidRecord` / `RestoreRecord` (+ handlers under `timerWrite`). All scored-lap queries exclude `voided_at IS NULL`. ScanPopup emits discard; App.vue calls API; LiveTiming shows void/restore when PIN unlocked.

**Tech Stack:** Go/GORM, Gin, Vue 3, Vitest, existing PIN JWT `timerWrite`

**Spec:** `docs/superpowers/specs/2026-07-26-discard-lap-design.md`

---

### Task 1: Model `voided_at` + TimingService void/restore (TDD)

**Files:**
- Modify: `backend/internal/models/models.go` (`TimingRecord`)
- Modify: `backend/internal/services/timing_service.go`
- Modify: `backend/internal/services/timing_service_test.go`
- Modify: `backend/internal/services/csv_export.go` (column `voided_at`)

- [x] **Step 1: Add failing tests** in `timing_service_test.go` for:
  - Void RFID lap sets `voided_at`, cascades karaoke via `source_lap_id`
  - Void karaoke alone leaves RFID active
  - Idempotent void
  - Restore clears `voided_at`
  - Restore karaoke while source voided → error
  - Idempotent restore

- [x] **Step 2: Add field** to `TimingRecord`:
```go
VoidedAt *time.Time `gorm:"type:timestamp" json:"voided_at,omitempty"`
```

- [x] **Step 3: Implement** `VoidRecord(id)` / `RestoreRecord(id)` returning `(record *TimingRecord, cascaded []uuid.UUID, err error)`. Use transaction. Errors: `ErrTimingRecordNotFound`, `ErrKaraokeSourceStillVoided` (new).

- [x] **Step 4: CSV** — append `voided_at` to timing CSV headers/rows and parse on import (RFC3339 or empty).

- [x] **Step 5: Run** `go test ./internal/services/ -count=1 -run Timing` — pass. Commit.

---

### Task 2: Exclude voided from scoring + cooldown + karaoke

**Files:**
- Modify: `backend/internal/services/scan/scan_service.go` (`cooldownRemaining`, `scoredLapCount`, placement queries)
- Modify: `backend/internal/services/results_service.go` (same `record_type IN` queries)
- Modify: `backend/internal/services/scan/karaoke_service.go` (reject voided source; existing karaoke check ignore voided)
- Modify: team scoring paths if they query timing rows directly
- Test: extend `scan_service_test.go` — void latest lap clears cooldown; lap count drops

- [x] Add `AND voided_at IS NULL` (or `Where("voided_at IS NULL")`) on every scored-lap query.
- [x] Karaoke: if `source.VoidedAt != nil` → `ErrInvalidSourceLap`; existing karaoke lookup only non-voided.
- [x] Run scan + results tests. Commit.

---

### Task 3: HTTP handlers + routes + live stream

**Files:**
- Modify: `backend/internal/handlers/timing.go` — `VoidTimingRecord`, `RestoreTimingRecord`
- Modify: `backend/cmd/server/main.go` — register routes
- Modify: `backend/internal/handlers/handlers_test.go` — auth + cascade
- Modify: `backend/internal/services/live_stream_hub.go` — allow `lap_voided` / `lap_restored` types (same struct)
- Modify: `specs/002-rfid-race-scanner/contracts/api-rfid-scanner.md`

```go
// POST /api/timing/records/:id/void
// POST /api/timing/records/:id/restore
// both append(timerWrite, ...)
```

Response JSON: `record`, `cascaded_ids`, `lap_count`, `placement` (compute via ScanService helpers or Timing+Results). Publish live event after success. Call CSV notify if existing create paths do.

- [x] Tests require PIN JWT; unauth → 401/403.
- [x] Commit.

---

### Task 4: Frontend API + ScanPopup discard

**Files:**
- Modify: `frontend/src/types/models.ts` — `voided_at?: string | null`
- Modify: `frontend/src/services/api.ts` — `voidRecord` / `restoreRecord`
- Modify: `frontend/src/components/ScanPopup.vue` + `.test.ts`
- Modify: `frontend/src/App.vue` — handle `@discard`

- [x] ScanPopup: when `result==='lap'` && `timing_record_id`, show Discard; confirm dialog Keep/Discard; emit `discard` on confirm.
- [x] App.vue: call `timingRecordsApi.voidRecord`, clear scan, optional toast.
- [x] Vitest pass. Commit.

---

### Task 5: LiveTiming Recent Records void/restore UI

**Files:**
- Modify: `frontend/src/views/LiveTiming.vue` (+ test if exists)
- Modify: `docs/production-reader.md` §6

- [x] When `pinAuth.isAuthenticated`, Actions column: Discard (active) / Restore (voided) with confirm.
- [x] Voided rows visually marked.
- [x] Docs table updated to point at UI.
- [x] Commit.

---

### Task 6: Verification

- [x] `go test ./internal/services/ ./internal/handlers/ ./internal/services/scan/ -count=1`
- [x] `cd frontend && npm test -- --run ScanPopup LiveTiming` (or full unit suite if fast)
- [x] Fix failures. Final commit if needed.
