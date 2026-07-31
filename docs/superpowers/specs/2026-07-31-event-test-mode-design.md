# Event Test Mode — Design

**Date:** 2026-07-31  
**Status:** Approved (advisor)  
**Surface:** EventLive ops bar → full-screen dialog

## Goal

Add a per-event **Test mode** for hardware/checkout practice. While open, the station intercepts RFID taps and manual bib entries into an in-memory combined race-flow + leaderboard (all races/categories). Closing discards everything; production scores are never written.

## Decisions

| Topic | Choice |
|-------|--------|
| Persistence | Frontend-only in-memory (Pinia). Discard on close / refresh |
| Entry | EventLive ops bar button “Test mode”, PIN-authenticated only |
| Intercept | Frontend choke point before `enqueueScan` / EventTaps create; no backend scoring |
| Identity | Resolve from event participant roster (`tag_uids` / `rfid_tag_uid` / bib) loaded on open |
| Board | One combined leaderboard + one race-flow plot; start empty |
| Cooldown | None (rapid taps allowed) |
| Karaoke / void | Out of scope (taps-only; close clears all) |
| Multi-station | This browser/station only; banner warns other readers still score |
| ScanPopup | Suppressed while dialog open; feedback shown in-dialog |
| Manual | Bib form inside dialog; EventTaps submit blocked for same event with toast |

## UX

1. PIN-unlocked operator on `/events/:eventId/live` clicks **Test mode**.
2. Full-screen dialog: station-local banner, last-tap feedback, bib input, race flow chart, combined leaderboard.
3. RFID / manual entries update the in-memory board only.
4. **Close & discard** (confirm if any taps) clears state and restores normal scoring.

## Data flow

```
RFID tag_read → useReaderStation
  → if test mode open for station.eventId:
       resolve participant from roster → store.recordTap → (no enqueueScan)
  → else: existing enqueueScan path

Manual bib (dialog) → resolve via roster / participants API → store.recordTap

EventTaps submit → if test mode open for event → block + toast
```

## Files

| Action | Path |
|--------|------|
| Create | `frontend/src/stores/eventTestMode.ts` (+ spec) |
| Create | `frontend/src/utils/eventTestModeFlow.ts` (+ spec) |
| Create | `frontend/src/components/EventTestModeDialog.vue` (+ test) |
| Create | `frontend/e2e/event-test-mode.spec.ts` |
| Modify | `useReaderStation.ts`, `App.vue`, `EventLive.vue`, `EventTaps.vue`, `RaceFlowChart.vue` |

## Acceptance criteria

1. Authenticated operator can open Test mode from EventLive.
2. While open, RFID/manual on that station do not create TimingRecords or change live standings.
3. Combined board shows taps across races/categories; starts empty; updates live.
4. No 60s cooldown in test mode.
5. ScanPopup suppressed; in-dialog feedback.
6. Close discards; reopen starts empty.
7. Banner states station-local scope.
8. Automated tests cover store + intercept + e2e non-pollution.

## Out of scope

- Backend dry-run / multi-station shared practice board
- Karaoke bonus / per-lap discard in test mode
- Seeding production live laps into the test board
