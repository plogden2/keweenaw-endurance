# API Contract: RFID Race Scanner

**Base**: `/api` (existing Gin API)  
**Auth**: Public GETs for live views; management mutations require PIN-exchanged JWT (`Authorization: Bearer …`). Station scan ingest and karaoke on an armed station do not require re-PIN per tap.

**Transport**: Tag stream is **WebSocket** only (`GET /api/rfid/stream`).

## Auth

### POST `/api/auth/pin`
```json
{ "pin": "1738" }
```
Response `200`: `{ "token": "<jwt>", "role": "admin", "expires_at": … }`  
Errors: `401` invalid pin.

## Station configuration

### PUT `/api/stations/current`
```json
{
  "event_id": "<uuid>",
  "mode": "finish",
  "checkpoint_id": null,
  "device_id": "laptop-finish-1",
  "name": "Finish Mat A"
}
```
`mode`: `finish` (default) | `checkpoint`.

### GET `/api/stations/current`
Station config + online/offline + pending sync counts + live CSV last-written timestamp.

## RFID hardware / scan stream

### GET `/api/rfid/stream` (WebSocket)
Server → client:
```json
{ "type": "tag_read", "tag_uid": "DEMO-TAG-0001", "read_at": "2026-08-01T12:00:01-04:00", "device_id": "laptop-finish-1" }
```

### POST `/api/rfid/write-tag` (PIN)
```json
{ "participant_id": "<uuid>", "bib_id": "<uuid>", "race_id": "<uuid>", "logical_uuid": "<ignored>" }
```
Provide `participant_id` (existing) **or** `bib_id`. When `bib_id` is set, programs the chip with that bib’s UUID and returns `{ "bib_id", "tag_uid", "tag_uids" }` where `tag_uid` is the bib logical UUID. `logical_uuid` is ignored for new writes (bib id wins). Participant writes still return the participant JSON.

### Event bibs inventory

### GET `/api/events/{eventId}/bibs`
Public. Lists event bibs with `tag_count`, `tag_uids`, and optional assigned participant (`participant_id`, `participant_name`, `race_id`).

### POST `/api/events/{eventId}/bibs/bulk` (PIN)
```json
{ "from": 1, "to": 100 }
```
Ensures bibs for every integer in `[from, to]` inclusive (max span 500). Returns created/existing bib rows. Refreshes live CSV.

### GET `/api/events/{eventId}/bibs/{bibId}/tags`
Public. Active tag associations for a bib: `{ "data": [ … ] }`.

### POST `/api/events/{eventId}/bibs/{bibId}/tags` (PIN)
Empty body (or `{}`) → hardware write of bib UUID via `WriteTagForBib`.  
`{ "tag_uid": "<uid>" }` → associate without hardware write.  
Response `201`: `{ "bib_id", "tag_uid", "tag_uids" }`. Refreshes live CSV.

### POST `/api/rfid/inject` (test/dev only)
When `GO_ENV=test` or `RFID_INJECT=true`: `{ "tag_uid": "DEMO-TAG-0001" }`

## Lap timing

### POST `/api/events/{eventId}/scans`
```json
{
  "tag_uid": "DEMO-TAG-0001",
  "device_id": "laptop-finish-1",
  "local_timestamp": "2026-08-01T12:00:01-04:00"
}
```

Responses:
- `result: "lap"` — includes `placement` (overall), `placement_category`, `lap_count`, `timing_record_id`, `karaoke_available`
- `result: "test_read"` — race scheduled
- `result: "cooldown"` — `retry_after_seconds`
- `result: "unknown_tag"` — reject (no lap)

### POST `/api/timing-records/{id}/karaoke-bonus`
`201` bonus; `409` if already exists for that lap.

### GET `/api/events/{eventId}/taps`
Public, paginated timing-record audit list across all event races. Query parameters:
`page` (default `1`), `limit` (default `50`), optional `race_id`, and optional
participant `q` search (bib or name). Returns:
```json
{ "data": [{ "id": "...", "record_type": "rfid_lap", "participant": {}, "checkpoint": {} }], "total": 1, "page": 1, "limit": 50 }
```

### GET `/api/events/{eventId}/participants`
Public, paginated participant picker for the event. Query parameters: `page`
(default `1`), `limit` (default `50`), optional `q` search (bib or name).
Each participant includes its race for display labels.

### POST `/api/events/{eventId}/taps` (PIN / timerWrite)
Records a manual finish tap for an event participant. `karaoke_bonus: true`
creates a standalone bonus without `source_lap_id`.
```json
{ "participant_id": "<uuid>", "karaoke_bonus": false, "timestamp": "2026-08-01T12:00:01Z" }
```
`timestamp` is optional and defaults to the server timestamp. Returns `201` with
the timing-record JSON. The server refreshes the live CSV and emits a live
`lap_recorded` update after a successful tap.

### POST `/api/timing/records/{id}/void` (PIN / timerWrite)
Soft-voids a timing record (`voided_at` set). Voiding an `rfid_lap` cascades to its linked `karaoke_bonus`. Idempotent if already voided. Returns `{ record, cascaded_ids, lap_count, placement }`.

### POST `/api/timing/records/{id}/restore` (PIN / timerWrite)
Clears `voided_at`. Restoring karaoke while the source RFID lap is still voided → `409`. Idempotent if already active. Same response shape as void.

### POST `/api/races/{id}/start` | `/api/races/{id}/finish` (PIN)
Manual status transitions; auto-start also occurs at `start_time`.

## Live / leaderboard

### GET `/api/events/{eventId}/live`
Public. Includes races, countdowns, **overall** leaderboards (default) with category color keys, optional `category_id` filter, and series data for race-flow charts / overlap.

```json
{
  "event": { "id": "...", "name": "All You Can East Bluffet" },
  "category_legend": [
    { "key": "advanced_men", "label": "Advanced Men", "color": "#1a5276" }
  ],
  "races": [
    {
      "id": "...",
      "name": "12 Hour",
      "status": "scheduled",
      "start_time": "2026-08-01T08:00:00-04:00",
      "countdown_seconds": 3600,
      "leaderboard_overall": [
        {
          "place": 1,
          "participant_id": "...",
          "bib_number": "12",
          "name": "Alex Rivera",
          "category_key": "advanced_men",
          "laps": 14,
          "last_lap_at": "..."
        }
      ],
      "flow_series": []
    }
  ]
}
```

### GET `/api/events/{eventId}/live/stream` (WebSocket, public)

Server → client on scored lap / karaoke bonus lap bump:

```json
{
  "type": "lap_recorded",
  "event_id": "<uuid>",
  "race_id": "<uuid>",
  "participant_id": "<uuid>",
  "participant_name": "Alex Rivera",
  "bib_number": "42",
  "lap_count": 7,
  "recorded_at": "2026-07-18T16:00:00Z"
}
```

Not sent for cooldown, unknown_tag, test_read, or checkpoint-only results.

## Results export

### GET `/api/events/{eventId}/results.xlsx` (PIN)
Returns an Excel standings workbook for all non-cancelled races in the event.
Response `200` is an attachment with:

- `Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
- `Content-Disposition: attachment; filename="<event>-results-<date>.xlsx"`

## Participants (Racers)

`/api/races/{raceId}/participants`:
- `?q=` server-side filter (client also debounces)
- Default sequential bib; unique per **event** (not only per race)
- `category_id` required on create for this feature
- List includes `tag_uids[]`
- Duplicate bib across races in the same event → `400`

### GET/POST `/api/races/{raceId}/participants/{id}/tags`

## Sync

Config: `HOSTED_API_URL` (optional).

### GET `/api/rfid/sync-status`
### POST `/api/sync/push`
### POST `/api/sync/pull`

## CSV disaster recovery

Live file maintained on disk; optional copy endpoint:

### GET `/api/events/{eventId}/live-csv`
PIN optional or required per deploy; returns current live snapshot bytes (not a one-shot export job).

### POST `/api/events/{eventId}/import.csv`
PIN required. Multipart file; replace semantics for that event on this station.

See [csv-race-export.md](./csv-race-export.md).
