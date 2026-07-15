# Proxmark3 Driver + Logical Tag UUID — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a real Proxmark3 CLI reader/writer and rework RFID identity so each racer has a stable logical UUID that is written into tag user memory and returned on every read (silicon UID is never the scoring key).

**Architecture:** Change the `rfid.Reader` contract so `WritePayload` / `Poll` operate on logical UUID bytes in user memory; update `RFIDService.WriteTag` to write the participant’s existing logical UUID without requiring a client-supplied silicon UID; keep `MockReader` for CI by simulating on-chip memory; add `CLIProxmarkReader` selected via `RFID_HARDWARE=true`.

**Tech Stack:** Go (`backend/internal/rfid`, services, handlers), Vue Racers program UI, Python Bluffet seed (logical UUIDs), Docker hardware overlay

**Spec:** `docs/superpowers/specs/2026-07-15-proxmark3-tag-uuid-design.md`

**Implement this plan before** `docs/superpowers/plans/2026-07-15-hardware-bluffet-e2e.md`.

---

## File map

| File | Responsibility |
|---|---|
| `backend/internal/rfid/proxmark3.go` | Reader interface + MockReader chip-memory sim + DefaultReader |
| `backend/internal/rfid/cli_proxmark.go` | Real pm3 CLI bridge |
| `backend/internal/rfid/cli_proxmark_test.go` | Fake CLI tests |
| `backend/internal/rfid/payload.go` | Encode/decode logical UUID ↔ 16-byte user memory |
| `backend/internal/services/rfid_service.go` | WriteTag(participant) writes logical UUID; no silicon associate |
| `backend/internal/handlers/rfid.go` + `requests.go` | write-tag body without required `tag_uid` |
| `backend/internal/handlers/participants.go` | PostParticipantTag programs by participant only |
| `frontend/src/views/Racers.vue` | Program tag = write only (no UID field required) |
| `database/seed/generate_bluffet_seed.py` | Seed logical UUIDs (uuid5), not DEMO-TAG strings if migrating |
| `docker-compose.hardware.yml` | RFID_HARDWARE=true, inject off |
| `specs/002-rfid-race-scanner/quickstart.md` | Document hardware + logical UUID model |

---

### Task 1: Payload codec (UUID ↔ 16 bytes)

**Files:**
- Create: `backend/internal/rfid/payload.go`
- Create: `backend/internal/rfid/payload_test.go`

- [ ] **Step 1: Failing tests**

```go
package rfid

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestPayloadRoundTrip(t *testing.T) {
	id := uuid.MustParse("1441674d-a011-471a-a601-722b88b117f5")
	raw, err := EncodeLogicalUUID(id.String())
	require.NoError(t, err)
	require.Len(t, raw, 16)
	got, err := DecodeLogicalUUID(raw)
	require.NoError(t, err)
	require.Equal(t, id.String(), got)
}

func TestEncodeRejectsNonUUID(t *testing.T) {
	_, err := EncodeLogicalUUID("DEMO-TAG-0001")
	require.Error(t, err)
}
```

- [ ] **Step 2: Run — expect fail**

```powershell
cd backend; go test ./internal/rfid/ -run TestPayload -count=1
```

- [ ] **Step 3: Implement**

```go
package rfid

import (
	"fmt"

	"github.com/google/uuid"
)

func EncodeLogicalUUID(s string) ([]byte, error) {
	id, err := uuid.Parse(strings.TrimSpace(s))
	if err != nil {
		return nil, fmt.Errorf("logical tag id must be a UUID: %w", err)
	}
	b := id[:]
	out := make([]byte, 16)
	copy(out, b[:])
	return out, nil
}

func DecodeLogicalUUID(b []byte) (string, error) {
	if len(b) < 16 {
		return "", fmt.Errorf("payload too short")
	}
	id, err := uuid.FromBytes(b[:16])
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

func EncodeLogicalUUIDHex(s string) (string, error) {
	raw, err := EncodeLogicalUUID(s)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", raw), nil
}
```

Add `"strings"` import.

- [ ] **Step 4: Pass + commit**

```powershell
cd backend; go test ./internal/rfid/ -run TestPayload -count=1
git add backend/internal/rfid/payload.go backend/internal/rfid/payload_test.go
git commit -m "feat(rfid): add logical UUID payload codec for tag user memory"
```

---

### Task 2: Change Reader interface to payload-centric API

**Files:**
- Modify: `backend/internal/rfid/proxmark3.go`
- Modify: `backend/internal/rfid/proxmark3_test.go`
- Modify: all callers of `WriteTag(tagUID, participantID)`

- [ ] **Step 1: Update interface + MockReader chip memory**

```go
// Reader abstracts Proxmark3 (or compatible) RFID hardware.
// Identity is always a logical UUID stored in user memory — never the silicon UID.
type Reader interface {
	// WriteLogicalUUID programs the chip currently on the antenna with the racer's logical UUID.
	WriteLogicalUUID(logicalUUID string) error
	// Poll reads user memory and returns the logical UUID, or "" if no tag / empty.
	Poll() (logicalUUID string, err error)
	IsAvailable() bool
}

// MockReader simulates a single chip's user memory for CI.
type MockReader struct {
	Available bool
	WriteErr  error
	mu        sync.Mutex
	memory    string   // last programmed logical UUID
	queue     []string // inject/scripted polls (optional override)
}

func (m *MockReader) WriteLogicalUUID(logicalUUID string) error {
	if !m.Available {
		return ErrHardwareUnavailable
	}
	if m.WriteErr != nil {
		return m.WriteErr
	}
	if _, err := EncodeLogicalUUID(logicalUUID); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.memory = strings.ToLower(logicalUUID)
	return nil
}

func (m *MockReader) Poll() (string, error) {
	if !m.Available {
		return "", ErrHardwareUnavailable
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.queue) > 0 {
		uid := m.queue[0]
		m.queue = m.queue[1:]
		return uid, nil
	}
	return m.memory, nil
}

func (m *MockReader) Enqueue(logicalUUID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.queue = append(m.queue, strings.ToLower(logicalUUID))
}
```

Keep deprecated wrappers if needed briefly:

```go
// WriteTag is deprecated: use WriteLogicalUUID. Ignores siliconUID.
func (m *MockReader) WriteTag(siliconUID, logicalUUID string) error {
	_ = siliconUID
	return m.WriteLogicalUUID(logicalUUID)
}
```

Prefer updating `Proxmark3` wrapper to call `WriteLogicalUUID` only and delete old signature from the interface in the same PR.

- [ ] **Step 2: Fix compile across backend** (`go test ./...` will list breaks)

Update `Proxmark3.WriteTag` → `WriteLogicalUUID`, `RFIDService`, inject still `Enqueue`s logical ids.

- [ ] **Step 3: Commit**

```powershell
git add backend/internal/rfid backend/internal/services backend/internal/handlers
git commit -m "refactor(rfid): Reader API uses logical UUID payload, not silicon UID"
```

---

### Task 3: RFIDService.WriteTag programs logical UUID only

**Files:**
- Modify: `backend/internal/services/rfid_service.go`
- Modify: `backend/internal/services/rfid_service_test.go`

- [ ] **Step 1: Failing test — write uses participant logical id; Poll returns it**

```go
func TestWriteTag_ProgramsLogicalUUIDWithoutSilicon(t *testing.T) {
	db := /* existing setup */
	mock := rfid.NewMockReader()
	svc := NewRFIDService(db, mock)
	p := /* create participant */
	logical := uuid.New().String()
	_, err := svc.AssociateTag(p.ID.UUID(), logical)
	require.NoError(t, err)

	_, err = svc.WriteTag(p.ID.UUID()) // no silicon arg
	require.NoError(t, err)

	got, err := mock.Poll()
	require.NoError(t, err)
	require.Equal(t, strings.ToLower(logical), got)

	found, err := svc.LookupParticipantByUID(got)
	require.NoError(t, err)
	require.Equal(t, p.ID, found.ID)
}
```

- [ ] **Step 2: Implement service**

```go
// WriteTag programs the chip on the antenna with this participant's logical RFID UUID.
// Ensures an association exists (creates one with a new UUID if the racer has none).
func (s *RFIDService) WriteTag(participantID uuid.UUID) (*models.Participant, error) {
	if participantID == uuid.Nil {
		return nil, fmt.Errorf("%w: participant_id is required", ErrInvalidRFIDInput)
	}
	partSvc := NewParticipantService(s.db)
	participant, err := partSvc.GetParticipant(participantID)
	if err != nil {
		return nil, err
	}

	logical, err := s.ensureLogicalTagUUID(participantID)
	if err != nil {
		return nil, err
	}

	device := rfid.NewProxmark3(s.reader)
	if err := device.WriteLogicalUUID(logical); err != nil {
		return nil, err
	}
	return partSvc.GetParticipant(participantID)
}

func (s *RFIDService) ensureLogicalTagUUID(participantID uuid.UUID) (string, error) {
	var assoc models.RFIDTagAssociation
	err := s.db.Where("participant_id = ? AND active = ?", participantID, true).
		Order("created_at ASC").First(&assoc).Error
	if err == nil {
		return assoc.TagUID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}
	logical := uuid.New().String()
	if _, err := s.AssociateTag(participantID, logical); err != nil {
		return "", err
	}
	return logical, nil
}
```

Remove the old `WriteTag(participantID, tagUID string)` silicon association path. Keep `AssociateTag` for explicit multi-identity if still needed; default race-day path is one logical UUID per racer.

- [ ] **Step 3: Handler + request**

```go
type writeRFIDTagRequest struct {
	ParticipantID string `json:"participant_id" binding:"required"`
	// tag_uid removed — logical id comes from DB
}
```

```go
func (h *Handlers) WriteRFIDTag(c *gin.Context) {
	var req writeRFIDTagRequest
	// bind, resolve participant id, call services.RFID.WriteTag(id)
}
```

Update `PostParticipantTag` to call `WriteTag(participantID)` when body omits `tag_uid`, or always program logical UUID and ignore body `tag_uid` except for mock/CI inject of pre-seeded ids.

- [ ] **Step 4: Tests green + commit**

```powershell
cd backend; go test ./internal/services/ ./internal/handlers/ ./internal/rfid/ -count=1
git commit -am "feat(rfid): WriteTag programs racer logical UUID onto chip"
```

---

### Task 4: Seed logical UUIDs instead of DEMO-TAG strings

**Files:**
- Modify: `database/seed/generate_bluffet_seed.py`
- Regenerate: `database/seed/03-bluffet-2026.sql`
- Modify: `frontend/e2e/fixtures/rfid.ts` (`demoTags` → known uuid5 values)

- [ ] **Step 1: Generator uses uuid5 for tag identities**

```python
tag_uid = stable_uuid(f"tag:{race_key}:{i + 1}")  # already returns uuid string
```

Remove `DEMO-TAG-{seq:04d}`. Keep association rows with those UUIDs.

- [ ] **Step 2: Update e2e fixture constants**

```typescript
// Compute or hardcode the first three stable tag UUIDs from seed
demoTags: [
  // uuid5(EVENT_ID, 'tag:12-hour:1') etc. — paste exact strings from generator print
] as const
```

Add a tiny Python one-liner in the generator `__main__` print of first 3 tag UUIDs when `DEBUG_TAGS=1`.

- [ ] **Step 3: Regenerate SQL + fix tests that hardcode `DEMO-TAG-0001`**

```powershell
python database/seed/generate_bluffet_seed.py
cd backend; go test ./... -count=1
cd ../frontend; npm test -- --run
```

- [ ] **Step 4: Commit**

```powershell
git add database/seed frontend/e2e/fixtures/rfid.ts backend frontend
git commit -m "feat(seed): use logical UUIDs as RFID tag identities"
```

---

### Task 5: Racers UI — program without silicon UID field

**Files:**
- Modify: `frontend/src/views/Racers.vue`
- Modify: `frontend/src/services/api.ts` (writeTag / addTag payloads)
- Modify: related unit/e2e tests

- [ ] **Step 1: UI**

Replace UID input + write with a single action:

```html
<p class="muted">
  Place a tag on the Proxmark3, then write. This programs this racer’s permanent
  RFID UUID onto the chip. Replacement tags get the same UUID.
</p>
<button data-testid="program-tag-write" @click="writeTag(racer)">Write tag</button>
```

```typescript
async function writeTag(racer: Racer) {
  programming.value = true
  try {
    await rfidApi.writeTag({ participant_id: racer.id })
    // refresh tag list
  } finally {
    programming.value = false
  }
}
```

- [ ] **Step 2: Update `racers-page.spec.ts`** — no longer fill `program-tag-uid`

- [ ] **Step 3: Commit**

```powershell
git commit -am "fix(ui): program tag writes racer logical UUID without silicon UID entry"
```

---

### Task 6: CLI Proxmark3 reader

**Files:**
- Create: `backend/internal/rfid/cli_proxmark.go`
- Create: `backend/internal/rfid/cli_proxmark_test.go`
- Modify: `DefaultReader()` in `proxmark3.go`
- Modify: `backend/internal/config/config.go`

- [ ] **Step 1: Tests with fake executable** (same pattern as prior plan — echo UID/payload hex)

Poll must **read user memory**, not silicon UID. Example pm3 command (tune on hardware):

```text
hf mfu rdbl --blk 4
```

Parse 16 data bytes → `DecodeLogicalUUID`.

Write:

```text
hf mfu wrbl --blk 4 -d <32 hex chars from EncodeLogicalUUIDHex>
```

Exact command strings will be adjusted in Task 7 smoke against the attached reader/card type (NTAG vs MIFARE). Keep commands in constants at top of `cli_proxmark.go`.

- [ ] **Step 2: DefaultReader**

```go
func DefaultReader() Reader {
	if os.Getenv("GO_ENV") == "test" {
		return NewMockReader()
	}
	if hw := os.Getenv("RFID_HARDWARE"); hw == "1" || strings.EqualFold(hw, "true") {
		cli := os.Getenv("PROXMARK3_CLI")
		if cli == "" {
			cli = "pm3"
		}
		return NewCLIProxmarkReader(CLIProxmarkConfig{
			CLIPath: cli,
			Port:    os.Getenv("PROXMARK3_PORT"),
			Enabled: true,
		})
	}
	if os.Getenv("PROXMARK3_ENABLED") == "true" {
		return NewMockReader()
	}
	return &NoOpReader{}
}
```

- [ ] **Step 3: Commit**

```powershell
git add backend/internal/rfid backend/internal/config
git commit -m "feat(rfid): CLI Proxmark3 reader for logical UUID user-memory I/O"
```

---

### Task 7: Hardware smoke + compose overlay

**Files:**
- Create: `docker-compose.hardware.yml`
- Modify: `specs/002-rfid-race-scanner/quickstart.md`
- Create: `frontend/e2e/hardware-bluffet/proxmark-smoke.spec.ts` **or** a small Go integration under `backend/internal/rfid` gated by `RFID_HARDWARE=true`

- [ ] **Step 1: Overlay**

```yaml
services:
  backend:
    environment:
      RFID_INJECT: "false"
      RFID_HARDWARE: "true"
      PROXMARK3_ENABLED: "true"
      PROXMARK3_CLI: ${PROXMARK3_CLI:-pm3}
      PROXMARK3_PORT: ${PROXMARK3_PORT:-}
```

- [ ] **Step 2: Manual/automated smoke**

1. Place tag on reader  
2. PIN auth → `POST /api/rfid/write-tag` `{participant_id}` for a seeded racer  
3. Confirm `Poll` / live WS delivers that racer’s logical UUID  
4. Confirm lap/test-read popup  

Tune `hf …` commands until green.

- [ ] **Step 3: Document in quickstart** — logical UUID model + Windows USBIPD notes  

- [ ] **Step 4: Commit**

```powershell
git add docker-compose.hardware.yml specs/002-rfid-race-scanner/quickstart.md
git commit -m "chore: hardware compose overlay and Proxmark logical-UUID docs"
```

---

### Task 8: Optional admin peek endpoint

**Files:**
- Modify: `backend/internal/handlers/rfid.go`, `main.go`

- [ ] **Step 1:** `GET /api/rfid/read-payload` (admin) → single `Poll()` → `{ "logical_uuid": "…" }` for harness/debug  

- [ ] **Step 2: Test + commit**

```powershell
git commit -am "feat(rfid): admin read-payload endpoint for hardware debug"
```

---

## Self-review

| Design requirement | Task |
|---|---|
| Logical UUID written to tag | 1–3, 5 |
| UUID never changes per racer | 3 (`ensureLogicalTagUUID`), 4 |
| Silicon UID not scoring key | 2–3, 6 |
| Real Proxmark driver | 6–7 |
| Mock CI still works | 2, inject/Enqueue |
| Seed + fixtures updated | 4 |
| UI matches “writes racer UUID” | 5 |

**Removed from Bluffet e2e plan:** silicon UID reassignment; treating Proxmark driver as part of the dress-rehearsal plan.

---

## Execution handoff

Plan complete and saved to `docs/superpowers/plans/2026-07-15-proxmark3-tag-uuid.md`.

**Implement this plan first**, then `2026-07-15-hardware-bluffet-e2e.md`.

Two execution options for this Proxmark plan:

**1. Subagent-Driven (recommended)** — fresh subagent per task  

**2. Inline Execution** — execute in this session with checkpoints  

Which approach?
