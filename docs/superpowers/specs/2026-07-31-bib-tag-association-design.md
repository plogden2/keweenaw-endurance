# Bib-Associated RFID Tags — Design

**Date**: 2026-07-31  
**Status**: Approved (design)  
**Related**: RFID scanner (`002-rfid-race-scanner`), Proxmark3 UUID write, multi-tag-type classic

## Problem

Today RFID tags associate directly to a **Participant** and chips are programmed with the **racer UUID**. That forces programming after (or while) each person is registered, which blocks the race-morning workflow:

1. Night before: attach/program tags on physical bibs 1–N  
2. Race morning: hand a pre-tagged bib to a racer and enter that bib number on the Racers page  

Staff need tags bound to **bib numbers** (event-scoped inventory) so chips can be written before anyone is assigned.

## Goals

1. First-class event-scoped **Bib** entity; chips store **Bib UUID**.
2. Tag associations point at `bib_id` (multiple active tags per bib; no revoke in v1).
3. Bib numbers unique **per event** (no overlap across races in the same event).
4. **Event → Bibs** inventory: bulk-create range + program tags (night-before path).
5. **Racers**: assign/edit bib number; warn if bib has zero tags; ad-hoc write against current bib.
6. Scan resolves `tag → Bib → Participant`; distinct **unassigned** state when bib has tags but no racer.
7. Dual-resolve legacy chips that still hold a **participant UUID**.
8. Migration backfill + live CSV / multi-station sync include Bibs and tag→bib links.

## Non-goals (v1)

- Tag revoke / deactivate / kill-list
- Moving tag associations with the person on bib edit (tags always stay on the Bib)
- Writing `bib_number` or participant UUID as the sole *new* chip payload
- Per-race bib uniqueness / overlapping bibs across races in one event
- Auto-enforced disjoint ranges (kids vs adults) — docs/seed guidance only
- Cross-event reuse of the same chip without reprogram
- Hardware asset tracking beyond bib ↔ `tag_uid`
- Per-tag or per-bib cooldown
- Field rewrite of old chips (dual-resolve covers until reprogrammed)

## Decisions

| Topic | Decision |
|-------|----------|
| Approach | First-class event-scoped Bib entity |
| Chip payload (new writes) | Bib UUID only (UI shows bib number) |
| Association target | `RFIDTagAssociation.bib_id` (was `participant_id`) |
| Multi-tag | Multiple active tags per bib; no revoke |
| Bib uniqueness | Unique `(event_id, bib_number)` |
| Missing bib / no tags on assign | Allow save; **warn** if zero tags; program later from Racers |
| Night-before UX | Dedicated **Event → Bibs** inventory (bulk create + program) |
| Morning UX | Racers page: enter bib given to racer |
| Bib edit after tags/laps | Allow; tags stay with Bib numbers; **confirm** when source/target has tags or scored laps |
| Legacy chips | Dual-resolve: association→Bib first, else chip UUID as `participant.id` |
| Tag rebind A→B | Last write wins; warn; `tag_uid` remains DB-unique |
| Cooldown | Keyed by `participant_id` (all tags for that racer share one cooldown) |
| Timing history | Laps stay on `participant_id`; no rewrite on bib change |

## Data model

### Bib (new)

| Field | Rules |
|-------|--------|
| id | UUID PK — written to RFID chips |
| event_id | FK, required |
| bib_number | Unique within event |
| created_at | |

### RFIDTagAssociation (changed)

| Field | Rules |
|-------|--------|
| id | UUID PK |
| bib_id | FK → Bib (replaces `participant_id`) |
| tag_uid | Unique globally; required |
| created_at | |
| active | Always true in v1 |

### Participant (unchanged columns)

Keeps `bib_number`. Assignment ensures a Bib exists for `(event, bib_number)`. At most one Participant in the event may hold a given bib number (validated across all races in the event). Optional: expose `tag_uids` via the racer’s current Bib.

### Logical relationships

```text
Event 1──* Bib 1──* RFIDTagAssociation
Event 1──* Race 1──* Participant
Participant.bib_number + Event ──> Bib   (at most one holder per bib in the event)
```

## Staff UX

### Event → Bibs (PIN)

- Inventory table: bib number, tag count, assigned racer or “unassigned”
- Bulk-create a numeric range (e.g. 1–100)
- Select bib → program one or more tags via Proxmark3 (writes Bib UUID)
- Primary night-before programming surface

### Racers (per race, PIN)

- Search / add / edit as today
- Setting `bib_number` ensures event Bib exists; enforces **event-wide** uniqueness (clash with another race shown on save)
- Warn if that bib has zero tags
- Ad-hoc “Write tag” programs against the racer’s current Bib
- Changing bib allowed; confirm when source/target bib has tags or scored laps

### Writer entry points

- Primary: Bibs page  
- Secondary: Racers page (same write-to-bib API)

## Scan path

**Resolve order** (reader is event-bound):

1. `tag_uid` → `RFIDTagAssociation` → `Bib` → Participant by `(event_id, bib_number)`
2. Else treat chip payload UUID as legacy `participant.id` if that participant is in this event
3. Else **unknown**

Prefer the association/bib path when both could apply. One tap never double-counts.

**Feedback states** (mutually distinct; success sound only for scored lap):

| State | Meaning |
|-------|---------|
| unknown | No association and no legacy participant |
| unassigned | Bib (+tags) exists; no Participant holds that bib |
| test_read | Participant found; race not yet active (identity only; no lap) |
| cooldown | Scored path blocked by 60s rule |
| success | Lap recorded + popup + sound |
| rejected / non-scoring | Finished/cancelled or other non-score |

**After bib edit:** old bib’s tags resolve to unassigned (or a new holder); new bib’s tags resolve to this racer. Historical laps unchanged.

## Migration, CSV, sync

1. Add `bibs` table; change associations to `bib_id`
2. Backfill: for each Participant with `bib_number`, ensure Bib `(event, bib_number)`; repoint associations; keep participant IDs for legacy dual-resolve
3. New Proxmark3 writes use Bib UUID only
4. Live CSV includes Bib rows (`bib_id`, `event_id`, `bib_number`) and tag rows as `tag_uid` → `bib_id`; participants still export with `bib_number` + race
5. Import restores Bibs → associations → participants/laps
6. Multi-station: Bib create/program and Racers assign sync like other mutations; morning assign must see overnight Bibs
7. Offline tag→racer caches invalidate/rebuild when Bib/tag/assignment changes

## Spec drift to update (when implementing)

- FR-005 / Racers stories: program **bib UUID**, not racer UUID
- FR-024 / bib uniqueness: **event-unique**, not only per-race
- Data model + API contracts for associations and write endpoints

## Testing

- Pre-program bibs → morning assign → scored lap
- Unassigned bib tap (no lap; clear feedback)
- Bib reassignment with tags + confirm UX
- Tag rebind bib A → bib B (last write wins)
- Event-wide bib clash across races
- Multi-tag shared participant cooldown
- Legacy participant-UUID dual-resolve
- CSV / sync round-trip with Bibs
- Multi-station: overnight program on A, morning assign on B

## Implementation note

After the implementation plan is written and approved, execute via **subagent-driven development**.
