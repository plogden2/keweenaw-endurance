# Event Taps Editor — Design

**Date**: 2026-07-30  
**Status**: Implemented (2026-07-30) — branch `feature/event-taps-editor`  
**Replaces**: Race-scoped Manual entry / Live Timing station page as the ops taps editor

## Problem

Volunteers need one place to review, void, restore, and manually add taps across every race in an event. Today Manual entry is race-scoped (`/timing/live/:raceId`), lists all records without server pagination, uses bib/RFID + checkpoint form (not a searchable racer picker), and cannot create a standalone karaoke bonus without a source RFID lap.

## Goals

1. Event-scoped paginated taps table (default sort: tap time descending).
2. Soft-void (delete icon) and restore; voided rows always visible and grayed.
3. **Add tap** dialog: searchable racer dropdown (bib or name), karaoke toggle.
4. Karaoke toggle ON → create **only** `karaoke_bonus` with `source_lap_id = null`.
5. Karaoke toggle OFF → create scored `rfid_lap` at the race finish checkpoint.
6. PIN-gated mutations; list readable without PIN (same policy as live timing).

## Non-goals

- Hard delete
- Custom timestamp picker (v1 = now)
- Checkpoint selection in UI (server resolves finish)
- Keeping Participant Lookup / SyncStatus on this page
- Changing scan-popup linked karaoke (`POST /api/timing-records/:id/karaoke-bonus`)
- Changing reader-gui / `POST /api/rfid/manual-entry` bridge path
- Deprecating `GET /api/timing/live/:raceId` (charts still use it)

## Decisions

| Topic | Decision |
|-------|----------|
| Approach | Server-paginated event taps API |
| Route | `/events/:eventId/taps`; `/timing/live/:raceId` redirects to event taps |
| Delete | Soft-void via existing void/restore endpoints |
| Voided visibility | Always show, grayed + badge + restore |
| Karaoke manual | Standalone `karaoke_bonus`, `source_lap_id` null |
| Checkpoint | Auto finish checkpoint for participant’s race |
| Page chrome | Header + Add tap + table only |
| Auth | GET public; POST/void/restore require `timerWrite` |
| Migration | None — null `source_lap_id` already allowed |
| Page size | 50 |

## API

### List taps

```
GET /api/events/:eventId/taps?page=1&limit=50&race_id=&q=
```

Public. Response:

```json
{ "data": [ /* TimingRecord + participant (+ race name) */ ], "total": N, "page": 1, "limit": 50 }
```

- Default `limit=50`, sort `timestamp DESC`
- Include voided rows
- Optional `race_id`, `q` (bib/name filter)

### Create tap

```
POST /api/events/:eventId/taps   → timerWrite
```

Body:

```json
{
  "participant_id": "<uuid>",
  "karaoke_bonus": false,
  "timestamp": "<RFC3339 optional; default now>"
}
```

Server validates participant belongs to a race in the event, resolves that race’s finish checkpoint, creates `rfid_lap` or standalone `karaoke_bonus`. Side effects: same CSV / live-stream notify patterns as other scored creates where applicable.

### Participants for dropdown

```
GET /api/events/:eventId/participants?q=&page=&limit=
```

Public (or same as race participants list). Debounced search by bib/name; include race name on each row for disambiguation.

### Unchanged

- `POST /api/timing/records/:id/void` / `restore`
- `POST /api/rfid/manual-entry`
- `POST /api/timing-records/:id/karaoke-bonus`
- `GET /api/timing/live/:raceId`

## Data model

No schema change. Document that `source_lap_id` is optional for `karaoke_bonus`: null means standalone manual karaoke (not tied to an RFID lap). Unique index already only applies when `source_lap_id IS NOT NULL`.

## UI

**Page** `EventTaps.vue` at `/events/:eventId/taps`:

| Column | Content |
|--------|---------|
| Time | `timestamp` |
| Race | race name |
| Bib | bib number |
| Name | first + last |
| Type | Lap / Karaoke / Checkpoint |
| Sync | `sync_status` |
| Actions | delete icon → void (PIN); restore when voided (PIN) |

- **Add tap** opens dialog: searchable racer select, karaoke toggle, Submit/Cancel
- Confirm void/restore with `window.confirm` (reuse LiveTiming copy style)
- Redirect `/timing/live/:raceId` → `/events/:eventId/taps`
- Update Manual entry links: EventLive, PinUnlock, EventDetails, RaceDetails
- Retire race-scoped LiveTiming station layout as the editor (redirect only)

## Testing

- Backend: list pagination/sort; create lap + standalone karaoke; finish checkpoint; auth on POST; participant belongs to event
- Frontend: table voided styling; PIN-gated actions; Add tap dialog karaoke flag; pagination controls
- E2E (lightweight): PIN → add tap → void → restore
- Docs: `docs/production-reader.md` Manual entry → event taps URL
- Spec note: `data-model.md` karaoke `source_lap_id` optional

## Out of scope follow-ups

- Timestamp picker
- Filters UI beyond optional query params if timeboxed
- Client-side merge of race feeds (rejected approach)
