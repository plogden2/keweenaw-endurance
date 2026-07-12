# Data Model: RFID Race Scanner

**Feature**: `002-rfid-race-scanner` | **Date**: 2026-07-12

Extends existing entities in `backend/internal/models` and Postgres schema.

## Entity Relationship (logical)

```text
Event 1──* Race 1──* Participant 1──* RFIDTagAssociation
                │         └──* TimingRecord (rfid_lap | karaoke_bonus | checkpoint_pass)
                ├──* Category
                └──* TimingCheckpoint
Event 1──* ReaderStation (logical config; may be local-only + synced)
TimingRecord (karaoke_bonus) ──> TimingRecord (source rfid_lap)
```

## Entities

### Event
| Field | Rules |
|-------|--------|
| id, name, event_date, status, … | Existing |
| Reader stations bind to event, not a single race |

### Race
| Field | Rules |
|-------|--------|
| race_type | `lap_based` for this feature |
| duration_minutes | 720 / 360 / 90 for demo |
| start_time | America/Detroit wall time stored as timestamptz |
| status | `scheduled` → `active` (at `start_time` auto, or PIN start) → `finished` / `cancelled` |
| Pre-start | `scheduled` + before start_time → countdown + test reads only |
| Active | scored RFID laps allowed; auto-start at start_time |

### Category
| Field | Rules |
|-------|--------|
| name | e.g. Intermediate Men, Advanced Women, Men, Women |
| category_type / gender_filter | Existing fields; skill bands via name or `custom` |
| Demo | 12h/6h: Intermediate×Men/Women + Advanced×Men/Women; kids: Men/Women |

### Participant (Racer)
| Field | Rules |
|-------|--------|
| id | Stable UUID; written to RFID tags |
| race_id | Enrollment in one race |
| bib_number | Unique per race; default sequential on create |
| first_name, last_name, gender, … | Existing |
| rfid_tag_uid | Optional legacy/primary display field; not sole association |
| category_id | **Required** FK to Category for this feature (seed + UI assign skill×gender / kids gender) |
| UX term | “Racer” in UI = Participant in API |

### RFIDTagAssociation (new)
| Field | Rules |
|-------|--------|
| id | UUID PK |
| participant_id | FK, required |
| tag_uid | Unique globally (or per event); required |
| created_at | |
| active | Always true in v1 (no revoke UI/API) |

**Validation**: Programming a tag creates/updates association; lookup by tag_uid resolves participant; multiple rows per participant allowed.

### TimingCheckpoint
| Field | Rules |
|-------|--------|
| checkpoint_type | `start` / `finish` / `intermediate` |
| Finish-mode stations | Map to race finish / lap checkpoint |
| Checkpoint-mode stations | Bind to a specific checkpoint + sequence order |

### TimingRecord (extended)
| Field | Rules |
|-------|--------|
| existing fields | participant_id, checkpoint_id, timestamp, local_timestamp, device_id, sync_status |
| record_type (new) | `rfid_lap` \| `karaoke_bonus` \| `checkpoint_pass` |
| source_lap_id (new) | Nullable FK to timing_records; required for karaoke_bonus |
| station_id (new) | Optional; which reader produced the event |

**Validation**:
- Scored `rfid_lap` only if participant’s race is `active`
- Cooldown: reject new `rfid_lap` if prior scored `rfid_lap` for same participant within 60s
- At most one `karaoke_bonus` per `source_lap_id`
- Placement **default**: combined overall across categories in the race; UI color-codes by category with legend; category filter optional
- Placement sort: count `rfid_lap` + `karaoke_bonus`; tie-break earliest last lap timestamp

### ReaderStation (new)
| Field | Rules |
|-------|--------|
| id | UUID / stable device id string |
| event_id | Required |
| mode | `finish` (default) \| `checkpoint` |
| checkpoint_id | Required when mode=checkpoint |
| sequence_order | For checkpoint courses |
| name | Display label |
| last_seen_at | Heartbeat |

### CheckpointProgress (new or derived)
| Field | Rules |
|-------|--------|
| participant_id + race_id | |
| last_checkpoint_id / bitmap / ordered list | Advance on in-order passes; completing sequence creates `rfid_lap` |

May be derived from `checkpoint_pass` records rather than a separate table if simpler.

### OrganizerPinSession (logical)
Not a durable table required: PIN → JWT via auth service; config `ORGANIZER_PIN`.

### LiveCSVSnapshot (filesystem, not a table)
| Aspect | Rules |
|--------|--------|
| Path | Station-local file per event (e.g. `data/events/{eventId}/live-snapshot.csv`) |
| Update | Rewrite/append-consistent snapshot after relevant mutations |
| Purpose | Disaster recovery without manual export; import on replacement laptop |

## State Transitions

### Race.status
```text
scheduled --(start_time reached | PIN start)--> active --(duration elapsed | PIN finish)--> finished
    \-------------------------------------------------------> cancelled
```

### Tap handling
```text
tag read → resolve association → load participant+race
  if race.scheduled → test_read (no timing row of type rfid_lap)
  if race.active + finish mode + cooldown ok → rfid_lap + popup + sound
  if race.active + finish mode + cooldown fail → reject
  if race.active + checkpoint mode → checkpoint_pass / progress; maybe complete lap
  if unknown tag → unknown feedback
```

## Seed: All You Can East Bluffet 2026

| Item | Value |
|------|--------|
| Event | All You Can East Bluffet, 2026-08-01 |
| Races | 12 Hour, 6 Hour, 90-Minute Kids |
| Starts | 08:00, 08:00, 15:00 America/Detroit |
| Categories | per spec clarification |
| Participants | 100 total across races |
| Tags | Optional demo UIDs for e2e (`DEMO-TAG-0001` …) |

Align/replace current 5-race Bluffet seed generator output.
