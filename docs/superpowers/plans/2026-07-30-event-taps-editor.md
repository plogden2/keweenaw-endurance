# Event Taps Editor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Event-scoped paginated taps table with void/restore and Add-tap dialog (searchable racer + karaoke toggle), replacing race-scoped Manual entry as the ops editor.

**Architecture:** New `GET/POST /api/events/:eventId/taps` plus `GET /api/events/:eventId/participants`. Frontend `EventTaps.vue` at `/events/:eventId/taps`; old `/timing/live/:raceId` redirects. Standalone karaoke via `TimingService` create with `record_type=karaoke_bonus` and null `source_lap_id` (not `AddKaraokeBonus`). Reuse existing void/restore.

**Tech Stack:** Go/GORM, Gin, Vue 3, Pinia PIN auth, Vitest, Playwright optional

**Spec:** `docs/superpowers/specs/2026-07-30-event-taps-editor-design.md`

---

### Task 1: TimingService — list event taps + create event tap (TDD)

**Files:**
- Modify: `backend/internal/services/timing_service.go`
- Modify: `backend/internal/services/timing_service_test.go`
- Modify: `specs/002-rfid-race-scanner/data-model.md` (karaoke `source_lap_id` optional)

- [ ] **Step 1: Failing tests** in `timing_service_test.go`:
  - `ListRecordsByEvent` returns records across races, `timestamp DESC`, includes voided, paginates (`page`/`limit` → slice + total)
  - Optional `raceID` filter and `q` (bib/name ILIKE)
  - `CreateEventTap(eventID, participantID, karaokeBonus bool, timestamp *time.Time)`:
    - normal → `rfid_lap` at finish checkpoint, `source_lap_id` nil
    - karaoke → `karaoke_bonus`, `source_lap_id` nil
    - participant not in event → error
    - race missing finish checkpoint → error

- [ ] **Step 2: Implement** helpers. Suggested signatures:

```go
func (s *TimingService) ListRecordsByEvent(eventID uuid.UUID, page, limit int, raceID *uuid.UUID, q string) ([]models.TimingRecord, int64, error)

type CreateEventTapInput struct {
	EventID       uuid.UUID
	ParticipantID uuid.UUID
	KaraokeBonus  bool
	Timestamp     *time.Time // nil = now UTC
	DeviceID      string     // default "manual-event-taps"
}

func (s *TimingService) CreateEventTap(input CreateEventTapInput) (*models.TimingRecord, error)
```

Implementation notes:
- Join `participants` → `races` where `races.event_id = ?`
- Preload `Participant`, `Checkpoint`, and race name (preload `Participant.Race` if association exists, or select race name via join into a DTO — prefer Preload Race on Participant)
- Finish checkpoint: `WHERE race_id = ? AND checkpoint_type = 'finish'` (match existing checkpoint model field name)
- Create via existing `CreateRecord` after filling fields, or direct create if `CreateRecord` forces defaults that conflict
- DeviceID: `"manual-event-taps"`; SyncStatus: `"synced"` (or resolve via sync helper if ManualEntry does)

- [ ] **Step 3: Run** `go test ./internal/services/ -count=1 -run 'ListRecordsByEvent|CreateEventTap|Timing'` — pass. Commit.

---

### Task 2: ParticipantService — list by event (TDD)

**Files:**
- Modify: `backend/internal/services/participant_service.go` (or equivalent)
- Modify: matching `*_test.go`

- [ ] **Step 1: Failing test** — `ListParticipantsByEvent(eventID, page, limit, q)` returns participants whose `race.event_id` matches; `q` filters bib/name; includes race for UI label.

- [ ] **Step 2: Implement** with same pagination envelope style as `ListParticipants`.

- [ ] **Step 3: Run tests. Commit.**

---

### Task 3: HTTP handlers + routes

**Files:**
- Create or modify: `backend/internal/handlers/event_taps.go` (or extend `timing.go` / `events.go`)
- Modify: `backend/internal/handlers/participants.go` (event participants) if cleaner
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/internal/handlers/handlers_test.go`
- Modify: `specs/002-rfid-race-scanner/contracts/api-rfid-scanner.md`

- [ ] **Step 1: Routes**

```go
api.GET("/events/:id/taps", handlers.ListEventTaps)
api.GET("/events/:id/participants", handlers.ListEventParticipants) // if not already present
// under timerWrite group:
api.POST("/events/:id/taps", append(timerWrite, handlers.CreateEventTap)...)
```

- [ ] **Step 2: Handlers**
  - Parse `page`/`limit` (defaults 1/50), optional `race_id`, `q`
  - Create body: `participant_id`, `karaoke_bonus`, optional `timestamp`
  - POST unauth → 401/403; success → `201` with record JSON
  - After create: trigger live CSV / live stream if other create paths do (match ManualEntry / CreateRecord side effects)

- [ ] **Step 3: Handler tests** for list pagination, create lap, create karaoke null source, auth.

- [ ] **Step 4: Commit.**

---

### Task 4: Frontend API types + clients

**Files:**
- Modify: `frontend/src/types/models.ts`
- Modify: `frontend/src/services/api.ts`

- [ ] Add:

```ts
export interface CreateEventTapPayload {
  participant_id: string
  karaoke_bonus: boolean
  timestamp?: string
}

// Participant may already exist; ensure race?: Race for dropdown labels
```

```ts
export const eventTapsApi = {
  list: (eventId: string, params?: { page?: number; limit?: number; race_id?: string; q?: string }) =>
    apiClient.get<PaginatedResponse<TimingRecord>>(`/api/events/${eventId}/taps`, { params }),
  create: (eventId: string, payload: CreateEventTapPayload) =>
    apiClient.post<TimingRecord>(`/api/events/${eventId}/taps`, payload),
}

export const eventParticipantsApi = {
  list: (eventId: string, params?: { page?: number; limit?: number; q?: string }) =>
    apiClient.get<PaginatedResponse<Participant>>(`/api/events/${eventId}/participants`, { params }),
}
```

Ensure `TimingRecord.participant` can expose race name (nested `race` or `race_name`). Align with backend JSON.

- [ ] Commit.

---

### Task 5: EventTaps.vue + AddTapDialog + router

**Files:**
- Create: `frontend/src/views/EventTaps.vue`
- Create: `frontend/src/components/AddTapDialog.vue` (or inline dialog in view if small)
- Create: `frontend/src/views/EventTaps.test.ts`
- Create: `frontend/src/components/AddTapDialog.test.ts` (optional if covered by view tests)
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/views/LiveTiming.vue` — replace with redirect component OR router-only redirect
- Modify links: `EventLive.vue`, `PinUnlock.vue`, `EventDetails.vue`, `RaceDetails.vue`
- Modify: `docs/production-reader.md` Manual entry section

- [ ] **Router**

```ts
{
  path: '/events/:eventId/taps',
  name: 'event-taps',
  component: () => import('@/views/EventTaps.vue'),
},
{
  path: '/timing/live/:raceId',
  name: 'live-timing',
  // redirect: load race → eventId, or small redirect view
  component: () => import('@/views/LiveTimingRedirect.vue'),
},
```

Prefer a tiny `LiveTimingRedirect.vue` that fetches race by id and `router.replace(`/events/${eventId}/taps`)`.

- [ ] **EventTaps.vue**
  - Load event name; call `eventTapsApi.list` with `page`/`limit=50`
  - Table columns: Time, Race, Bib, Name, Type, Sync, Actions
  - Type labels: `rfid_lap`→Lap, `karaoke_bonus`→Karaoke, `checkpoint_pass`→Checkpoint
  - Voided: gray + badge; delete icon calls `timingRecordsApi.voidRecord` with confirm; restore with confirm
  - Actions only when `pinAuth.isAuthenticated`; Add tap button same
  - Pagination prev/next + “Page X of Y” (or total-based)
  - Reuse LiveTiming voided CSS patterns / chrome theme variables

- [ ] **AddTapDialog**
  - Debounced search via `eventParticipantsApi.list`
  - Options: `#bib Name (Race name)`
  - Karaoke toggle (`data-testid="karaoke-toggle"`)
  - Submit → `eventTapsApi.create` with `karaoke_bonus`
  - Close + emit refresh on success

- [ ] **Tests** (Vitest): renders rows newest-first from mocked API; voided class; PIN gates actions; dialog submits karaoke flag.

- [ ] Update Manual entry / Taps links to `/events/${eventId}/taps` (ensure eventId available; from race fetch if needed).

- [ ] Update `docs/production-reader.md`.

- [ ] Commit.

---

### Task 6: Verification

- [ ] `go test ./internal/services/ ./internal/handlers/ -count=1`
- [ ] `cd frontend && npm test -- --run EventTaps AddTapDialog LiveTiming` (adjust names)
- [ ] Fix failures. Final commit if needed.
- [ ] Optional e2e smoke if harness already covers Manual entry — update selectors/paths.

---

## Spec coverage checklist

| Spec item | Task |
|-----------|------|
| Paginated event list DESC | 1, 3, 5 |
| Void/restore + always show voided | 5 (reuse API) |
| Add tap dialog searchable racer | 2, 3, 5 |
| Karaoke-only standalone | 1, 3, 5 |
| Route + redirect + link rewires | 5 |
| No migration | — |
| Docs / data-model | 1, 5 |
