# Tasks: RFID Race Scanner

**Input**: Design documents from `/specs/002-rfid-race-scanner/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Included — spec FR-020 / US9 and constitution require e2e (and unit/integration) written first and failing before implementation.

**Organization**: Tasks grouped by user story for independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: User story label (US1–US9)
- Include exact file paths in descriptions

## Path Conventions

- Backend: `backend/internal/`
- Frontend: `frontend/src/`, `frontend/e2e/`, `frontend/prototypes/002-rfid-race-scanner/`
- Database: `database/migrations/`, `database/seed/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Tooling and folders for this feature on the existing Vue/Go stack

- [ ] T001 Ensure feature working dirs exist (`frontend/e2e/`, `frontend/prototypes/002-rfid-race-scanner/`, `backend/internal/services/scan/`); skip create if already present
- [ ] T002 [P] Add Playwright dev dependency and `frontend/e2e/playwright.config.ts` targeting docker-compose test URLs from `specs/002-rfid-race-scanner/quickstart.md`
- [ ] T003 [P] Add env knobs `ORGANIZER_PIN`, `RFID_INJECT`, `PROXMARK3_ENABLED`, `HOSTED_API_URL` to `backend/internal/config/config.go` and document in `docker-compose.yml` / `docker-compose.test.yml`
- [ ] T004 [P] Add npm script `test:e2e` in `frontend/package.json` for Playwright

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Schema, PIN auth, RFID poll abstraction, shared models, HTML prototypes, seed baseline — MUST complete before story implementation

**⚠️ CRITICAL**: No user story implementation until this phase is complete (failing story tests may be authored in Phase 3)

- [ ] T005 Write SQL migration `database/migrations/04-rfid-scanner.sql` for `rfid_tag_associations`, `reader_stations`, `timing_records.record_type`, `timing_records.source_lap_id`, `timing_records.station_id`, and `participants.category_id` FK per `specs/002-rfid-race-scanner/data-model.md`
- [ ] T006 [P] Extend GORM models in `backend/internal/models/models.go` for RFIDTagAssociation, ReaderStation, TimingRecord new fields, Participant.category_id; add model tests in `backend/internal/models/models_test.go`
- [ ] T007 [P] Extend `backend/internal/rfid/proxmark3.go` Reader interface with poll/read + MockReader scripted UIDs; unit tests in `backend/internal/rfid/proxmark3_test.go`
- [ ] T008 Implement PIN exchange `POST /api/auth/pin` in `backend/internal/services/auth_service.go` and `backend/internal/handlers/auth.go` (default PIN `1738`); tests in `backend/internal/services/auth_service_test.go` and `backend/internal/handlers/handlers_test.go`
- [ ] T009 Wire PIN JWT middleware on management routes in `backend/internal/handlers/handlers.go` while keeping public live GETs open per `specs/002-rfid-race-scanner/contracts/api-rfid-scanner.md`
- [ ] T010 [P] Align Bluffet seed generator `database/seed/generate_bluffet_seed.py` to 3 races, clarified categories, 100 racers with category_id + demo tag UIDs; regenerate `database/seed/03-bluffet-2026.sql`
- [x] T011 [P] HTML prototypes created and **user-approved** in `frontend/prototypes/002-rfid-race-scanner/` (racers, event-live, scan-popup, pin-unlock, station-config, csv-import-export) — Vue may proceed; race create/delete UI lives on PIN-unlocked management (see `pin-unlock.html` race-management section); no separate race-crud prototype
- [ ] T012 Add test-only inject endpoint `POST /api/rfid/inject` in `backend/internal/handlers/rfid.go` gated by `GO_ENV=test` or `RFID_INJECT=true`
- [ ] T013 Add frontend Pinia store skeleton `frontend/src/stores/pinAuth.ts` and `frontend/src/stores/station.ts` (no full UI yet)
- [ ] T013a [P] Add coverage tooling/scripts with **fail-under 100%** for new backend packages (`backend/internal/...` added by this feature) and new frontend modules under `frontend/src/` touched by this feature in `.github/workflows/ci.yml` (FR-029 / constitution VII)
- [ ] T013b Implement race auto-start at `start_time` plus PIN `POST /api/races/{id}/start|finish` in `backend/internal/services/race_service.go` and handlers; tests in `backend/internal/services/race_service_test.go`

**Checkpoint**: Foundation ready — e2e scaffolding and user stories can proceed

---

## Phase 3: User Story 9 - End-to-End Test Coverage Gate (Priority: P1)

**Goal**: Author the full failing Playwright (and supporting) e2e suite covering all acceptance scenarios so implementation is driven by red tests

**Independent Test**: `npx playwright test` runs against compose test stack with mock RFID; suite exists and fails for unimplemented behaviors

### Tests for User Story 9 ⚠️

> **NOTE: Write these tests FIRST; ensure they FAIL before story implementation phases**

- [ ] T014 [P] [US9] Add e2e helper fixtures in `frontend/e2e/fixtures/rfid.ts` (PIN login, inject tag, seed assumptions)
- [ ] T015 [P] [US9] Add failing e2e `frontend/e2e/live-lap-timing.spec.ts` for US1 (lap, cooldown, popup, sound without label, cross-route read, countdown, multi-race attribution, tabs/overlap/rotator, overall+legend)
- [ ] T016 [P] [US9] Add failing e2e `frontend/e2e/karaoke-bonus.spec.ts` for US2 acceptance
- [ ] T017 [P] [US9] Add failing e2e `frontend/e2e/racers-page.spec.ts` for US3 (debounced search, add, click-to-edit bib, inline multi-tag); assert search filter updates within 300ms after typing pause (SC-013)
- [ ] T018 [P] [US9] Add failing e2e `frontend/e2e/race-crud.spec.ts` for US4 acceptance
- [ ] T019 [P] [US9] Add failing e2e `frontend/e2e/offline-sync.spec.ts` for US5 acceptance
- [ ] T020 [P] [US9] Add failing e2e `frontend/e2e/csv-recovery.spec.ts` for US6 acceptance
- [ ] T021 [P] [US9] Add failing e2e `frontend/e2e/multi-station.spec.ts` for US7 (**3 finish stations** + checkpoint out-of-order)
- [ ] T022 [P] [US9] Add failing e2e `frontend/e2e/demo-seed.spec.ts` for US8 acceptance (event/races/categories/100 racers)
- [ ] T023 [US9] Document e2e run gate in `frontend/e2e/README.md` and assert CI hook path in `.github/workflows/ci.yml` (failing suite allowed until stories land, then required green)

**Checkpoint**: Red e2e suite exists for every user story

---

## Phase 4: User Story 1 - Live RFID Lap Timing at Reader Station (Priority: P1) 🎯 MVP

**Goal**: Event-scoped finish reader, continuous app-wide WebSocket reads, scored laps with popup/sound (no sound label), 1-minute cooldown, pre-start countdown + auto-start, overall boards with colors/legend, race tabs + overlap chart + fullscreen rotator, default live landing

**Independent Test**: Arm finish station; inject tags; verify lap/popup/sound/cooldown/countdown/test-read/overall board/tabs/rotator; navigate away and still process taps

### Tests for User Story 1 ⚠️

- [ ] T024 [P] [US1] Contract/unit tests for scan processing in `backend/internal/services/scan/scan_service_test.go` (active lap, cooldown, test_read, unknown tag)
- [ ] T025 [P] [US1] Handler tests for `POST /api/events/{eventId}/scans` and live GET (overall + legend) in `backend/internal/handlers/handlers_test.go`
- [ ] T026 [P] [US1] Vitest for `frontend/src/composables/useReaderStation.spec.ts` and scan popup / live board components under `frontend/src/components/`

### Implementation for User Story 1

- [ ] T027 [US1] Implement scan service (event resolve, finish-mode +1 lap, cooldown, test_read) in `backend/internal/services/scan/scan_service.go`
- [ ] T028 [US1] Implement **WebSocket** scan stream `GET /api/rfid/stream` in `backend/internal/handlers/rfid.go` feeding from Proxmark3 poll loop
- [ ] T029 [US1] Implement `PUT/GET /api/stations/current` in `backend/internal/handlers/station.go` and service in `backend/internal/services/station_service.go` (event bind, mode default `finish`)
- [ ] T030 [US1] Implement `GET /api/events/{eventId}/live` with countdown, overall leaderboard, category legend/colors, flow series in `backend/internal/handlers/events.go` / results service
- [ ] T031 [US1] Build Vue `frontend/src/views/EventLive.vue` per approved prototype (tabs 12h/6h/90m, overlap chart, overall+legend, fullscreen rotator) and route in `frontend/src/router/index.ts`
- [ ] T032 [US1] Build app-shell `frontend/src/composables/useReaderStation.ts` (WebSocket) subscribed in `frontend/src/App.vue`
- [ ] T033 [US1] Build `frontend/src/components/ScanPopup.vue` (overall place, no sound label) + audio `frontend/src/assets/audio/new-lap.mp3` (or approved equivalent)
- [ ] T034 [US1] Build `frontend/src/views/StationConfig.vue` + `frontend/src/views/PinUnlock.vue`; default navigate to EventLive when any race active
- [ ] T035 [US1] Wire frontend API clients in `frontend/src/services/api.ts` for station, scans, live, stream
- [ ] T036 [US1] Make `frontend/e2e/live-lap-timing.spec.ts` pass

**Checkpoint**: US1 MVP timing works end-to-end with mock RFID

---

## Phase 5: User Story 2 - Karaoke Bonus Lap After Scan (Priority: P1)

**Goal**: One-click karaoke bonus on scored-lap popup; one bonus per source lap; counts in totals/placement

**Independent Test**: Score a lap, click karaoke once, verify +1 bonus and leaderboard; second click does not duplicate

### Tests for User Story 2 ⚠️

- [ ] T037 [P] [US2] Service tests for karaoke bonus uniqueness in `backend/internal/services/scan/karaoke_service_test.go`
- [ ] T038 [P] [US2] Handler tests for `POST /api/timing-records/{id}/karaoke-bonus` in `backend/internal/handlers/handlers_test.go`

### Implementation for User Story 2

- [ ] T039 [US2] Implement karaoke bonus creation (`record_type=karaoke_bonus`, `source_lap_id`) in `backend/internal/services/scan/karaoke_service.go`
- [ ] T040 [US2] Expose karaoke endpoint in `backend/internal/handlers/timing.go` (no re-PIN on armed station)
- [ ] T041 [US2] Add karaoke control to `frontend/src/components/ScanPopup.vue` — button becomes “Karaoke bonus lap recorded” after one use (no sound label)
- [ ] T042 [US2] Include karaoke laps in overall placement/lap_count in `backend/internal/services/results_service.go`
- [ ] T043 [US2] Make `frontend/e2e/karaoke-bonus.spec.ts` pass

**Checkpoint**: US2 karaoke bonus works on finish-completed laps

---

## Phase 6: User Story 3 - Racers Page (Priority: P1)

**Goal**: Debounced searchable racers list, add racer with category, sequential/click-to-edit bibs, inline Proxmark3 multi-tag program (no revoke)

**Independent Test**: PIN → Racers; type to filter; add/edit bib (save icon when dirty); program two tags inline; both resolve on scan

### Tests for User Story 3 ⚠️

- [ ] T044 [P] [US3] Tests for multi-tag association CRUD in `backend/internal/services/rfid_service_test.go`
- [ ] T045 [P] [US3] Vitest for Racers view (debounce search, bib edit) in `frontend/src/views/Racers.test.ts`

### Implementation for User Story 3

- [ ] T046 [US3] Implement tag association APIs in `backend/internal/handlers/rfid.go` and participant search `?q=` + category_id in `backend/internal/handlers/participants.go`
- [ ] T047 [US3] Update `RFIDService.WriteTag` / lookup in `backend/internal/services/rfid_service.go` to use `rfid_tag_associations`
- [ ] T048 [US3] Sequential bib default + unique bib validation in `backend/internal/services/participant_service.go`
- [ ] T049 [US3] Build `frontend/src/views/Racers.vue` per approved prototype (debounced search, inline program, click-to-edit bib) + route
- [ ] T050 [US3] Make `frontend/e2e/racers-page.spec.ts` pass

**Checkpoint**: US3 enrollment and multi-tag programming complete

---

## Phase 7: User Story 5 - Local Persistence, Online Sync, Offline Continuity (Priority: P1)

**Goal**: Local Postgres authority + IndexedDB WAQ; offline scored taps; sync pending to hosted when online; status indicator

**Independent Test**: With local API up, go offline from hosted; taps still persist in local Postgres; IndexedDB only queues UI→API payloads if local API blips; on reconnect pending clears and hosted matches

### Tests for User Story 5 ⚠️

- [ ] T051 [P] [US5] Unit tests for offline queue/mirror in `frontend/src/services/timingStorage.test.ts` and `frontend/src/services/offlineQueue.test.ts`
- [ ] T052 [P] [US5] Backend sync merge/cooldown-dedupe tests in `backend/internal/services/sync_service_test.go`

### Implementation for User Story 5

- [ ] T053 [US5] Expand IndexedDB in `frontend/src/services/timingStorage.ts` as **WAQ/UI cache only** (pending scans/bonuses + minimal display cache of last-known event/race labels). MUST NOT be treated as offline system of record; local Postgres remains authority per plan R1
- [ ] T054 [US5] Expand `frontend/src/services/offlineQueue.ts` to enqueue scans/karaoke when API unreachable and replay on `online`
- [ ] T055 [US5] Implement `POST /api/sync/push` and `POST /api/sync/pull` using `HOSTED_API_URL` in `backend/internal/services/sync_service.go` + `backend/internal/handlers/sync.go`
- [ ] T056 [US5] Ensure scan path writes local DB first with `sync_status=pending_sync` when remote unreachable (station backend)
- [ ] T057 [US5] Add station online/offline/pending UI indicator on `frontend/src/views/EventLive.vue` / App header
- [ ] T058 [US5] Make `frontend/e2e/offline-sync.spec.ts` pass

**Checkpoint**: US5 offline continuity verified

---

## Phase 8: User Story 4 - Easy Race Entry and Deletion (Priority: P2)

**Goal**: Simple PIN-gated create/delete race under an event with confirmation

**Independent Test**: Create lap race; appears for racers/reader; delete with confirm removes it; cancel leaves it

### Tests for User Story 4 ⚠️

- [ ] T059 [P] [US4] Handler tests for race create/delete authz (PIN required) in `backend/internal/handlers/handlers_test.go`

### Implementation for User Story 4

- [ ] T060 [US4] Ensure race create/delete endpoints enforce PIN JWT in `backend/internal/handlers/races.go` (or events handlers)
- [ ] T061 [US4] Add PIN-gated race create/delete UI on event management (follow `pin-unlock.html` race-management section): create form + delete with confirm; no separate race-crud prototype required
- [ ] T062 [US4] On race delete, stop accepting taps for that race’s participants while event reader continues for others
- [ ] T063 [US4] Make `frontend/e2e/race-crud.spec.ts` pass

**Checkpoint**: US4 race CRUD complete

---

## Phase 9: User Story 8 - Demo Seed AYCEB 2026 (Priority: P2)

**Goal**: Seed matches clarified event shape and powers demos/e2e

**Independent Test**: Load seed; verify name/date, 3 races/start times, category matrix, 100 racers

### Tests for User Story 8 ⚠️

- [ ] T064 [P] [US8] Seed validation test (SQL or Go) asserting counts/categories in `backend/internal/database/bluffet_seed_test.go`

### Implementation for User Story 8

- [ ] T065 [US8] Finalize `database/seed/generate_bluffet_seed.py` + `database/seed/03-bluffet-2026.sql` (timezone America/Detroit starts, demo tags)
- [ ] T066 [US8] Document seed load steps in `specs/002-rfid-race-scanner/quickstart.md` (already drafted — verify commands)
- [ ] T067 [US8] Make `frontend/e2e/demo-seed.spec.ts` pass

**Checkpoint**: US8 demo seed accepted

---

## Phase 10: User Story 6 - CSV Disaster Recovery (Priority: P2)

**Goal**: Continuously maintained live CSV on each station + PIN-gated import restores racers, tags, laps on a replacement laptop without requiring a manual export beforehand

**Independent Test**: Change data offline; confirm live CSV updates; import on fresh station; continue scanning with prior counts

### Tests for User Story 6 ⚠️

- [ ] T068 [P] [US6] Round-trip + live-snapshot update tests in `backend/internal/services/csv_export_test.go` per `specs/002-rfid-race-scanner/contracts/csv-race-export.md`

### Implementation for User Story 6

- [ ] T069 [US6] Implement continuous live CSV writer (update on lap/racer/tag/race mutations) + `GET /api/events/{eventId}/live-csv` in `backend/internal/services/csv_export.go`
- [ ] T070 [US6] Implement import `POST /api/events/{eventId}/import.csv` with replace-semantics for event scope
- [ ] T071 [US6] Build CSV UI in `frontend/src/views/CsvRecovery.vue` per approved prototype (live status + import; PIN-gated)
- [ ] T072 [US6] Make `frontend/e2e/csv-recovery.spec.ts` pass (assert live file updates without export click)

**Checkpoint**: US6 CSV recovery works

---

## Phase 11: User Story 7 - Multiple Concurrent Reader Stations (Priority: P2)

**Goal**: Arbitrary finish readers (default) + optional checkpoint mode; race-wide cooldown after sync; out-of-order checkpoint does not complete lap

**Independent Test**: Three finish stations record distinct racers; cooldown shared after sync; checkpoint station rejects out-of-order completion

### Tests for User Story 7 ⚠️

- [ ] T073 [P] [US7] Multi-station cooldown merge tests in `backend/internal/services/sync_service_test.go`
- [ ] T074 [P] [US7] Checkpoint sequence tests in `backend/internal/services/scan/checkpoint_service_test.go`

### Implementation for User Story 7

- [ ] T075 [US7] Implement checkpoint-mode progress + lap completion rules in `backend/internal/services/scan/checkpoint_service.go`
- [ ] T076 [US7] Station config UI supports mode `finish`|`checkpoint` + checkpoint picker in `frontend/src/views/StationConfig.vue`
- [ ] T077 [US7] Karaoke only on completed laps (not intermediate checkpoint_pass) enforced in karaoke service + ScanPopup
- [ ] T078 [US7] Sync dedupe within 60s window for same participant RFID laps across stations
- [ ] T079 [US7] Make `frontend/e2e/multi-station.spec.ts` pass

**Checkpoint**: US7 multi-station + checkpoint mode complete

---

## Phase 12: Polish & Cross-Cutting Concerns

**Purpose**: Hardening, docs, CI green, quickstart validation

- [ ] T080 [P] Add Mario Kart–rights note or approved equivalent asset README in `frontend/src/assets/audio/README.md`
- [ ] T081 [P] Update root `README.md` with RFID scanner quickstart link to `specs/002-rfid-race-scanner/quickstart.md`
- [ ] T082 Run full suite: backend `go test ./...` with **100%** coverage fail-under for new packages, frontend Vitest, Playwright e2e — all US1–US9 green
- [ ] T083 Validate manual steps in `specs/002-rfid-race-scanner/quickstart.md` against docker compose
- [ ] T084 [P] Accessibility pass on scan popup, countdown, Racers search, and live legend (labels/focus) in affected Vue views
- [ ] T085 Ensure CI in `.github/workflows/ci.yml` runs e2e with mock RFID, seeded DB, and **100%** coverage fail-under for new packages (same scope as T013a)
- [ ] T086 [P] Document Proxmark3 Docker USB device passthrough notes in `specs/002-rfid-race-scanner/quickstart.md` (optional hardware path)
- [ ] T087 Manual acceptance: SC-001 (≤2s popup+sound), SC-003 (≤3s karaoke), SC-012 (1s countdown), SC-013 (~300ms debounce) — record pass/fail in PR notes
- [ ] T088 Manual field checklist: SC-005 4-hour offline soak (or shortened soak with documented waiver)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 Setup**: No dependencies
- **Phase 2 Foundational**: Depends on Setup — **BLOCKS** all story implementation
- **Phase 3 US9 (e2e red suite)**: Depends on Foundational (Playwright + inject + seed baseline)
- **Phases 4–7 (P1 stories US1, US2, US3, US5)**: Depend on Foundational + US9 red tests; prefer US1 → US2 → US3 → US5
- **Phases 8–11 (P2 stories US4, US8, US6, US7)**: Depend on Foundational; US8 seed finalize can parallelize after T010; US6/US7 after US1+US5 strongly recommended
- **Phase 12 Polish**: Depends on desired stories complete; requires all e2e green for feature done

### User Story Dependencies

- **US9**: After Foundational — produces failing tests that drive implementation
- **US1**: After Foundational + US9 live-lap e2e — **MVP**
- **US2**: After US1 scan popup / scored lap id available
- **US3**: After Foundational (multi-tag model); integrates with US1 lookup
- **US5**: After US1 scan write path exists
- **US4**: After PIN foundation; independent of karaoke
- **US8**: After T010 baseline; finalize when category/racer counts stable
- **US6**: After US5 sync model + US3 tags + US1 laps
- **US7**: After US1 station + US5 sync

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Models/services before handlers before Vue
- HTML prototype approval before Vue for new surfaces (T011)
- Story e2e green before claiming story done

### Parallel Opportunities

- T002–T004 setup in parallel
- T006–T007, T010–T011 foundational in parallel after T005
- T014–T022 US9 e2e specs in parallel
- After US1: US3 and US4 can proceed in parallel with different owners
- US8 seed tests parallel with US4 UI

---

## Parallel Example: User Story 9

```bash
# Launch failing e2e specs together:
Task: "frontend/e2e/live-lap-timing.spec.ts"
Task: "frontend/e2e/karaoke-bonus.spec.ts"
Task: "frontend/e2e/racers-page.spec.ts"
Task: "frontend/e2e/race-crud.spec.ts"
Task: "frontend/e2e/offline-sync.spec.ts"
Task: "frontend/e2e/csv-recovery.spec.ts"
Task: "frontend/e2e/multi-station.spec.ts"
Task: "frontend/e2e/demo-seed.spec.ts"
```

## Parallel Example: User Story 1

```bash
# Tests in parallel:
Task: "backend/internal/services/scan/scan_service_test.go"
Task: "handler scan/live tests in handlers_test.go"
Task: "frontend useReaderStation.spec.ts"

# Then implement services → stream → Vue
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 Setup
2. Complete Phase 2 Foundational
3. Complete Phase 3 US9 red e2e suite
4. Complete Phase 4 US1
5. **STOP and VALIDATE**: `live-lap-timing.spec.ts` green — demo MVP reader

### Incremental Delivery

1. Setup + Foundational + US9 red suite
2. US1 MVP → demo
3. US2 karaoke → demo
4. US3 racers/tags → demo
5. US5 offline → field readiness
6. US4 + US8 polish event ops/seed
7. US6 CSV + US7 multi-station → full race-day resilience
8. Polish / CI green

### Parallel Team Strategy

1. Team completes Setup + Foundational + US9 together
2. Then:
   - Dev A: US1 → US2
   - Dev B: US3 → US4
   - Dev C: US5 → US6 → US7
   - Shared: US8 seed + CI

---

## Notes

- [P] = different files, no incomplete-task dependencies
- [Story] labels map to spec user stories US1–US9
- HTML prototypes **approved** (T011 done) — Vue tasks may proceed
- Default station mode is **finish**; checkpoint is US7
- Organizer PIN default **1738**; leaderboard/countdown public
- Live CSV always maintained; no manual export required
- WebSocket for scan stream (not SSE)
- Local Postgres is station authority; IndexedDB is WAQ only
- No tag revoke in v1
- Commit after each task or logical group
