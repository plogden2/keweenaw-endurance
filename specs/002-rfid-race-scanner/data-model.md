# Data Model: RFID Race Scanner

**Feature**: `002-rfid-race-scanner` | **Date**: 2026-07-12

Extends existing entities in `backend/internal/models` and Postgres schema.

## Entity Relationship (logical)

```text
Event 1──* Bib 1──* RFIDTagAssociation
Event 1──* Race 1──* Participant
                │         └──* TimingRecord (rfid_lap | karaoke_bonus | checkpoint_pass)
                ├──* Category
                └──* TimingCheckpoint
Participant.bib_number + Event ──> Bib   (at most one holder per bib in the event)
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
| name | e.g. Intermediate Men, Expert Women, Men, Women |
| category_type / gender_filter | Existing fields; skill bands via name or `custom` |
| Demo | 12h/6h: Intermediate×Men/Women + Expert×Men/Women; kids: Men/Women |

### Bib (event inventory)
| Field | Rules |
|-------|--------|
| id | UUID PK — written to RFID chips on new Proxmark3 writes |
| event_id | FK, required |
| bib_number | Unique within event (`UNIQUE (event_id, bib_number)`) |
| created_at | |

**Validation**: Bulk-create ensures a Bib for every integer in a range. Assignment of a participant bib number EnsureBibs the matching event Bib.

### Participant (Racer)
| Field | Rules |
|-------|--------|
| id | Stable UUID; kept for identity and **legacy** chip dual-resolve (not the primary new write payload) |
| race_id | Enrollment in one race |
| bib_number | Unique per **event** (validated across all races); default sequential on create |
| first_name, last_name, gender, … | Existing |
| rfid_tag_uid | Optional legacy/primary display field; not sole association |
| category_id | **Required** FK to Category for this feature (seed + UI assign skill×gender / kids gender) |
| UX term | “Racer” in UI = Participant in API |
| tag_uids | Optional list exposed via the racer’s current Bib associations |

### RFIDTagAssociation
| Field | Rules |
|-------|--------|
| id | UUID PK |
| bib_id | FK → Bib (replaces former `participant_id`) |
| tag_uid | Unique globally; required |
| created_at | |
| active | Always true in v1 (no revoke UI/API) |

**Validation**: Programming a tag creates/updates association on the bib; lookup by `tag_uid` resolves Bib then Participant by `(event_id, bib_number)`; multiple rows per bib allowed. Legacy chips may still carry a participant UUID and dual-resolve when no association matches.

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
| source_lap_id (new) | Optional FK to timing_records for `karaoke_bonus`; null denotes a standalone manual karaoke tap |
| station_id (new) | Optional; which reader produced the event |

**Validation**:
- Scored `rfid_lap` only if participant’s race is `active`
- Cooldown: reject new `rfid_lap` if prior scored `rfid_lap` for same participant within 60s
- At most one linked `karaoke_bonus` per non-null `source_lap_id`; standalone manual karaoke rows have `source_lap_id = null`
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
tag read → RFIDTagAssociation → Bib → Participant by (event_id, bib_number)
  else treat chip UUID as legacy participant.id in this event
  if bib exists but no participant holds it → unassigned (no lap)
  if race.scheduled → test_read (no timing row of type rfid_lap)
  if race.active + finish mode + cooldown ok → rfid_lap + popup + sound
  if race.active + finish mode + cooldown fail → reject
  if race.active + checkpoint mode → checkpoint_pass / progress; maybe complete lap
  if unknown tag → unknown feedback
```

Cooldown remains keyed by `participant_id` (all tags for that racer share one cooldown).

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
