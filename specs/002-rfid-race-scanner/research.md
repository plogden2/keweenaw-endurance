# Research: RFID Race Scanner

**Feature**: `002-rfid-race-scanner` | **Date**: 2026-07-12

## R1 — Station-local persistence vs browser-only offline

**Decision**: Each reader laptop runs the standard Docker stack with **local PostgreSQL as the station authority** for race data and taps. Expand existing **IndexedDB** (`timingStorage` / `offlineQueue`) as a **short-term write-ahead / UI cache only** when the local API blips—not the system of record. Sync service replicates to **hosted PostgreSQL** when online (`HOSTED_API_URL`). **Live CSV** on disk is continuously updated for disaster recovery (see R12).

**Rationale**: Spec requires offline lap recording, multi-station sync, and CSV restore on a new laptop. The repo already uses IndexedDB for pending timing payloads and Docker Postgres for source of truth. Full offline leaderboards and multi-tag lookups need a real relational store on the station; IndexedDB alone is insufficient for authoritative multi-entity restore.

**Alternatives considered**:
- IndexedDB-only offline mirror — rejected for weak multi-station merge and CSV round-trip fidelity
- SQLite embedded in Go binary outside Docker — rejected (constitution: container-first, Postgres stack)
- Always-online Cloud SQL only — rejected (venue connectivity constraint)

## R2 — Proxmark3 continuous read architecture

**Decision**: Extend `backend/internal/rfid.Reader` with **poll/read**. Backend exposes a **WebSocket** scan stream (`GET /api/rfid/stream`) scoped to the configured station/event. Frontend `useReaderStation` composable subscribes **at app shell**. CI uses `MockReader` + `POST /api/rfid/inject`.

**Rationale**: Spec requires always-on reading across routes. WebSocket chosen over SSE for bidirectional future and simpler inject fan-out.

**Alternatives considered**:
- Browser WebUSB — rejected
- SSE — rejected in favor of WebSocket (single transport)
- Page-only polling — rejected (FR-007)

## R3 — Multi-tag model

**Decision**: Add `rfid_tag_associations` (tag_uid → participant_id). Deprecate sole reliance on `participants.rfid_tag_uid` (keep column as denormalized “primary/last written” optional for backward compat during migration). **No revoke in v1** — all rows remain active.

**Rationale**: Spec FR-006; current single column cannot express multiple tags.

**Alternatives considered**:
- JSON array on participant — rejected (indexing/lookup and uniqueness weaker)
- Revoke/deactivate flags now — rejected (clarification A)

## R4 — Event-scoped reader + finish/checkpoint modes

**Decision**: `reader_stations` (or local station config) binds to **event_id**, mode `finish` (default) or `checkpoint`, optional `checkpoint_id` / sequence order. Finish tap → scored lap (+ cooldown). Checkpoint tap → progress update; lap completes only when sequence rules satisfied. Karaoke offered only on **completed lap**.

**Rationale**: Clarifications B + C; matches existing `timing_checkpoints` entity for checkpoint mode.

**Alternatives considered**:
- Race-scoped reader only — rejected (concurrent 12h/6h)
- Checkpoint-only course model — rejected as default (finish is primary AYCEB flow)

## R5 — Cooldown & multi-station duplicate reconciliation

**Decision**: Cooldown = 60s since last **scored RFID lap** for that participant (event-wide). Enforce locally using local DB last-lap time; when online, also check hosted. On sync merge: if two scored RFID laps for same participant have timestamps within 60s, keep earliest, discard/mark duplicate the later.

**Rationale**: Spec FR-011 + multi-station edge case; karaoke does not reset cooldown.

**Alternatives considered**:
- Per-station cooldown only — rejected (double-count at two mats)
- Soft warn without discard on sync — rejected (results integrity)

## R6 — PIN access (1738)

**Decision**: Add `POST /api/auth/pin` exchanging PIN for a short-lived management JWT (reuse existing JWT middleware/roles as `admin`/`timer`). Default PIN from config `ORGANIZER_PIN=1738`. Public GET leaderboard/live/countdown unauthenticated. Mutating management + write-tag + CSV import require PIN token. Configured station scan processing and karaoke POST allowed with station token or timer role without re-prompting each tap (session after station arm).

**Rationale**: Clarification C; existing AuthService is username/password JWT — PIN is a thinner race-day path without per-user accounts.

**Alternatives considered**:
- Client-only PIN with no server check — rejected (CSV/delete must be server-enforced)
- Full RBAC user accounts for volunteers — deferred (out of clarification choice)

## R7 — Karaoke bonus representation

**Decision**: Add `record_type` on timing records: `rfid_lap` | `karaoke_bonus` (and optionally `checkpoint_pass`). Karaoke bonus references `source_lap_id` (the RFID lap just completed); unique constraint `(source_lap_id)` where type=karaoke to enforce one bonus per scan. Counts toward lap totals/placement.

**Rationale**: FR-018/019; one-shot bonus; distinct from RFID for cooldown rules.

**Alternatives considered**:
- Increment a counter without a row — rejected (audit/sync/CSV poorer)
- Fake checkpoint named Karaoke — rejected (ambiguous with real checkpoints)

## R8 — Demo seed alignment (AYCEB 2026)

**Decision**: Evolve `database/seed/generate_bluffet_seed.py` / `03-bluffet-2026.sql` to match clarified model: **3 races** (12h, 6h, 90-min kids), categories Intermediate/Advanced × Men/Women for 12h & 6h, Men/Women for kids, starts 08:00 / 08:00 / 15:00 America/Detroit, **100 participants** with sequential bibs and optional sample tag UIDs for e2e.

**Rationale**: Existing seed uses 5 races (Expert/Intermediate as races) and Youth age_group — conflicts with clarified spec.

**Alternatives considered**:
- Keep 5-race seed and map in UI — rejected (tests/spec mismatch)

## R9 — E2E strategy

**Decision**: Introduce **Playwright** under `frontend/e2e` (or repo `e2e/`) driven against `docker-compose.test.yml`. Hardware simulated via backend `MockReader` + test-only inject endpoint (or WS inject) enabled when `GO_ENV=test`. Cover all user stories before feature implementation tasks.

**Rationale**: No Playwright/Cypress in repo today; constitution + FR-020 require e2e gate; physical Proxmark3 not available in CI.

**Alternatives considered**:
- Vitest-only component tests — insufficient for multi-page continuous reader + offline
- Manual-only hardware tests — allowed as supplement, not the gate

## R10 — UI delivery process

**Decision**: HTML prototypes under `frontend/prototypes/002-rfid-race-scanner/` were produced and **user-approved 2026-07-12**. Vue implementation may proceed against those decisions (see prototype README).

**Rationale**: Constitution V satisfied.

**Alternatives considered**:
- Skip prototypes — rejected

## R11 — Pre-start countdown & test reads

**Decision**: Live event view computes countdown from each race `start_time` while `status=scheduled`. Test-read path returns participant identity without creating `rfid_lap`. **Auto-promote race to `active` at `start_time`**; PIN can also start/finish early or late.

**Rationale**: Spec FR-008a/b/c; staggered kids start.

**Alternatives considered**:
- Manual-only start — rejected
- Counting taps before start — rejected

## R12 — Live CSV snapshot

**Decision**: On each station, maintain a **live CSV file** for the configured event, rewritten/updated after every relevant mutation (laps, karaoke, racers, tags, races). Optional copy/download of that file; **no separate export job** required for recovery. Import replaces event-scoped local data on the target station.

**Rationale**: User clarification — continuity if network lost or laptop fails without remembering to export.

**Alternatives considered**:
- Manual export only — rejected
- Binary DB copy — deferred; CSV sufficient for v1

## R13 — Live view & Racers UX (approved)

**Decision**:
- Racers: debounced search; table-first; inline program; click-to-edit bib with save only when dirty
- Live: tabs 12h / 6h / 90m; overlap chart for concurrent races; default **combined overall** with category colors + legend; fullscreen rotator (flow + leaderboard side-by-side)
- Scan popup: no sound label; karaoke button becomes “recorded” after one bonus
- PIN / Station / CSV layouts per approved Option A prototypes

**Rationale**: Prototype approval session.

## R14 — Hosted sync target

**Decision**: Configure `HOSTED_API_URL` (and auth) on stations for push/pull sync to the online API/DB. When unset, station operates local-only with live CSV still updating.

**Rationale**: Clarifies FR-014 for race-day compose vs cloud.
