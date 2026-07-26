# Race Teams Implementation Plan

> **For agentic workers:** Execute tasks in order. Prefer TDD for scoring/API. Do **not** git commit unless the user asks. User directed continuous execution to completion including dress rehearsal.

**Goal:** Per-race teams with average-lap scoring, live/results team boards, race-flow team filter, management UI, CSV, seed, and dress-rehearsal coverage.

**Architecture:** `teams` + `participants.team_id`; compute averages on read; Individuals|Teams toggle on live boards.

**Tech Stack:** Go/GORM/Postgres, Vue 3, Playwright hardware e2e, Bluffet seed generator.

---

### Task 1: Schema + model

**Files:**
- Create: `database/migrations/05-race-teams.sql`
- Modify: `backend/internal/models/models.go`
- Modify: `backend/internal/database/database.go` (AutoMigrate Team before Participant)

- [ ] Add `Team` model; `Participant.TeamID`; Race `Teams` relation
- [ ] SQL migration creating `teams` + `participants.team_id`
- [ ] Unit test AutoMigrate / model create in `models_test.go` or team service test setup

### Task 2: Team service + handlers

**Files:**
- Create: `backend/internal/services/team_service.go`, `team_service_test.go`
- Create: `backend/internal/handlers/teams.go`
- Modify: `backend/cmd/server/main.go`, `handlers.go` / services wiring
- Modify: participant create/update to accept `team_id` (same race validation)

- [ ] CRUD + SetMembers (≥2 or 0; max 12; same race)
- [ ] HTTP routes under `/api/races/:id/teams` and `/api/teams/:id`

### Task 3: Team scoring + live + scan

**Files:**
- Modify: `backend/internal/services/results_service.go` (+ tests)
- Modify: `backend/internal/services/scan/scan_service.go` (+ tests)
- Modify: live/results handlers if response types change

- [ ] `BuildTeamLeaderboard(raceID)` with avg + tie-breaks
- [ ] Attach `leaderboard_teams` on `LiveRaceView`
- [ ] ScanResult team fields when participant has team

### Task 4: CSV round-trip

**Files:**
- Modify: `backend/internal/services/csv_export.go` (+ tests)

- [ ] Export `#SECTION,teams` before participants; add `team_id` to participants
- [ ] Import teams then participants with team_id

### Task 5: Frontend live + flow + popup + racers

**Files:**
- Modify: `frontend/src/views/EventLive.vue`, race results view if any
- Modify: `frontend/src/components/RaceFlowChart.vue`, `raceFlowData.ts` (+ tests)
- Modify: lap popup / scan UI components
- Modify: `frontend/src/views/Racers.vue` (+ API client)
- Modify: types/API wrappers as needed

- [ ] Individuals | Teams toggle
- [ ] Team filter on race flow
- [ ] Team line on lap celebration/popup
- [ ] PIN Teams management

### Task 6: Seed + dress rehearsal

**Files:**
- Modify: `database/seed/generate_bluffet_seed.py` → regenerate hardware SQL
- Modify: `frontend/e2e/hardware-bluffet/` assertions (spectator team board / filter / popup)
- Run: regenerate seed; run `npm run test:e2e:bluffet-hardware` (or documented prod harness)

- [ ] 4×4 teams on 12h
- [ ] E2E asserts team surfaces
- [ ] Run dress rehearsal and report outcome

---

## Self-review checklist

1. Avg = sum/roster_count with DNS in denominator — Task 3
2. Race flow team filter — Task 5
3. Dress rehearsal teams + run — Task 6
4. No cross-race / multi-team — enforced in Task 2
