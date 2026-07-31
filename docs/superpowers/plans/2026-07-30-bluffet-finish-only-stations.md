# Implementation plan: Bluffet finish-only stations

Spec: `docs/superpowers/specs/2026-07-30-bluffet-finish-only-stations-design.md`

## Task 1 — Backend Bluffet policy + ManualEntry autofill

**Files:**
- `backend/internal/eventpolicy/` (new leaf: IDs + `IsBluffetEventID`)
- `backend/internal/bridgeapp/bluffet.go` (alias constants to eventpolicy if needed)
- `backend/internal/services/station_service.go` (+ tests)
- `backend/internal/services/scan/scan_service.go` — force finish mode for Bluffet at read time
- `backend/internal/services/rfid_service.go` and/or `handlers/rfid.go` (+ tests)
- `backend/internal/handlers/requests.go` — relax checkpoint_id binding

**Do:**
1. Add `eventpolicy.IsBluffetEventID` (full + short + suffix).
2. `PutCurrent`: if Bluffet and mode=checkpoint → error (map to 400).
3. `ProcessScan` / station mode: if station event is Bluffet, treat as finish even if DB says checkpoint.
4. ManualEntry: optional checkpoint_id binding; when empty and race’s event is Bluffet, resolve finish checkpoint in service; never autofill over a non-empty bad id.
5. Unit tests for PutCurrent reject, scan force-finish, ManualEntry autofill.

**Verify:** `go test` for affected packages.

## Task 2 — Frontend Station + Manual entry UI

**Files:**
- `frontend/src/utils/bluffet.ts` (or similar) — `isBluffetEventId`
- `frontend/src/views/StationConfig.vue`
- `frontend/src/components/ManualTimingForm.vue`
- `frontend/src/views/LiveTiming.vue` if needed
- `frontend/e2e/multi-station.spec.ts` — adjust per advisor

**Do:**
1. Bluffet: Station UI finish-only; save mode=finish, checkpoint_id=null.
2. Bluffet: Manual form hide checkpoint; submit without requiring picker (server autofills).
3. Fix e2e so checkpoint-mode coverage still exists off Bluffet or is correctly scoped.

**Verify:** existing frontend unit tests if present; e2e file still coherent.

## Task 3 — Reader GUI + docs

**Files:**
- `backend/cmd/reader-gui/ui.go`
- `docs/production-reader.md`

**Do:**
1. Hide checkpoint selector for Bluffet event; keep race selector / all-races finish behavior.
2. Docs: Bluffet is finish-only; no checkpoint arming/picker.

**Verify:** `go test` / build reader-gui if feasible; docs mention finish-only.
