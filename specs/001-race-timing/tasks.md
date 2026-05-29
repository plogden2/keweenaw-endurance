# Tasks: Keweenaw Endurance Race Timing System

**Input**: Design documents from `/specs/001-race-timing/`

**Prerequisites**: plan.md, spec.md

**Tests**: Included — spec and constitution mandate strict TDD for all new code.

**Organization**: Tasks grouped by user story for independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: User story label (US1–US13)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure) ✅ COMPLETE

**Purpose**: Project initialization and basic structure

- [X] T001 Create Docker Compose development environment in `docker-compose.yml`
- [X] T002 Initialize Go backend project in `backend/` with Gin framework
- [X] T003 [P] Initialize Vue.js + TypeScript frontend project in `frontend/` with Vite
- [X] T004 [P] Configure PostgreSQL schema in `database/init/01-init.sql`
- [X] T005 [P] Configure test infrastructure in `docker-compose.test.yml`

---

## Phase 2: Foundational (Blocking Prerequisites) ✅ COMPLETE

**Purpose**: Core infrastructure that MUST be complete before user story work

- [X] T006 Define GORM models in `backend/internal/models/models.go`
- [X] T007 Implement database migration in `backend/internal/database/database.go`
- [X] T008 [P] Implement config loading in `backend/internal/config/config.go`
- [X] T009 [P] Implement CORS and security middleware in `backend/internal/middleware/middleware.go`
- [X] T010 Wire API routes in `backend/cmd/server/main.go`
- [X] T011 Fix Go module version compatibility in `backend/go.mod` and Dockerfiles

**Checkpoint**: Foundation ready — user story implementation can begin

---

## Phase 3: User Story 1 - Event Management (Priority: P1) ✅ COMPLETE

**Goal**: Organizers can create, view, update, and delete race events

**Independent Test**: CRUD operations on `/api/events` return correct JSON with validation errors for bad input

### Tests for User Story 1

- [X] T012 [P] [US1] Event service tests in `backend/internal/services/event_service_test.go`
- [X] T013 [P] [US1] Event handler integration tests in `backend/internal/handlers/handlers_test.go`

### Implementation for User Story 1

- [X] T014 [US1] Implement EventService in `backend/internal/services/event_service.go`
- [X] T015 [US1] Implement event handlers in `backend/internal/handlers/events.go`
- [X] T016 [US1] Wire event service into `backend/internal/services/services.go`

**Checkpoint**: Event CRUD fully functional via API

---

## Phase 4: User Story 2 - Race Management (Priority: P1) ✅ COMPLETE

**Goal**: Organizers can manage races within events

**Independent Test**: CRUD operations on `/api/races` with event relationship validation

### Tests for User Story 2

- [X] T017 [P] [US2] Race service tests in `backend/internal/services/race_service_test.go`
- [X] T018 [P] [US2] Race handler tests in `backend/internal/handlers/handlers_test.go`

### Implementation for User Story 2

- [X] T019 [US2] Implement RaceService in `backend/internal/services/race_service.go`
- [X] T020 [US2] Implement race handlers in `backend/internal/handlers/races.go`

**Checkpoint**: Race CRUD fully functional via API

---

## Phase 5: User Story 3 - Participant Registration (Priority: P1) ✅ COMPLETE

**Goal**: Register participants for races with bib numbers and RFID tags

**Independent Test**: CRUD on `/api/participants` with uniqueness constraints enforced

### Tests for User Story 3

- [X] T021 [P] [US3] Participant service tests in `backend/internal/services/participant_service_test.go`
- [X] T022 [P] [US3] Participant handler tests in `backend/internal/handlers/handlers_test.go`

### Implementation for User Story 3

- [X] T023 [US3] Implement ParticipantService in `backend/internal/services/participant_service.go`
- [X] T024 [US3] Implement participant handlers in `backend/internal/handlers/participants.go`

**Checkpoint**: Participant registration fully functional via API

---

## Phase 6: Polish (MVP Backend) ✅ COMPLETE

- [X] T025 [P] Update `.gitignore` with essential patterns for Go, Node.js, Docker
- [X] T026 Run full backend test suite via `docker-compose -f docker-compose.test.yml run backend-test`
- [X] T027 Remove stub handler implementations from `backend/internal/handlers/handlers.go`

---

## Phase 7: User Story 4 - Checkpoint & Category Management (Priority: P1)

**Goal**: Organizers configure timing checkpoints and result categories for each race

**Independent Test**: CRUD on `/api/races/:raceId/checkpoints` and `/api/races/:raceId/categories` with type validation and race scoping

### Tests for User Story 4

- [X] T028 [P] [US4] Checkpoint service tests in `backend/internal/services/checkpoint_service_test.go`
- [X] T029 [P] [US4] Category service tests in `backend/internal/services/category_service_test.go`
- [X] T030 [P] [US4] Checkpoint and category handler tests in `backend/internal/handlers/handlers_test.go`

### Implementation for User Story 4

- [X] T031 [US4] Implement CheckpointService in `backend/internal/services/checkpoint_service.go`
- [X] T032 [US4] Implement CategoryService in `backend/internal/services/category_service.go`
- [X] T033 [US4] Implement checkpoint handlers in `backend/internal/handlers/checkpoints.go`
- [X] T034 [US4] Implement category handlers in `backend/internal/handlers/categories.go`
- [X] T035 [US4] Wire checkpoint and category services and routes in `backend/internal/services/services.go` and `backend/cmd/server/main.go`

**Checkpoint**: Checkpoints and categories manageable via API for any race

---

## Phase 8: User Story 5 - Timing Records & Results (Priority: P1) 🎯 NEXT MVP

**Goal**: Record timing events at checkpoints and calculate race results and leaderboards

**Independent Test**: `POST /api/timing/record` creates validated records; `GET /api/timing/results/:raceId` and `GET /api/timing/leaderboard/:raceId` return correctly ranked results for time-based and lap-based races

### Tests for User Story 5

- [X] T036 [P] [US5] Timing service tests in `backend/internal/services/timing_service_test.go`
- [X] T037 [P] [US5] Results calculation tests in `backend/internal/services/results_service_test.go`
- [X] T038 [P] [US5] Timing handler tests in `backend/internal/handlers/handlers_test.go`

### Implementation for User Story 5

- [X] T039 [US5] Implement TimingService in `backend/internal/services/timing_service.go`
- [X] T040 [US5] Implement ResultsService with time-based and lap-based logic in `backend/internal/services/results_service.go`
- [X] T041 [US5] Replace timing handler stubs in `backend/internal/handlers/timing.go`
- [X] T042 [US5] Add `PUT /api/timing/records/:id` route and handler in `backend/internal/handlers/timing.go` and `backend/cmd/server/main.go`

**Checkpoint**: Core timing loop works — record times, view results and leaderboards

---

## Phase 9: User Story 6 - RFID Integration API (Priority: P1) ✅ COMPLETE

**Goal**: RFID tag lookup, manual timing entry, and multi-station sync status

**Independent Test**: `GET /api/rfid/scan/:uid` returns participant; `POST /api/rfid/manual-entry` creates a timing record; `GET /api/rfid/sync-status` reports pending records

### Tests for User Story 6

- [X] T043 [P] [US6] RFID service tests in `backend/internal/services/rfid_service_test.go`
- [X] T044 [P] [US6] RFID handler tests in `backend/internal/handlers/handlers_test.go`

### Implementation for User Story 6

- [X] T045 [US6] Create proxmark3 hardware interface in `backend/internal/rfid/proxmark3.go`
- [X] T046 [US6] Implement RFIDService in `backend/internal/services/rfid_service.go`
- [X] T047 [US6] Replace RFID handler stubs in `backend/internal/handlers/rfid.go`
- [X] T048 [US6] Add `POST /api/rfid/sync-pending` route in `backend/cmd/server/main.go` and `backend/internal/handlers/rfid.go`

**Checkpoint**: RFID endpoints functional with hardware abstraction for field deployment

---

## Phase 10: User Story 7 - Frontend API Client & State (Priority: P2)

**Goal**: Shared API client and Pinia stores for backend data access

**Independent Test**: Unit tests confirm stores fetch, cache, and expose events, races, and participants from the API

### Tests for User Story 7

- [X] T049 [P] [US7] API client unit tests in `frontend/src/services/api.test.ts`
- [X] T050 [P] [US7] Pinia store tests in `frontend/src/stores/events.test.ts`

### Implementation for User Story 7

- [X] T051 [P] [US7] Create Axios API client in `frontend/src/services/api.ts`
- [X] T052 [P] [US7] Create Pinia stores for events, races, and participants in `frontend/src/stores/events.ts`, `frontend/src/stores/races.ts`, and `frontend/src/stores/participants.ts`
- [X] T053 [US7] Register Pinia in `frontend/src/main.ts`
- [X] T054 [P] [US7] Configure Vitest in `frontend/vitest.config.ts`

**Checkpoint**: Frontend can fetch and hold race data from the backend API

---

## Phase 11: User Story 8 - Timing Section UI (Priority: P2)

**Goal**: Active and past event tables wired to the API with event and race detail navigation

**Independent Test**: `/timing` displays live event data; clicking an event navigates to `/timing/:eventId`; race details show leaderboard tab with API data

### Tests for User Story 8

- [X] T055 [P] [US8] Timing page component tests in `frontend/src/views/Timing.test.ts`
- [X] T056 [P] [US8] EventDetails component tests in `frontend/src/views/EventDetails.test.ts`
- [X] T057 [P] [US8] RaceDetails component tests in `frontend/src/views/RaceDetails.test.ts`

### Implementation for User Story 8

- [X] T058 [US8] Wire `Timing.vue` to events API in `frontend/src/views/Timing.vue`
- [X] T059 [P] [US8] Create `EventDetails.vue` in `frontend/src/views/EventDetails.vue`
- [X] T060 [P] [US8] Create `RaceDetails.vue` with leaderboard tab in `frontend/src/views/RaceDetails.vue`
- [X] T061 [P] [US8] Create shared `AppHeader.vue` with Inferior Timing logo in `frontend/src/components/AppHeader.vue`

**Checkpoint**: Timing index and detail pages functional end-to-end with backend

---

## Phase 11b: Frontend TypeScript Migration (Priority: P2)

**Goal**: Align the frontend codebase and toolchain with the TypeScript specification

**Independent Test**: `npm run typecheck` and `npm run test:ci` pass with all modules using `.ts` extensions and Vue SFCs using `<script setup lang="ts">`

- [X] T091 [P] Add TypeScript toolchain in `frontend/tsconfig.json`, `frontend/tsconfig.node.json`, and `frontend/tsconfig.vitest.json`
- [X] T092 [P] Add shared API/domain types in `frontend/src/types/models.ts` aligned with backend JSON shapes
- [X] T093 [P] Convert build and test configs plus entrypoint: `frontend/vite.config.ts`, `frontend/vitest.config.ts`, `frontend/src/main.ts`, `frontend/src/router/index.ts`
- [X] T094 [P] Migrate services, stores, tests, and helpers from `.js` to `.ts` under `frontend/src/services/`, `frontend/src/stores/`, `frontend/src/test/`, and `frontend/src/views/*.test.ts`
- [X] T095 Add `vue-tsc --noEmit` typecheck script to `frontend/package.json` and wire into `docker-compose.test.yml` frontend-test command

**Checkpoint**: No remaining frontend `.js` source files except generated assets; typecheck enforced in CI

---

## Phase 12: User Story 9 - Live Timing Station UI (Priority: P2)

**Goal**: Real-time timing station interface with bib/RFID lookup and manual entry

**Independent Test**: `/timing/live/:raceId` shows participant lookup, manual timing form submits records, and sync status is visible

### Tests for User Story 9

- [X] T062 [P] [US9] LiveTiming component tests in `frontend/src/views/LiveTiming.test.ts`
- [X] T063 [P] [US9] ManualTimingForm component tests in `frontend/src/components/ManualTimingForm.test.ts`

### Implementation for User Story 9

- [X] T064 [US9] Create `LiveTiming.vue` with bib and RFID lookup in `frontend/src/views/LiveTiming.vue`
- [X] T065 [US9] Create `ManualTimingForm.vue` in `frontend/src/components/ManualTimingForm.vue`
- [X] T066 [US9] Create sync status indicator component in `frontend/src/components/SyncStatus.vue`

**Checkpoint**: Timing operators can record times from the browser UI

---

## Phase 13: User Story 10 - Landing Page (Priority: P2)

**Goal**: Minimal teaser landing page with race cards and featured Bluffet link per constitutional UI protocol

**Independent Test**: `/` shows hero, featured Bluffet link, and teaser race cards with external links only; HTML prototype approved before Vue work

### Tests for User Story 10

- [X] T067 [P] [US10] Home page component tests in `frontend/src/views/Home.test.ts`

### Implementation for User Story 10

- [X] T068 [US10] Create HTML landing page prototype in `prototypes/landing.html`
- [X] T069 [US10] Refine `Home.vue` to match approved prototype in `frontend/src/views/Home.vue`
- [X] T070 [P] [US10] Create reusable `RaceCard.vue` teaser component in `frontend/src/components/RaceCard.vue`

**Checkpoint**: Landing page meets constitutional HTML-first workflow and spec design

---

## Phase 14: User Story 11 - PWA & Offline Sync (Priority: P3)

**Goal**: Progressive Web App with offline timing queue and background sync

**Independent Test**: App loads offline after first visit; pending timing records queue locally and sync when connection restores

### Tests for User Story 11

- [X] T071 [P] [US11] Offline queue unit tests in `frontend/src/services/offlineQueue.test.ts`
- [X] T072 [P] [US11] IndexedDB storage tests in `frontend/src/services/timingStorage.test.ts`

### Implementation for User Story 11

- [X] T073 [US11] Configure Vite PWA plugin and manifest in `frontend/vite.config.ts` and `frontend/public/manifest.json`
- [X] T074 [US11] Implement offline timing queue in `frontend/src/services/offlineQueue.ts`
- [X] T075 [US11] Implement IndexedDB timing storage in `frontend/src/services/timingStorage.ts`
- [X] T076 [US11] Register service worker in `frontend/src/main.ts`

**Checkpoint**: Timing station works without network connectivity

---

## Phase 15: User Story 12 - Race Flow Visualization (Priority: P3)

**Goal**: Interactive race flow charts and statistics on race detail pages

**Independent Test**: Race Flow tab renders position-over-time chart; Statistics tab shows participant distribution and split analysis

### Tests for User Story 12

- [X] T077 [P] [US12] RaceFlowChart component tests in `frontend/src/components/RaceFlowChart.test.ts`

### Implementation for User Story 12

- [X] T078 [US12] Add chart library dependency in `frontend/package.json`
- [X] T079 [US12] Implement `RaceFlowChart.vue` in `frontend/src/components/RaceFlowChart.vue`
- [X] T080 [US12] Add Race Flow and Statistics tabs to `frontend/src/views/RaceDetails.vue`

**Checkpoint**: Race analytics visualizations available on race detail pages

---

## Phase 16: User Story 13 - Authentication & Authorization (Priority: P2)

**Goal**: JWT authentication and role-based access control for admin and timer operations

**Independent Test**: Protected routes reject unauthenticated requests; admin role required for write operations; viewer role allows read-only access

### Tests for User Story 13

- [ ] T081 [P] [US13] Auth service tests in `backend/internal/services/auth_service_test.go`
- [ ] T082 [P] [US13] Auth middleware tests in `backend/internal/middleware/middleware_test.go`

### Implementation for User Story 13

- [ ] T083 [US13] Implement AuthService with JWT in `backend/internal/services/auth_service.go`
- [ ] T084 [US13] Implement login handler in `backend/internal/handlers/auth.go`
- [ ] T085 [US13] Wire JWT and RBAC middleware to write routes in `backend/internal/middleware/middleware.go` and `backend/cmd/server/main.go`

**Checkpoint**: Admin and timer endpoints secured with role-based permissions

---

## Phase 17: Polish & Cross-Cutting Concerns

**Purpose**: Infrastructure, CI, and validation across all stories

- [ ] T086 [P] Create `.dockerignore` in repository root
- [ ] T087 [P] Add GitHub Actions CI workflow in `.github/workflows/ci.yml`
- [ ] T088 [P] Add Redis caching for leaderboard queries in `backend/internal/services/results_service.go`
- [ ] T089 Run full backend test suite via `docker-compose -f docker-compose.test.yml run backend-test`
- [ ] T090 [P] Run frontend test suite via `docker-compose -f docker-compose.test.yml run frontend-test`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phases 1–6**: ✅ Complete (MVP backend: events, races, participants)
- **Phase 7 (US4)**: Depends on Phase 2 — blocks US5 and US6
- **Phase 8 (US5)**: Depends on US4 (checkpoints required for timing records)
- **Phase 9 (US6)**: Depends on US5 (manual entry creates timing records)
- **Phase 10 (US7)**: Depends on Phases 3–5 API — blocks US8, US9, US10 frontend
- **Phase 11 (US8)**: Depends on US7 and US5 (leaderboard data)
- **Phase 11b (TypeScript)**: Depends on US7–US8; blocks US9–US12 frontend work
- **Phase 12 (US9)**: Depends on US8 and US6 (RFID/manual entry APIs)
- **Phase 13 (US10)**: Depends on US7 and Phase 11b; HTML prototype (T068) before Vue refinement (T069)
- **Phase 14 (US11)**: Depends on US9 (offline queue for timing station)
- **Phase 15 (US12)**: Depends on US8 (RaceDetails page exists)
- **Phase 16 (US13)**: Can start after Phase 2; should complete before production deploy
- **Phase 17**: Depends on all desired user stories

### User Story Dependencies

| Story | Depends On | Blocks |
|-------|-----------|--------|
| US4 Checkpoints & Categories | Foundation | US5, US6 |
| US5 Timing & Results | US4 | US6, US8 |
| US6 RFID API | US5 | US9 |
| US7 Frontend State | US1–US3 API | US8, US9, US10 |
| US8 Timing UI | US5, US7 | US9, US12 |
| TS Migration | US7, US8 | US9, US10, US11, US12 |
| US9 Live Timing UI | US6, US8, TS Migration | US11 |
| US10 Landing Page | US7, TS Migration | — |
| US11 PWA Offline | US9 | — |
| US12 Visualization | US8 | — |
| US13 Auth | Foundation | Production deploy |

### Within Each User Story

- Tests MUST be written and FAIL before implementation (TDD)
- Services before handlers
- Handlers before route wiring
- Backend API before dependent frontend views

### Parallel Opportunities

- T028–T030 (US4 tests) can run in parallel
- T031–T032 (US4 services) can run in parallel after tests
- T036–T038 (US5 tests) can run in parallel
- T049–T050, T051–T052, T054 (US7) can run in parallel
- T055–T057 (US8 tests) and T059–T061 (US8 components) can run in parallel
- T091–T094 (TypeScript migration) can run in parallel after toolchain (T091) is in place
- T086–T088 (Polish) can run in parallel

---

## Parallel Example: User Story 5

```bash
# Launch all tests for User Story 5 together:
Task: "Timing service tests in backend/internal/services/timing_service_test.go"
Task: "Results calculation tests in backend/internal/services/results_service_test.go"
Task: "Timing handler tests in backend/internal/handlers/handlers_test.go"

# After tests fail, implement services sequentially:
Task: "Implement TimingService in backend/internal/services/timing_service.go"
Task: "Implement ResultsService in backend/internal/services/results_service.go"
Task: "Replace timing handler stubs in backend/internal/handlers/timing.go"
```

---

## Implementation Strategy

### Completed MVP (Phases 1–6)

Backend CRUD for events, races, and participants is done and tested. Timing and RFID handlers remain stubbed (`501 Not Implemented`).

### Next MVP Increment (Phases 7–8)

1. Complete Phase 7: US4 — checkpoints and categories
2. Complete Phase 8: US5 — timing records, results, leaderboards
3. **STOP and VALIDATE**: Record a timing event via API and retrieve ranked results
4. Demo core timing loop before frontend or RFID hardware work

### Incremental Delivery After Timing MVP

1. US6 RFID API → field-ready tag lookup and manual entry
2. US7 + US8 → browser-based timing index and detail pages
3. Phase 11b → migrate frontend to TypeScript before new UI work
4. US9 → live timing station for race day operators
5. US10 → landing page (HTML prototype first per constitution)
6. US13 → secure admin write operations
7. US11 + US12 → offline resilience and analytics (P3)

### Parallel Team Strategy

With multiple developers after Phase 6:

- **Developer A**: US4 → US5 (backend timing core)
- **Developer B**: US7 → US8 (frontend timing views)
- **Developer C**: US6 (RFID integration layer)

Stories integrate at API boundaries without blocking each other once US5 and US7 are underway.

---

## Notes

- [P] tasks = different files, no dependencies on incomplete tasks in the same group
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Verify tests fail before implementing
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Frontend modules use TypeScript (`.ts` for services, stores, router, tests; `<script setup lang="ts">` in Vue SFCs)
- US7–US8 were initially implemented in JavaScript; Phase 11b migrates those files to TypeScript before new frontend stories
- Frontend views referenced in `frontend/src/router/index.ts`: `LiveTiming.vue` is created in US9; `EventDetails.vue` and `RaceDetails.vue` exist from US8
