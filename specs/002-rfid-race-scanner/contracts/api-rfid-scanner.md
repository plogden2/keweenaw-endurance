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
{ "participant_id": "<uuid>", "tag_uid": "<uid>" }
```
Creates `rfid_tag_associations` (multi-tag; no revoke).

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

## Participants (Racers)

`/api/races/{raceId}/participants`:
- `?q=` server-side filter (client also debounces)
- Default sequential bib; unique per race
- `category_id` required on create for this feature
- List includes `tag_uids[]`

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
