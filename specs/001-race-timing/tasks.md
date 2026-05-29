# Tasks: Keweenaw Endurance Race Timing System

**Input**: Design documents from `/specs/001-race-timing/`

**Prerequisites**: plan.md, spec.md

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [X] T001 Create Docker Compose development environment in `docker-compose.yml`
- [X] T002 Initialize Go backend project in `backend/` with Gin framework
- [X] T003 [P] Initialize Vue.js frontend project in `frontend/` with Vite
- [X] T004 [P] Configure PostgreSQL schema in `database/init/01-init.sql`
- [X] T005 [P] Configure test infrastructure in `docker-compose.test.yml`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before user story work

- [X] T006 Define GORM models in `backend/internal/models/models.go`
- [X] T007 Implement database migration in `backend/internal/database/database.go`
- [X] T008 [P] Implement config loading in `backend/internal/config/config.go`
- [X] T009 [P] Implement CORS and security middleware in `backend/internal/middleware/middleware.go`
- [X] T010 Wire API routes in `backend/cmd/server/main.go`
- [X] T011 Fix Go module version compatibility in `backend/go.mod` and Dockerfiles

**Checkpoint**: Foundation ready — user story implementation can begin

---

## Phase 3: User Story 1 - Event Management (Priority: P1) 🎯 MVP

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

## Phase 4: User Story 2 - Race Management (Priority: P1)

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

## Phase 5: User Story 3 - Participant Registration (Priority: P1)

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

## Phase 6: Polish & Cross-Cutting Concerns

- [X] T025 [P] Update `.gitignore` with essential patterns for Go, Node.js, Docker
- [X] T026 Run full backend test suite via `docker-compose -f docker-compose.test.yml run backend-test`
- [X] T027 Remove stub handler implementations from `backend/internal/handlers/handlers.go`

---

## Dependencies

- Phase 1 → Phase 2 → Phase 3 → Phase 4 → Phase 5 → Phase 6
- T012–T013 before T014–T016 (TDD)
- T017–T018 before T019–T020 (TDD)
- T021–T022 before T023–T024 (TDD)
