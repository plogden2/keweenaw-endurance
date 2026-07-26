# Discard / Remove Lap — Design

**Date**: 2026-07-26  
**Status**: Implemented (2026-07-26) — advisor-approved requirements  
**Feature branch context**: `002-rfid-race-scanner` / race-day PIN management

## Problem

Accidental RFID taps and bad manual entries cannot be corrected in the UI today. `docs/production-reader.md` documents workarounds only (manual entry for misses, post-race cleanup, CSV wipe). Volunteers need a fast, PIN-gated way to discard a scored lap without destroying auditability.

## Goals

1. After PIN unlock, discard the just-scored lap from the scan popup (confirm dialog).
2. From PIN-gated Recent Records (Live Timing), void or restore any older scored record.
3. Soft-void only (`voided_at`); restore supported; no hard delete in v1.
4. Voided rows excluded from lap counts, placement, cooldown, karaoke eligibility, team averages.
5. Cascading: voiding an `rfid_lap` also voids its linked `karaoke_bonus`.
6. Side effects: live CSV update, live stream recount signal, cooldown based on latest non-voided RFID lap.

## Non-goals

- Hard delete, per-action PIN re-prompt, station-armed requirement
- Voiding test reads / cooldown / unknown-tag feedback (no timing row)
- Auto-restore of karaoke when restoring RFID lap
- Public/spectator discard controls

## Decisions (advisor)

| Topic | Decision |
|-------|----------|
| UX | Scan popup Discard + PIN Recent Records void/restore |
| Auth | Unlocked PIN session (`timerWrite` / admin JWT) |
| Semantics | Soft-void + restore |
| Scope | `rfid_lap` (hardware + manual), `karaoke_bonus` alone |
| Karaoke | Cascade void with source RFID lap |
| Confirm | Keep / Discard dialog (not one-click, not type-to-confirm) |
| API | `POST /api/timing/records/:id/void` and `/restore` |

## API

```
POST /api/timing/records/:id/void   → timerWrite
POST /api/timing/records/:id/restore → timerWrite
```

Response:

```json
{
  "record": { "...TimingRecord including voided_at" },
  "cascaded_ids": ["..."],
  "lap_count": 7,
  "placement": 3
}
```

- Idempotent void/restore → `200`
- Restore karaoke while source RFID still voided → `409`
- Live stream: publish `type: "lap_voided"` or `"lap_restored"` with updated `lap_count` (same channel shape as `lap_recorded`)

## Data model

Add nullable `voided_at` on `timing_records` (`*time.Time`). Active = `voided_at IS NULL`. Include `voided_at` in live CSV export/import.

## UI

1. **ScanPopup** (PIN session already required to show): Discard lap → confirm → `void` → dismiss + brief toast.
2. **LiveTiming Recent Records**: when PIN unlocked, Actions column with Discard / Restore + confirm; show voided rows with visual strike/badge.

## Testing

- Backend: void cascade, restore rules, cooldown after void latest lap, scoredLapCount excludes voided, handler auth
- Frontend: ScanPopup discard confirm flow; LiveTiming actions when authenticated
- Update `docs/production-reader.md` §6
