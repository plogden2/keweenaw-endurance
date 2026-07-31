# Bib-Associated RFID Tags Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Associate RFID tags with event-scoped bibs (not racers), so chips can be programmed the night before and racers get a bib number on race morning.

**Architecture:** Add `Bib` (`event_id` + unique `bib_number`, UUID written on chips). Point `RFIDTagAssociation` at `bib_id`. Scan resolves `tag_uid → Bib → Participant` (by event + bib number), with dual-resolve for legacy participant-UUID chips and a distinct `unassigned_bib` result. Event **Bibs** page handles bulk create/program; Racers assigns bibs, warns on zero tags, and supports ad-hoc writes.

**Tech Stack:** Go/GORM, Gin, Vue 3, Vitest, Playwright, Proxmark3 write path (existing `WriteLogicalUUID`)

**Spec:** `docs/superpowers/specs/2026-07-31-bib-tag-association-design.md`

---

## File map

| File | Action |
|------|--------|
| `backend/internal/models/models.go` | Add `Bib`; change `RFIDTagAssociation` to `bib_id` |
| `backend/internal/database/database.go` | AutoMigrate `Bib` + data migration/backfill |
| `database/migrations/06-bib-tag-association.sql` | Document schema |
| `backend/internal/services/bib_service.go` | New: ensure/list/bulk-create/get tags |
| `backend/internal/services/rfid_service.go` | Associate/Write by bib; participant write via current bib |
| `backend/internal/services/participant_service.go` | Event-wide bib uniqueness; ensure Bib on create/update |
| `backend/internal/services/scan/scan_service.go` | Resolve via bib; `unassigned_bib`; legacy participant.id |
| `backend/internal/services/csv_export.go` | `bibs` section; tags → `bib_id` |
| `backend/internal/handlers/bibs.go` | Event bibs HTTP API |
| `backend/cmd/server/main.go` | Routes |
| `frontend/src/views/EventBibs.vue` | New inventory UI |
| `frontend/src/views/Racers.vue` | Warn / confirm / write-to-bib |
| `frontend/src/router/index.ts` | `/events/:eventId/bibs` |
| `frontend/src/services/api.ts` | `eventBibsApi` + write payloads |
| `specs/002-rfid-race-scanner/{spec,data-model,contracts}/*` | FR/model drift |
| `frontend/e2e/racers-page.spec.ts` (+ new bibs e2e) | Program bib UUID |

---

### Task 1: Bib model + association FK + AutoMigrate backfill

**Files:**
- Modify: `backend/internal/models/models.go`
- Modify: `backend/internal/models/models_test.go`
- Modify: `backend/internal/database/database.go`
- Create: `database/migrations/06-bib-tag-association.sql`
- Modify: all test `AutoMigrate(...)` lists that include `RFIDTagAssociation` to also include `&models.Bib{}`

- [ ] **Step 1: Failing model test** — `TestBibModel` creates Bib with `EventID` + `BibNumber`; unique `(event_id, bib_number)` enforced.

- [ ] **Step 2: Update models**

```go
// Bib is an event-scoped race number; RFID chips store Bib.ID.
type Bib struct {
	ID        uuidutil.PublicUUID `gorm:"type:uuid;primary_key" json:"id"`
	EventID   uuidutil.PublicUUID `gorm:"type:uuid;not null;uniqueIndex:idx_bibs_event_number" json:"event_id"`
	BibNumber string              `gorm:"type:varchar(20);not null;uniqueIndex:idx_bibs_event_number" json:"bib_number"`
	CreatedAt time.Time           `gorm:"autoCreateTime" json:"created_at"`

	Event           Event                `gorm:"foreignKey:EventID" json:"event,omitempty"`
	TagAssociations []RFIDTagAssociation `gorm:"foreignKey:BibID" json:"tag_associations,omitempty"`
}

func (Bib) TableName() string { return "bibs" }

// RFIDTagAssociation — replace ParticipantID with:
BibID uuidutil.PublicUUID `gorm:"type:uuid;not null;index" json:"bib_id"`
Bib   Bib                 `gorm:"foreignKey:BibID" json:"bib,omitempty"`
// Remove Participant / ParticipantID fields from this struct.
```

Add `BeforeCreate` for Bib UUID if Participant pattern uses one.

- [ ] **Step 3: Migrate()** — AutoMigrate `&models.Bib{}` before associations. After AutoMigrate, run `migrateTagAssociationsToBibs(db)`:

  1. If column `rfid_tag_associations.participant_id` still exists (information_schema / pragma), for each distinct association join participant → race → event:
     - `Ensure` Bib `(event_id, participant.bib_number)`
     - `UPDATE rfid_tag_associations SET bib_id = ? WHERE id = ?`
  2. For participants with `bib_number` but no Bib yet, create Bib rows (even without tags).
  3. Drop `participant_id` when safe (Postgres: `ALTER TABLE ... DROP COLUMN IF EXISTS participant_id`). On SQLite test DB, recreate table or leave orphan column only if drop is painful — prefer dialect-aware drop; tests use fresh AutoMigrate so new shape is enough.

- [ ] **Step 4: SQL doc** `database/migrations/06-bib-tag-association.sql` matching final schema (`bibs`, associations with `bib_id`).

- [ ] **Step 5: Run** `go test ./internal/models/ ./internal/database/ -count=1` — pass. Commit: `feat(db): add event Bib and tag associations by bib_id`

---

### Task 2: BibService (ensure, list, bulk create)

**Files:**
- Create: `backend/internal/services/bib_service.go`
- Create: `backend/internal/services/bib_service_test.go`

- [ ] **Step 1: Failing tests**
  - `EnsureBib(eventID, "42")` creates once; second call returns same id
  - `BulkCreateBibs(eventID, 1, 5)` creates 1–5; idempotent for existing numbers
  - `ListBibs(eventID)` returns bib number, tag count, optional assigned participant `{id, name, race_id}` (join participants across event races where `bib_number` matches)
  - Reject empty bib number / invalid range (`from > to`, negative, span > 500)

- [ ] **Step 2: Implement**

```go
type BibService struct{ db *gorm.DB }

func NewBibService(db *gorm.DB) *BibService

func (s *BibService) EnsureBib(eventID uuid.UUID, bibNumber string) (*models.Bib, error)
func (s *BibService) BulkCreateBibs(eventID uuid.UUID, from, to int) ([]models.Bib, error)
func (s *BibService) ListBibs(eventID uuid.UUID) ([]BibListItem, error)
func (s *BibService) GetBib(eventID, bibID uuid.UUID) (*models.Bib, error)
func (s *BibService) ListBibTags(bibID uuid.UUID) ([]models.RFIDTagAssociation, error)

type BibListItem struct {
	ID              uuidutil.PublicUUID  `json:"id"`
	BibNumber       string               `json:"bib_number"`
	TagCount        int                  `json:"tag_count"`
	TagUIDs         []string             `json:"tag_uids,omitempty"`
	ParticipantID   *uuidutil.PublicUUID `json:"participant_id,omitempty"`
	ParticipantName string               `json:"participant_name,omitempty"`
	RaceID          *uuidutil.PublicUUID `json:"race_id,omitempty"`
}
```

- [ ] **Step 3: Wire** `Services.Bib` in services constructor (same pattern as RFID/Participant).

- [ ] **Step 4: Run** `go test ./internal/services/ -count=1 -run Bib` — pass. Commit: `feat(bibs): add BibService ensure/list/bulk-create`

---

### Task 3: RFIDService — associate/write by bib

**Files:**
- Modify: `backend/internal/services/rfid_service.go`
- Modify: `backend/internal/services/rfid_service_test.go`

- [ ] **Step 1: Failing tests**
  - `AssociateTagToBib(bibID, tagUID)` creates row; second tag on same bib OK
  - Rebind: tag on bib A then associate to bib B → row moves to B (last write wins), no error
  - `WriteTagForBib(bibID)` writes **bib.ID** string via `WriteLogicalUUID` / bridge; ensures association with that UUID if missing
  - `WriteTag(participantID)` ensures Bib for participant’s event+bib_number, then `WriteTagForBib`
  - `ListParticipantTags` returns associations for participant’s current Bib
  - `LookupParticipantByUID`: association → load Bib → find Participant in that event with matching bib_number; else legacy `rfid_tag_uid`; else try parse UUID as `participants.id`

- [ ] **Step 2: Implement** — replace `AssociateTag(participantID, …)` internals:

```go
func (s *RFIDService) AssociateTagToBib(bibID uuid.UUID, tagUID string) (*models.RFIDTagAssociation, error)
// AssociateTag keeps signature for callers: resolve participant → EnsureBib → AssociateTagToBib
func (s *RFIDService) WriteTagForBib(bibID uuid.UUID) (*models.Bib, error)
func (s *RFIDService) WriteTag(participantID uuid.UUID) (*models.Participant, error) // via bib
```

`ensureLogicalTagUUID` becomes bib-centric: logical UUID = `bib.ID.String()` (lowercase consistent with existing encode). Association `tag_uid` stores that UUID (same as today stores participant logical id).

Mirror `participants.rfid_tag_uid` to latest tag for display only (optional, keep if tests rely on it).

- [ ] **Step 3: Fix all compile breakages** in handlers/tests that construct `RFIDTagAssociation{ParticipantID: ...}` → use `BibID` after EnsureBib.

- [ ] **Step 4: Run** `go test ./internal/services/ -count=1 -run RFID` — pass. Commit: `feat(rfid): associate and write tags to bib UUID`

---

### Task 4: ParticipantService — event-wide bib uniqueness + ensure Bib

**Files:**
- Modify: `backend/internal/services/participant_service.go`
- Modify: `backend/internal/services/participant_service_test.go`

- [ ] **Step 1: Failing tests**
  - Create participant with bib `7` in race A; create another in race B (same event) with bib `7` → error `bib_number must be unique within event`
  - Update moving to an in-use bib → same error
  - Create/update calls EnsureBib for the event
  - `attachTagUIDs` loads tags via bib (not `participant_id` on associations)
  - Optional response helper: after create/update, `TagUIDs` empty when bib has no tags (frontend warns)

- [ ] **Step 2: Implement uniqueness** — resolve `event_id` from race; query:

```go
// participants JOIN races ON races.id = participants.race_id
// WHERE races.event_id = ? AND participants.bib_number = ? AND participants.id != ?
```

Error message: `bib_number must be unique within event`.

On successful create/update of `bib_number`, `NewBibService(s.db).EnsureBib(eventID, bibNumber)`.

- [ ] **Step 3: Run** `go test ./internal/services/ -count=1 -run Participant` — pass. Commit: `feat(participants): event-scoped bib uniqueness and EnsureBib`

---

### Task 5: Scan resolve — bib path, unassigned, legacy participant id

**Files:**
- Modify: `backend/internal/services/scan/scan_service.go`
- Modify: `backend/internal/services/scan/scan_service_test.go`
- Modify: frontend scan result handling if it switches on `result` string (grep `unknown_tag`)

- [ ] **Step 1: Failing tests**
  - Tag associated to Bib with no participant → `result == "unassigned_bib"`, message set, `BibNumber` populated, no timing row
  - Tag associated to Bib with participant → lap/test_read as today
  - No association; chip UUID equals `participant.id` in event → resolve legacy (dual-resolve)
  - Prefer association/bib path when both would match
  - Cooldown still by `participant_id`

- [ ] **Step 2: Implement**

```go
const ResultUnassignedBib = "unassigned_bib"

func (s *ScanService) resolveParticipant(eventID uuid.UUID, tagUID string) (*models.Participant, *models.Bib, error)
```

Or return a small struct. In `ProcessScan`:
- if bib found and participant nil → `ScanResult{Result: ResultUnassignedBib, BibNumber: bib.BibNumber, Message: "…"}`
- if err not found → try `WHERE participants.id = ?` joined to race.event_id (legacy)
- else unknown_tag

- [ ] **Step 3: UI feedback** — wherever lap popup / reader GUI maps results, show distinct copy for `unassigned_bib` (frontend composable `useReaderStation` + reader-gui if needed). Minimum: API returns the result; live popup treats non-lap like unknown with message.

- [ ] **Step 4: Run** `go test ./internal/services/scan/ -count=1` — pass. Commit: `feat(scan): resolve tags via bib with unassigned state`

---

### Task 6: HTTP API — event bibs + write-tag by bib

**Files:**
- Create: `backend/internal/handlers/bibs.go`
- Modify: `backend/internal/handlers/rfid.go`, `participants.go`, `requests.go`
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/internal/handlers/handlers_test.go`
- Modify: `specs/002-rfid-race-scanner/contracts/api-rfid-scanner.md`

- [ ] **Step 1: Routes** (PIN/`timerWrite` or admin — match Racers write auth)

```go
api.GET("/events/:id/bibs", handlers.ListEventBibs)
// write group:
api.POST("/events/:id/bibs/bulk", handlers.BulkCreateEventBibs) // body: { "from": 1, "to": 100 }
api.POST("/events/:id/bibs/:bibId/tags", handlers.PostBibTag)   // empty → WriteTagForBib; {tag_uid} → AssociateTagToBib
api.GET("/events/:id/bibs/:bibId/tags", handlers.ListBibTags)
```

Keep `POST /api/rfid/write-tag` with `participant_id` (existing). Extend body optional `bib_id` — if set, write that bib.

```go
type writeRFIDTagRequest struct {
	ParticipantID string `json:"participant_id"`
	BibID         string `json:"bib_id"`
	RaceID        string `json:"race_id"`
	LogicalUUID   string `json:"logical_uuid"` // ignored for new writes; bib id wins
}
```

- [ ] **Step 2: Handler tests** — bulk create, list shows assignment, write tag for bib returns bib id as logical, participant write still works, event-wide bib clash on participant create returns 400.

- [ ] **Step 3: Refresh live CSV** after bib/tag mutations (same hook as participant tags).

- [ ] **Step 4: Commit:** `feat(api): event bibs inventory and bib tag write endpoints`

---

### Task 7: Live CSV — bibs section + tag bib_id

**Files:**
- Modify: `backend/internal/services/csv_export.go`
- Modify: `backend/internal/services/csv_export_test.go`
- Modify: `specs/002-rfid-race-scanner/contracts/csv-race-export.md` (if present)

- [ ] **Step 1: Failing round-trip test** — Bibs + tag→bib survive `BuildCSV` → `ImportCSV`.

- [ ] **Step 2: Export**
  - Section `bibs`: `id,event_id,bib_number,created_at`
  - Section `tags`: `id,bib_id,tag_uid,created_at` (replace `participant_id`)

- [ ] **Step 3: Import** — create bibs before tags; tags attach to `bib_id`; participants still import with `bib_number` (EnsureBib).

- [ ] **Step 4: Run** `go test ./internal/services/ -count=1 -run CSV` — pass. Commit: `feat(csv): export/import event bibs and tag→bib links`

---

### Task 8: Frontend API clients + types

**Files:**
- Modify: `frontend/src/types/models.ts` (or inline in api.ts if that’s the pattern)
- Modify: `frontend/src/services/api.ts`
- Modify: any `WriteTagPayload` / `shouldRouteWriteTagLocal`

- [ ] **Step 1: Add**

```ts
export interface BibListItem {
  id: string
  bib_number: string
  tag_count: number
  tag_uids?: string[]
  participant_id?: string
  participant_name?: string
  race_id?: string
}

export const eventBibsApi = {
  list: (eventId: string) => api.get(`/events/${eventId}/bibs`),
  bulkCreate: (eventId: string, from: number, to: number) =>
    api.post(`/events/${eventId}/bibs/bulk`, { from, to }),
  listTags: (eventId: string, bibId: string) =>
    api.get(`/events/${eventId}/bibs/${bibId}/tags`),
  addTag: (eventId: string, bibId: string, body?: { tag_uid?: string }) =>
    api.post(`/events/${eventId}/bibs/${bibId}/tags`, body ?? {}),
}

// Extend writeTag payload:
// { participant_id?: string, bib_id?: string, race_id?: string, logical_uuid?: string }
```

- [ ] **Step 2: Unit smoke** if api has tests; else skip. Commit: `feat(frontend): eventBibsApi client`

---

### Task 9: EventBibs.vue page + nav

**Files:**
- Create: `frontend/src/views/EventBibs.vue`
- Create: `frontend/src/views/EventBibs.test.ts`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/views/EventLive.vue` (ops link)
- Match PIN patterns from `Racers.vue` / `EventTaps.vue`

- [ ] **Step 1: Failing Vitest** — renders inventory; bulk create calls API; program tag calls write with `bib_id` (mock `rfidApi.writeTag` or `eventBibsApi.addTag`).

- [ ] **Step 2: Implement UI**
  - Route: `/events/:eventId/bibs`, `data-testid="event-bibs-page"`
  - PIN gate for mutations
  - Table: bib #, tag count, assigned racer / “unassigned”
  - Bulk form: from/to + Create (`data-testid="bibs-bulk-create"`)
  - Per-row Program (`data-testid="bib-program-tag"`) → write bib UUID via existing bridge write path (`logical_uuid` = bib.id)

- [ ] **Step 3: Ops link** on EventLive next to Racers: “Bibs” → `/events/${eventId}/bibs`

- [ ] **Step 4: Run** `npx vitest run src/views/EventBibs.test.ts` — pass. Commit: `feat(frontend): event Bibs inventory page`

---

### Task 10: Racers.vue — warn, confirm, write via bib

**Files:**
- Modify: `frontend/src/views/Racers.vue`
- Modify: `frontend/src/views/Racers.test.ts`

- [ ] **Step 1: Failing tests**
  - After save bib when `tag_uids` empty → show warn (`data-testid="bib-no-tags-warn"`)
  - Changing bib when current racer has tags → confirm dialog; cancel leaves old bib
  - `writeTag` still works (payload may include `logical_uuid` from bib / participant tags)

- [ ] **Step 2: Implement**
  - On successful bib save: if `!(data.tag_uids?.length)` show warning text
  - Before bib save when `racer.tag_uids?.length` or (optional) target clash: `window.confirm` or inline confirm
  - Program tag: keep calling write-tag with `participant_id` (backend routes to bib)

- [ ] **Step 3: Run** `npx vitest run src/views/Racers.test.ts` — pass. Commit: `feat(racers): bib assign warn and confirm on tagged reassignment`

---

### Task 11: Spec / contract drift + e2e

**Files:**
- Modify: `specs/002-rfid-race-scanner/spec.md` (FR-005 → bib UUID; FR-024 → event-unique; Racers story)
- Modify: `specs/002-rfid-race-scanner/data-model.md`
- Modify: `specs/002-rfid-race-scanner/contracts/api-rfid-scanner.md`
- Modify: `frontend/e2e/racers-page.spec.ts`
- Create: `frontend/e2e/event-bibs.spec.ts` (bulk create + program mock inject path if hardware unavailable)

- [ ] **Step 1: Update specs** to match design (no racer-UUID-as-primary-payload).

- [ ] **Step 2: E2E**
  - Racers: program tag → association resolves to racer via bib (adjust assertion from “racer logical UUID” to bib id / successful tag list)
  - Event bibs: bulk 1–3, open program, inject/associate tag, assign bib on Racers, scan yields lap (reuse inject endpoint patterns from `live-lap-timing` / `racers-page`)

- [ ] **Step 3: Run** targeted Playwright specs in CI-like env if available; otherwise mark commands in commit message. Commit: `test+docs: bib-tag association e2e and spec updates`

---

### Task 12: Full regression gate

- [ ] **Step 1:** `docker compose -f docker-compose.test.yml run --rm backend-test` (or `cd backend && go test $(go list ./... | grep -vE '/cmd/reader-gui$|/cmd/reader-setup$') -count=1`)
- [ ] **Step 2:** `cd frontend && npx vitest run` for touched views
- [ ] **Step 3:** Fix fallout (handler fixtures, seed demos creating associations, reader-gui write if it assumes participant UUID only — update to bib UUID when writing from bib inventory; participant path unchanged via API)
- [ ] **Step 4: Final commit** if fixes needed: `fix: bib-tag association regression cleanup`

---

## Self-review (plan vs spec)

| Spec requirement | Task |
|------------------|------|
| Bib entity, chip = bib UUID | 1, 3 |
| Associations → bib_id, multi-tag, no revoke | 1, 3 |
| Event-unique bib numbers | 4 |
| Event Bibs bulk + program | 2, 6, 9 |
| Racers assign + warn + ad-hoc write | 4, 10 |
| Bib edit allow + confirm when tags/laps | 10 (tags); laps: confirm if `tag_uids.length` OR document confirm whenever tags on source/target — enhance with API flag later if needed |
| Scan unassigned / dual-resolve / cooldown | 5 |
| Migration + CSV + multi-station | 1, 7 |
| Spec FR drift | 11 |
| Legacy dual-resolve | 3, 5 |

**Note on confirm+laps:** v1 confirm when source/target has tags (from `tag_uids` / bib list). Optional stretch: if changing bib after racer has scored laps, same confirm (check lap count from existing participant fields if exposed; otherwise tags-only confirm satisfies design “tags or scored laps” partially — prefer adding `has_scored_laps` only if list payload already has lap counts; else confirm on any bib change when race is `active`).

---

## Out of scope (do not implement)

Per design: revoke, moving tags with person, per-race overlapping bibs, auto kids/adults ranges, per-tag cooldown, field chip rewrite tool.
