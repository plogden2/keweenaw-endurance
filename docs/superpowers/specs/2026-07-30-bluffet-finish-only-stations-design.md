# Bluffet finish-only stations (no checkpoint UI)

**Date:** 2026-07-30  
**Status:** Approved for implementation  
**Scope:** All You Can East Bluffet 2026 only

## Goal

Bluffet has a single start/finish lap point (one station). Remove checkpoint-mode options and checkpoint pickers for this event so operators cannot misconfigure intermediate checkpoint mode. Manual lap entry auto-uses each race’s finish (“Lap Check”) checkpoint.

## Non-goals

- Do not delete `timing_checkpoints` rows or the finish checkpoint model (RFID/manual scoring still need finish checkpoint IDs).
- Do not remove checkpoint station mode for other/future events.
- Do not change RFID finish-mode scan scoring behavior.

## Event identity

Treat as Bluffet when event id is any of:

- Full: `1441674d-a011-471a-a601-722b88b117f5`
- Short: `b117f5` (and case-insensitive / suffix match already used in bridgeapp)

**Advisor decisions (2026-07-30):**

- Go helper lives in a new leaf package `backend/internal/eventpolicy` (cannot put in `bridgeapp` — import cycle with `services`). `bridgeapp` may alias constants to it.
- Frontend helper: `frontend/src/utils/bluffet.ts`, must match short IDs (`b117f5`) as returned by API JSON, not only the full UUID.
- ManualEntry autofill: when `checkpoint_id` empty and race’s event is Bluffet (derive from race row; request has no event_id).
- Relax `checkpoint_id` binding globally (optional string); non-Bluffet still requires a checkpoint in the service.
- Force finish mode at **scan read time** for Bluffet stations (`ProcessScan` / station mode), so an already-armed checkpoint station cannot keep mis-scoring mid-event.
- `multi-station.spec.ts` checkpoint test: `test.skip` for Bluffet with comment; cover out-of-order in Go if needed.

## Behavior

### Station arming (Bluffet)

- UI (`StationConfig.vue`): hide mode toggle and checkpoint picker; always save `mode=finish`, `checkpoint_id=null`.
- API (`StationService.PutCurrent`): reject `mode=checkpoint` for Bluffet with HTTP 400 and a clear error.

### Manual lap entry (Bluffet)

- Website (`ManualTimingForm` / `LiveTiming`): hide checkpoint control; do not require operator selection.
- Reader GUI: hide “Checkpoint (manual)” for Bluffet; when a single race is selected, keep autofill of that race’s finish checkpoint.
- Backend (`RFIDService.ManualEntry` and/or handler): if Bluffet (or request race belongs to Bluffet) and `checkpoint_id` is omitted/nil, resolve the race’s finish checkpoint (`checkpoint_type=finish`). Still 400 if no finish checkpoint exists.

### RFID scans

Unchanged: finish-mode station → `ProcessScan` → race’s finish checkpoint → `rfid_lap`.

### Other events

Checkpoint mode and checkpoint pickers remain available.

## Surfaces

| Area | Change |
|------|--------|
| `frontend/src/views/StationConfig.vue` | Bluffet finish-only UI |
| `frontend/src/components/ManualTimingForm.vue` + `LiveTiming.vue` | Hide checkpoint for Bluffet |
| `frontend` shared helper (small) | `isBluffetEventId(id)` |
| `backend/.../station_service.go` (+ handler) | Reject checkpoint mode for Bluffet |
| `backend/.../rfid` ManualEntry path | Autofill finish checkpoint |
| `backend/cmd/reader-gui/ui.go` | Hide checkpoint picker for Bluffet |
| `docs/production-reader.md` | Document finish-only Bluffet ops |
| Tests | Unit: station reject + manual autofill; keep generic checkpoint e2e on non-Bluffet fixture if needed |

## Errors

- Bluffet + `mode=checkpoint` → `400` (“All You Can East Bluffet supports finish station only”).
- Manual entry missing finish checkpoint for race → `400`.

## Testing

- Go unit tests for `PutCurrent` Bluffet reject and ManualEntry autofill.
- Frontend: light unit/component coverage if existing patterns allow; otherwise rely on Go + targeted e2e.
- Do not break `frontend/e2e/multi-station.spec.ts` checkpoint coverage — point it at a non-Bluffet event or skip Bluffet-only assertion.

## Rollout

Safe mid-event: UI/API policy only; no migration. Existing finish-armed station remains valid.
