# Hardware East Bluffet e2e Dress Rehearsal — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver a Proxmark3-backed, wall-clock East Bluffet dress-rehearsal harness that an agent can start and monitor every minute, including chaos (reader crash, 5‑min API outage), 1080p×3 video → 1440p side-by-side, and an iterate-until-clean issues loop.

**Architecture:** Fix real-hardware gaps (USB/serial Proxmark reader + single-tag reprogram/reassign), add a compressed-duration Bluffet seed flag, then build a dedicated Playwright hardware project under `frontend/e2e/hardware-bluffet/` that owns the timeline while writing `status.json` / `issues.md` for the monitoring agent. Spectators use desktop + iPhone 13 projects; API outage uses Playwright offline/route abort while the backend keeps polling Proxmark into local Postgres.

**Tech Stack:** Go (`backend/internal/rfid`), Vue/Playwright, Python seed generator, ffmpeg, Docker Compose, Cursor agent runbook

**Spec:** `docs/superpowers/specs/2026-07-15-hardware-bluffet-e2e-design.md`

---

## File map

| File | Responsibility |
|---|---|
| `backend/internal/rfid/serial_proxmark.go` | Real Proxmark3 reader via `pm3` CLI (or serial) |
| `backend/internal/rfid/serial_proxmark_test.go` | Unit tests with fake exec |
| `backend/internal/rfid/proxmark3.go` | `DefaultReader()` selects mock vs serial vs noop |
| `backend/internal/config/config.go` | `PROXMARK3_PORT`, `PROXMARK3_CLI`, hardware mode |
| `backend/internal/services/rfid_service.go` | `WriteTag` reassigns single physical UID (reprogram) |
| `database/seed/generate_bluffet_seed.py` | `--durations=30,15,5` + race display names |
| `docker-compose.hardware.yml` | `RFID_INJECT=false`, hardware env, optional device notes |
| `frontend/e2e/hardware-bluffet/playwright.config.ts` | Video 1080p, long timeout, 3 projects |
| `frontend/e2e/hardware-bluffet/dress-rehearsal.spec.ts` | Single orchestrated ~32 min test |
| `frontend/e2e/hardware-bluffet/lib/*.ts` | status, issues, clock, lap-engine, roster, chaos, spectators, video |
| `frontend/e2e/hardware-bluffet/README.md` | Agent entrypoint + 1‑min monitor loop |
| `scripts/compose-bluffet-hardware-video.ps1` | ffmpeg 3×1080p → 1440p side-by-side |
| `frontend/package.json` | `test:e2e:bluffet-hardware` script |
| `.gitignore` | `e2e-artifacts/` |

---

### Task 1: Allow WriteTag to reassign one physical UID

**Why:** `AssociateTag` rejects moving a `tag_uid` to another participant. One physical chip cannot cycle racers without reprogram/reassign.

**Files:**
- Modify: `backend/internal/services/rfid_service.go` (`WriteTag`, optionally `AssociateTag`)
- Modify: `backend/internal/services/rfid_service_test.go`

- [ ] **Step 1: Write failing test for reassignment via WriteTag**

Add to `rfid_service_test.go`:

```go
func TestWriteTag_ReassignsExistingUIDToNewParticipant(t *testing.T) {
	db := setupRFIDTestDB(t) // use existing test DB helper in this file
	mock := rfid.NewMockReader()
	svc := NewRFIDService(db, mock)

	p1 := createTestParticipant(t, db) // existing helper or inline create
	p2 := createTestParticipant(t, db)

	_, err := svc.WriteTag(p1.ID.UUID(), "HW-UID-1")
	require.NoError(t, err)

	_, err = svc.WriteTag(p2.ID.UUID(), "HW-UID-1")
	require.NoError(t, err)

	got, err := svc.LookupParticipantByUID("HW-UID-1")
	require.NoError(t, err)
	assert.Equal(t, p2.ID, got.ID)

	var old models.RFIDTagAssociation
	err = db.Where("participant_id = ? AND tag_uid = ?", p1.ID, "HW-UID-1").First(&old).Error
	require.NoError(t, err)
	assert.False(t, old.Active)
}
```

Adapt helper names to match the file’s existing patterns (`TestAssociateTag_*`).

- [ ] **Step 2: Run test — expect fail**

```powershell
cd backend; go test ./internal/services/ -run TestWriteTag_ReassignsExistingUIDToNewParticipant -count=1
```

Expected: FAIL (tag already associated).

- [ ] **Step 3: Implement reassignment inside WriteTag**

In `WriteTag`, after successful `device.WriteTag(...)`, before/instead of plain `AssociateTag`:

```go
// Reprogram path: deactivate any active rows for this tag_uid owned by others.
if err := s.db.Model(&models.RFIDTagAssociation{}).
	Where("tag_uid = ? AND participant_id <> ? AND active = ?", tagUID, participantID, true).
	Update("active", false).Error; err != nil {
	return nil, err
}
// Also clear legacy mirror on previous owners when needed.
_ = s.db.Model(&models.Participant{}).
	Where("rfid_tag_uid = ? AND id <> ?", tagUID, participantID).
	Update("rfid_tag_uid", "").Error

if _, err := s.AssociateTag(participantID, tagUID); err != nil {
	return nil, err
}
```

Keep `AssociateTag` itself strict for non-write callers (multi-tag “no silent steal”); only `WriteTag` reprograms.

- [ ] **Step 4: Re-run test — expect pass**

```powershell
cd backend; go test ./internal/services/ -run TestWriteTag_Reassigns -count=1
```

- [ ] **Step 5: Commit**

```powershell
git add backend/internal/services/rfid_service.go backend/internal/services/rfid_service_test.go
git commit -m "fix(rfid): allow WriteTag to reassign a physical UID between racers"
```

---

### Task 2: Real Proxmark3 reader via CLI bridge

**Why:** `DefaultReader()` always returns `MockReader` when `PROXMARK3_ENABLED=true`. There is no USB path.

**Files:**
- Create: `backend/internal/rfid/serial_proxmark.go`
- Create: `backend/internal/rfid/serial_proxmark_test.go`
- Modify: `backend/internal/rfid/proxmark3.go`
- Modify: `backend/internal/config/config.go`

- [ ] **Step 1: Failing test for CLI reader Poll/WriteTag**

```go
package rfid

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCLIProxmarkReader_PollParsesUID(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "pm3fake.cmd")
	// Windows: write a .cmd that echoes a hf search-like line
	body := "@echo off\r\necho [#] UID: DEADBEEF0102\r\n"
	if runtime.GOOS != "windows" {
		script = filepath.Join(dir, "pm3fake.sh")
		body = "#!/bin/sh\necho '[#] UID: DEADBEEF0102'\n"
		require.NoError(t, os.WriteFile(script, []byte(body), 0o755))
	} else {
		require.NoError(t, os.WriteFile(script, []byte(body), 0o644))
	}
	r := NewCLIProxmarkReader(CLIProxmarkConfig{
		CLIPath: script,
		Enabled: true,
	})
	uid, err := r.Poll()
	require.NoError(t, err)
	require.Equal(t, "DEADBEEF0102", uid)
}
```

- [ ] **Step 2: Run — expect fail (type missing)**

```powershell
cd backend; go test ./internal/rfid/ -run TestCLIProxmarkReader_PollParsesUID -count=1
```

- [ ] **Step 3: Implement `CLIProxmarkReader`**

```go
// serial_proxmark.go
package rfid

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

var uidLine = regexp.MustCompile(`(?i)\bUID:\s*([0-9A-F]+)`)

type CLIProxmarkConfig struct {
	CLIPath string // e.g. pm3 or full path
	Port    string // optional COM3 / /dev/ttyACM0 passed as -p
	Enabled bool
	Timeout time.Duration
	// execCommand injectable for tests
	execCommand func(name string, arg ...string) *exec.Cmd
}

type CLIProxmarkReader struct {
	cfg CLIProxmarkConfig
	mu  sync.Mutex
}

func NewCLIProxmarkReader(cfg CLIProxmarkConfig) *CLIProxmarkReader {
	if cfg.Timeout == 0 {
		cfg.Timeout = 8 * time.Second
	}
	if cfg.execCommand == nil {
		cfg.execCommand = exec.Command
	}
	return &CLIProxmarkReader{cfg: cfg}
}

func (r *CLIProxmarkReader) IsAvailable() bool {
	return r != nil && r.cfg.Enabled && strings.TrimSpace(r.cfg.CLIPath) != ""
}

func (r *CLIProxmarkReader) run(pm3Args string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	args := []string{}
	if r.cfg.Port != "" {
		args = append(args, "-p", r.cfg.Port)
	}
	args = append(args, "-c", pm3Args)
	cmd := r.cfg.execCommand(r.cfg.CLIPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	out := stdout.String() + stderr.String()
	if err != nil {
		return out, fmt.Errorf("pm3: %w: %s", err, out)
	}
	return out, nil
}

func (r *CLIProxmarkReader) Poll() (string, error) {
	if !r.IsAvailable() {
		return "", ErrHardwareUnavailable
	}
	// Prefer ISO14443A search; adjust -c string to match installed pm3 build.
	out, err := r.run("hf search")
	if err != nil {
		// empty field is not fatal
		if strings.Contains(strings.ToLower(out), "no tag") || strings.Contains(out, "No tag") {
			return "", nil
		}
		return "", err
	}
	m := uidLine.FindStringSubmatch(out)
	if m == nil {
		return "", nil
	}
	return strings.ToUpper(m[1]), nil
}

func (r *CLIProxmarkReader) WriteTag(tagUID, participantID string) error {
	if !r.IsAvailable() {
		return ErrHardwareUnavailable
	}
	// NTAG/MIFARE user-memory write — command string must match local pm3 scripts.
	// Minimum contract: leave chip present; association uses hardware UID from Poll.
	// Optional: write participantID into user pages for forensic dumps.
	_, err := r.run(fmt.Sprintf("hf mf wrbl --blk 4 -d %s", encodePayload(participantID)))
	if err != nil {
		// If write command unsupported on this card, still OK if Poll UID works and DB reassigns.
		// Surface error so harness can log an issue.
		return err
	}
	_ = tagUID
	return nil
}

func encodePayload(participantID string) string {
	// 16 hex bytes padded/truncated — keep deterministic for tests
	h := fmt.Sprintf("%x", participantID)
	if len(h) > 32 {
		h = h[:32]
	}
	for len(h) < 32 {
		h += "0"
	}
	return h
}

// DefaultReader selection
func DefaultReader() Reader {
	if os.Getenv("GO_ENV") == "test" {
		return NewMockReader()
	}
	if os.Getenv("RFID_HARDWARE") == "1" || os.Getenv("RFID_HARDWARE") == "true" {
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

Tune the exact `hf …` strings on the attached reader in Task 8 smoke; keep regex UID parsing stable.

- [ ] **Step 4: Wire config env docs in `config.go`**

Add to `RFIDConfig`:

```go
Hardware     bool   // RFID_HARDWARE
ProxmarkCLI  string // PROXMARK3_CLI
ProxmarkPort string // PROXMARK3_PORT
```

Load via existing `getEnv` helpers. Do not break current compose mock defaults.

- [ ] **Step 5: Tests pass + commit**

```powershell
cd backend; go test ./internal/rfid/ -count=1
git add backend/internal/rfid backend/internal/config/config.go
git commit -m "feat(rfid): add CLI Proxmark3 reader selected by RFID_HARDWARE"
```

---

### Task 3: Hardware compose overlay

**Files:**
- Create: `docker-compose.hardware.yml`
- Modify: `specs/002-rfid-race-scanner/quickstart.md` (short pointer only)

- [ ] **Step 1: Add overlay**

```yaml
# docker-compose.hardware.yml — use with: docker compose -f docker-compose.yml -f docker-compose.hardware.yml up --build
services:
  backend:
    environment:
      RFID_INJECT: "false"
      RFID_HARDWARE: "true"
      PROXMARK3_ENABLED: "true"
      PROXMARK3_CLI: ${PROXMARK3_CLI:-pm3}
      PROXMARK3_PORT: ${PROXMARK3_PORT:-}
      HOSTED_API_URL: ${HOSTED_API_URL:-}
    # Uncomment when USBIPD → WSL device is available:
    # devices:
    #   - "${PROXMARK3_DEVICE:-/dev/ttyACM0}:/dev/ttyACM0"
```

**Windows note in README (Task 11):** Prefer running `pm3` on the Windows host and the backend with `RFID_HARDWARE=true` bound to host network **or** USBIPD into WSL. Document whichever path works on this laptop during smoke.

- [ ] **Step 2: Commit**

```powershell
git add docker-compose.hardware.yml
git commit -m "chore: add docker-compose.hardware overlay for Proxmark dress rehearsal"
```

---

### Task 4: Compressed Bluffet seed durations

**Files:**
- Modify: `database/seed/generate_bluffet_seed.py`
- Regenerate: `database/seed/03-bluffet-2026.sql` only when explicitly using flag for hardware runs — **keep default generator output as 720/360/90 for CI**

- [ ] **Step 1: Add CLI flag**

At top of `main()` parse:

```python
import argparse
parser = argparse.ArgumentParser()
parser.add_argument(
    "--durations",
    default="720,360,90",
    help="Comma durations minutes for 12h,6h,kids races",
)
parser.add_argument(
    "--names",
    default="12 Hour,6 Hour,90-Minute Kids",
    help="Comma race display names",
)
parser.add_argument(
    "--output",
    default=str(OUTPUT_SQL),
)
args = parser.parse_args()
durs = [int(x) for x in args.durations.split(",")]
names = [x.strip() for x in args.names.split(",")]
assert len(durs) == 3 and len(names) == 3
```

Apply `durs[i]` / `names[i]` into the `races` list. Keep stable UUIDs.

- [ ] **Step 2: Generate hardware SQL artifact (separate file)**

```powershell
python database/seed/generate_bluffet_seed.py --durations=30,15,5 --names="30 Minute,15 Minute,5-Minute Kids" --output=database/seed/03-bluffet-2026-hardware.sql
```

- [ ] **Step 3: Commit generator + hardware SQL**

```powershell
git add database/seed/generate_bluffet_seed.py database/seed/03-bluffet-2026-hardware.sql
git commit -m "feat(seed): add compressed-duration Bluffet seed for hardware e2e"
```

---

### Task 5: Artifact helpers (status + issues)

**Files:**
- Create: `frontend/e2e/hardware-bluffet/lib/artifacts.ts`
- Create: `frontend/e2e/hardware-bluffet/lib/clock.ts`
- Add: `e2e-artifacts/` to `.gitignore`

- [ ] **Step 1: Implement artifact writers**

```typescript
// frontend/e2e/hardware-bluffet/lib/artifacts.ts
import fs from 'node:fs'
import path from 'node:path'

export type IssueSeverity = 'critical' | 'minor' | 'idea'
export type Issue = {
  ts: string
  severity: IssueSeverity
  title: string
  details: string
  phase: string
  screenshot?: string
}

export type RunStatus = {
  runId: string
  phase: string
  tZeroIso: string
  nowIso: string
  elapsedSec: number
  lapsScored: number
  pendingSync: number
  chaos: { apiOutage: boolean; readerDown: boolean }
  lastProxmark?: string
  lastError?: string
  healthy: boolean
}

export function createRunDir(root = path.join(process.cwd(), '..', '..', 'e2e-artifacts', 'bluffet-hardware')) {
  const runId = new Date().toISOString().replace(/[:.]/g, '-')
  const dir = path.join(root, runId)
  fs.mkdirSync(dir, { recursive: true })
  fs.writeFileSync(path.join(dir, 'issues.jsonl'), '')
  fs.writeFileSync(path.join(dir, 'issues.md'), '# Issues\n\n')
  return { runId, dir }
}

export function writeStatus(dir: string, status: RunStatus) {
  fs.writeFileSync(path.join(dir, 'status.json'), JSON.stringify(status, null, 2))
}

export function appendIssue(dir: string, issue: Issue) {
  fs.appendFileSync(path.join(dir, 'issues.jsonl'), JSON.stringify(issue) + '\n')
  const shot = issue.screenshot ? ` ![shot](${issue.screenshot})` : ''
  fs.appendFileSync(
    path.join(dir, 'issues.md'),
    `- **[${issue.severity}]** ${issue.title} — ${issue.details} _(phase=${issue.phase}, ${issue.ts})_ ${shot}\n`,
  )
}
```

```typescript
// frontend/e2e/hardware-bluffet/lib/clock.ts
export function sampleLapDelayMs(rng = Math.random): number {
  // Approximate mean ~60s, clamp [30s, 180s] via triangular-ish mix
  const u = rng()
  const ms = 30_000 + u * u * 150_000 // skew toward shorter end of range
  return Math.min(180_000, Math.max(30_000, Math.round(ms)))
}

export function sleep(ms: number) {
  return new Promise((r) => setTimeout(r, ms))
}
```

- [ ] **Step 2: Gitignore + commit**

```gitignore
e2e-artifacts/
```

```powershell
git add frontend/e2e/hardware-bluffet/lib .gitignore
git commit -m "feat(e2e): add hardware Bluffet status and issues artifact helpers"
```

---

### Task 6: Roster + lap engine

**Files:**
- Create: `frontend/e2e/hardware-bluffet/lib/roster.ts`
- Create: `frontend/e2e/hardware-bluffet/lib/lapEngine.ts`
- Create: `frontend/e2e/hardware-bluffet/lib/proxmark.ts`

- [ ] **Step 1: Implement roster selection**

```typescript
// roster.ts
import type { APIRequestContext } from '@playwright/test'
import { API_BASE, BLUFFET, pinToken } from '../../fixtures/rfid'

export type Racer = {
  id: string
  raceId: string
  bib: string
  firstName: string
  lastName: string
  tagUid: string // seeded DEMO-TAG; physical UID discovered at runtime
}

export async function loadSeededRacers(request: APIRequestContext): Promise<Racer[]> {
  const token = await pinToken(request)
  const racers: Racer[] = []
  for (const race of Object.values(BLUFFET.races)) {
    const res = await request.get(`${API_BASE}/api/races/${race.id}/participants`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    if (!res.ok()) throw new Error(`participants ${res.status()}`)
    const body = await res.json()
    const rows = body.data ?? body
    for (const p of rows) {
      racers.push({
        id: p.id,
        raceId: race.id,
        bib: String(p.bib_number ?? p.bibNumber ?? ''),
        firstName: p.first_name ?? p.firstName,
        lastName: p.last_name ?? p.lastName,
        tagUid: p.rfid_tag_uid ?? p.rfidTagUid ?? '',
      })
    }
  }
  return racers
}

/** Deterministic picks from sorted ids */
export function pickNoShows(racers: Racer[], n = 9): Set<string> {
  const sorted = [...racers].sort((a, b) => a.id.localeCompare(b.id))
  return new Set(sorted.slice(0, n).map((r) => r.id))
}

export function pickDnfs(activeIds: string[], n = 10, seed = 42): Set<string> {
  const arr = [...activeIds].sort()
  const out = new Set<string>()
  let x = seed
  while (out.size < Math.min(n, arr.length)) {
    x = (x * 1103515245 + 12345) & 0x7fffffff
    out.add(arr[x % arr.length])
  }
  return out
}
```

- [ ] **Step 2: Proxmark write→await read helper**

```typescript
// proxmark.ts
import type { APIRequestContext, Page } from '@playwright/test'
import { API_BASE, pinToken } from '../../fixtures/rfid'

export async function reprogramAndAwaitLap(opts: {
  request: APIRequestContext
  readerPage: Page
  participantId: string
  physicalUid: string
  eventId: string
  timeoutMs?: number
}) {
  const token = await pinToken(opts.request)
  const write = await opts.request.post(`${API_BASE}/api/rfid/write-tag`, {
    headers: { Authorization: `Bearer ${token}` },
    data: { participant_id: opts.participantId, tag_uid: opts.physicalUid },
  })
  if (!write.ok()) {
    throw new Error(`write-tag failed: ${write.status()} ${await write.text()}`)
  }
  // Real read comes from backend poll → WS → scan popup on reader page
  await opts.readerPage.getByTestId('scan-popup').waitFor({
    state: 'visible',
    timeout: opts.timeoutMs ?? 30_000,
  })
}
```

Discover `physicalUid` once at suite start via a poll endpoint or first WS message; if no public “last UID” API exists, add a tiny `GET /api/rfid/last-uid` **or** read from an on-page debug attribute. Prefer: call write after operator places tag; for automation, use:

```typescript
// One-time: POST inject is DISABLED in hardware mode.
// Discover UID by writing a known participant then reading association — 
// better: extend backend Poll path to expose GET /api/rfid/hardware-uid (Task 2b if needed).
```

If missing, **Task 2b** (same PR): add `GET /api/rfid/hardware-uid` (admin) that calls `Poll()` once and returns `{ tag_uid }`.

- [ ] **Step 3: Lap engine**

```typescript
// lapEngine.ts
import { sampleLapDelayMs } from './clock'

export type LapState = {
  nextDue: Map<string, number> // participantId -> epoch ms
  scored: number
}

export function initLapState(activeIds: string[], t0: number): LapState {
  const nextDue = new Map<string, number>()
  for (const id of activeIds) {
    nextDue.set(id, t0 + sampleLapDelayMs())
  }
  return { nextDue, scored: 0 }
}

export function dueRacers(state: LapState, now: number): string[] {
  return [...state.nextDue.entries()]
    .filter(([, due]) => due <= now)
    .sort((a, b) => a[1] - b[1])
    .map(([id]) => id)
}

export function scheduleNext(state: LapState, id: string, now: number) {
  state.nextDue.set(id, now + sampleLapDelayMs())
  state.scored += 1
}

export function removeRacer(state: LapState, id: string) {
  state.nextDue.delete(id)
}
```

**Throughput note:** One physical tag serializes laps. Realized mean gap per racer will exceed 1 minute when the active field is large; the engine still samples 30s–3m eligibility delays. Record a `idea`/`minor` issue if realized mean ≫ 90s so the agent can decide whether to shrink the active field.

- [ ] **Step 4: Commit**

```powershell
git add frontend/e2e/hardware-bluffet/lib
git commit -m "feat(e2e): add Bluffet hardware roster and lap engine"
```

---

### Task 7: Playwright hardware config + video

**Files:**
- Create: `frontend/e2e/hardware-bluffet/playwright.config.ts`
- Modify: `frontend/package.json`

- [ ] **Step 1: Config**

```typescript
import { defineConfig, devices } from '@playwright/test'
import path from 'node:path'

const artifactDir = process.env.BLUFFET_HW_ARTIFACT_DIR ?? path.join('..', '..', 'e2e-artifacts', 'bluffet-hardware', 'current')

export default defineConfig({
  testDir: './',
  fullyParallel: false,
  workers: 1,
  timeout: 45 * 60 * 1000, // 45 minutes wall clock
  expect: { timeout: 15_000 },
  reporter: [['list'], ['json', { outputFile: path.join(artifactDir, 'playwright-report.json') }]],
  use: {
    baseURL: process.env.E2E_BASE_URL ?? 'http://localhost:3000',
    trace: 'retain-on-failure',
    screenshot: 'on',
    video: {
      mode: 'on',
      size: { width: 1920, height: 1080 },
    },
  },
  projects: [
    { name: 'reader', use: { ...devices['Desktop Chrome'], viewport: { width: 1920, height: 1080 } } },
    { name: 'spectator-laptop', use: { ...devices['Desktop Chrome'], viewport: { width: 1920, height: 1080 } } },
    { name: 'spectator-iphone', use: { ...devices['iPhone 13'] } },
  ],
})
```

Playwright records per-test videos under `test-results/`. The orchestrator copies/renames to `reader.webm` etc. after the run (Task 10).

Because one test must drive **three contexts**, prefer **one project** (`reader`) that launches extra `browser.newContext()` for spectators inside the spec (simpler video naming). Update config to a **single project** if multi-project cannot share one timeline:

```typescript
projects: [{ name: 'hardware-bluffet', use: { ...devices['Desktop Chrome'] } }],
```

Inside the spec, create:

```typescript
const laptop = await browser.newContext({
  viewport: { width: 1920, height: 1080 },
  recordVideo: { dir: path.join(dir, 'raw'), size: { width: 1920, height: 1080 } },
})
const iphone = await browser.newContext({
  ...devices['iPhone 13'],
  recordVideo: { dir: path.join(dir, 'raw'), size: { width: 1920, height: 1080 } },
})
```

- [ ] **Step 2: npm script**

```json
"test:e2e:bluffet-hardware": "playwright test --config=e2e/hardware-bluffet/playwright.config.ts"
```

- [ ] **Step 3: Commit**

```powershell
git add frontend/e2e/hardware-bluffet/playwright.config.ts frontend/package.json
git commit -m "feat(e2e): add hardware Bluffet Playwright config with 1080p video"
```

---

### Task 8: Smoke Proxmark write→read on attached hardware

**Files:**
- Create: `frontend/e2e/hardware-bluffet/proxmark-smoke.spec.ts` (short)

- [ ] **Step 1: Smoke test (~2 min)**

```typescript
import { test, expect } from '@playwright/test'
import { BLUFFET, armFinishStation, pinToken } from '../fixtures/rfid'
import { reprogramAndAwaitLap } from './lib/proxmark'

test('proxmark write then real read scores a lap', async ({ page, request }) => {
  test.setTimeout(120_000)
  const token = await pinToken(request)
  await armFinishStation(request, token, BLUFFET.eventId)
  // discover UID via GET /api/rfid/hardware-uid (added in Task 2b)
  const uidRes = await request.get(`${process.env.E2E_API_URL ?? 'http://localhost:8080'}/api/rfid/hardware-uid`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  expect(uidRes.ok()).toBeTruthy()
  const { tag_uid } = await uidRes.json()
  await page.goto(`/events/${BLUFFET.eventId}/live`)
  // Ensure race is active or accept test-read pre-start
  await reprogramAndAwaitLap({
    request,
    readerPage: page,
    participantId: /* first non-noshow id from API */,
    physicalUid: tag_uid,
    eventId: BLUFFET.eventId,
  })
  await expect(page.getByTestId('scan-popup')).toBeVisible()
})
```

- [ ] **Step 2: Run against live stack + hardware; fix CLI command strings until green**

```powershell
cd frontend; npx playwright test --config=e2e/hardware-bluffet/playwright.config.ts proxmark-smoke.spec.ts
```

- [ ] **Step 3: Commit working CLI strings + smoke**

```powershell
git commit -am "test(e2e): Proxmark hardware smoke write/read"
```

---

### Task 9: Dress rehearsal orchestrator spec

**Files:**
- Create: `frontend/e2e/hardware-bluffet/dress-rehearsal.spec.ts`
- Create: `frontend/e2e/hardware-bluffet/lib/chaos.ts`
- Create: `frontend/e2e/hardware-bluffet/lib/spectators.ts`
- Create: `frontend/e2e/hardware-bluffet/lib/setup.ts`

- [ ] **Step 1: Setup helper — seed times, arm station**

```typescript
// setup.ts
import type { APIRequestContext } from '@playwright/test'
import { API_BASE, BLUFFET, armFinishStation, pinToken } from '../fixtures/rfid'

export async function setCompressedStartTimes(request: APIRequestContext, tZero: Date) {
  const token = await pinToken(request)
  const kidsStart = new Date(tZero.getTime() + 20 * 60_000)
  const updates = [
    [BLUFFET.races.twelveHour.id, tZero],
    [BLUFFET.races.sixHour.id, tZero],
    [BLUFFET.races.kids.id, kidsStart],
  ] as const
  for (const [id, start] of updates) {
    const res = await request.put(`${API_BASE}/api/races/${id}`, {
      headers: { Authorization: `Bearer ${token}` },
      data: { start_time: start.toISOString() },
    })
    if (!res.ok()) throw new Error(`update race ${id}: ${res.status()} ${await res.text()}`)
  }
}

export async function addLateSignup(
  request: APIRequestContext,
  raceId: string,
  categoryId: string,
  name: { first: string; last: string },
) {
  const token = await pinToken(request)
  const res = await request.post(`${API_BASE}/api/races/${raceId}/participants`, {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      first_name: name.first,
      last_name: name.last,
      category_id: categoryId,
    },
  })
  if (!res.ok()) throw new Error(`late signup: ${await res.text()}`)
  return res.json()
}
```

- [ ] **Step 2: Chaos helpers**

```typescript
// chaos.ts
import type { Browser, BrowserContext, Page } from '@playwright/test'

export async function startApiOutage(contexts: BrowserContext[]) {
  for (const ctx of contexts) {
    await ctx.setOffline(true)
  }
}

export async function endApiOutage(contexts: BrowserContext[]) {
  for (const ctx of contexts) {
    await ctx.setOffline(false)
  }
}

export async function crashReader(context: BrowserContext) {
  await context.close()
}

export async function reopenReader(browser: Browser, dir: string): Promise<{ context: BrowserContext; page: Page }> {
  const context = await browser.newContext({
    viewport: { width: 1920, height: 1080 },
    recordVideo: { dir, size: { width: 1920, height: 1080 } },
  })
  const page = await context.newPage()
  return { context, page }
}
```

**Outage semantics (local Docker, empty `HOSTED_API_URL`):** set spectator contexts (+ optional reader UI context) offline for 5 minutes. Backend continues Proxmark→Postgres. On restore, spectators load live view and must show laps scored during the window. Assert via API `request` (always online) vs UI counts.

If `HOSTED_API_URL` is set to a second service, also pause hosted push by blocking only hosted origin; prefer documenting this as optional stretch.

- [ ] **Step 3: Spectator churn**

```typescript
// spectators.ts
import type { Page } from '@playwright/test'
import { BLUFFET } from '../fixtures/rfid'

export async function spectatorLoop(page: Page, friends: string[], stopAt: number) {
  await page.goto(`/events/${BLUFFET.eventId}/live`)
  while (Date.now() < stopAt) {
    const q = friends[Math.floor(Math.random() * friends.length)]
    const search = page.getByTestId('racers-search').or(page.getByTestId('race-flow-legend-search'))
    if (await search.count()) {
      await search.first().fill(q)
    }
    // rotate tabs if present
    for (const tab of ['race-tab-12h', 'race-tab-6h', 'race-tab-90m']) {
      const t = page.getByTestId(tab)
      if (await t.count()) await t.click().catch(() => {})
    }
    await page.waitForTimeout(15_000 + Math.random() * 20_000)
  }
}
```

Update tab testids if compressed seed renames races — prefer stable testids already in `EventLive.vue` (`race-tab-12h` etc. keyed by duration bucket).

- [ ] **Step 4: Main timeline spec (skeleton)**

```typescript
import { test, expect, devices } from '@playwright/test'
import path from 'node:path'
import { BLUFFET, armFinishStation, pinToken, ORGANIZER_PIN } from '../fixtures/rfid'
import { createRunDir, writeStatus, appendIssue } from './lib/artifacts'
import { setCompressedStartTimes, addLateSignup } from './lib/setup'
import { loadSeededRacers, pickNoShows, pickDnfs } from './lib/roster'
import { initLapState, dueRacers, scheduleNext, removeRacer } from './lib/lapEngine'
import { reprogramAndAwaitLap } from './lib/proxmark'
import { startApiOutage, endApiOutage, crashReader, reopenReader } from './lib/chaos'
import { spectatorLoop } from './lib/spectators'
import { sleep } from './lib/clock'

test('East Bluffet hardware dress rehearsal', async ({ browser, request }) => {
  test.setTimeout(45 * 60_000)
  const { runId, dir } = createRunDir()
  const tZero = new Date(Date.now() + 2 * 60_000)
  await setCompressedStartTimes(request, tZero)

  const racers = await loadSeededRacers(request)
  const noShows = pickNoShows(racers, 9)
  const starters = racers.filter((r) => !noShows.has(r.id))
  // ... discover physicalUid, arm station, open contexts, status loop ...
  // T-30s: 2 late signups
  // T+0: lap engine while rotating reader pages (fullscreen-rotator bias)
  // T+2m: 1 late signup
  // schedule DNFs ~10
  // schedule crash ~T+8m, manual-entry catch-up
  // schedule API outage ~T+12m for 5m on spectator contexts
  // T+20 kids start (automatic via start_time)
  // end at T+30; finalize issues; expect issues.md empty for success gate optional
})
```

Fill remaining orchestration carefully in implementation — keep phases updating `writeStatus` at least every 10s so the agent’s 60s poll sees fresh data.

**Manual entry after crash:** use `POST /api/rfid/manual-entry` with bib + race + checkpoint for taps that were due while reader UI was down (backend may still have scored if Proxmark poll continued — if so, assert no loss and skip duplicate manual entry; if poll stops without browser, document as limitation and use manual entry for gaps).

- [ ] **Step 5: Commit orchestrator**

```powershell
git add frontend/e2e/hardware-bluffet
git commit -m "feat(e2e): East Bluffet hardware dress rehearsal orchestrator"
```

---

### Task 10: ffmpeg side-by-side compose

**Files:**
- Create: `scripts/compose-bluffet-hardware-video.ps1`

- [ ] **Step 1: Script**

```powershell
param(
  [Parameter(Mandatory = $true)][string]$RunDir
)
$reader = Join-Path $RunDir "reader.webm"
$laptop = Join-Path $RunDir "spectator-laptop.webm"
$iphone = Join-Path $RunDir "spectator-iphone.webm"
$out = Join-Path $RunDir "side-by-side-1440p.mp4"

# Scale each to 853x1440 (approx thirds of 2560x1440) then hstack
ffmpeg -y `
  -i $reader -i $laptop -i $iphone `
  -filter_complex "[0:v]scale=853:1440:force_original_aspect_ratio=decrease,pad=853:1440:(ow-iw)/2:(oh-ih)/2,setsar=1[v0];[1:v]scale=853:1440:force_original_aspect_ratio=decrease,pad=853:1440:(ow-iw)/2:(oh-ih)/2,setsar=1[v1];[2:v]scale=853:1440:force_original_aspect_ratio=decrease,pad=853:1440:(ow-iw)/2:(oh-ih)/2,setsar=1[v2];[v0][v1][v2]hstack=inputs=3[v]" `
  -map "[v]" -an -c:v libx264 -crf 20 -pix_fmt yuv420p $out

Write-Host "Wrote $out"
```

Harness post-step: rename Playwright videos into those three filenames before invoking the script.

- [ ] **Step 2: Commit**

```powershell
git add scripts/compose-bluffet-hardware-video.ps1
git commit -m "feat(scripts): compose three 1080p Bluffet videos into 1440p side-by-side"
```

---

### Task 11: Agent runbook (start + 1‑minute monitor)

**Files:**
- Create: `frontend/e2e/hardware-bluffet/README.md`
- Create: `.cursor/rules/bluffet-hardware-e2e.mdc` (optional short rule)

- [ ] **Step 1: README content (agent contract)**

```markdown
# East Bluffet hardware e2e

## Start (agent entrypoint)

When asked to **start the East Bluffet e2e test**:

1. Ensure Proxmark3 present; `pm3` works; tag on antenna.
2. `docker compose -f docker-compose.yml -f docker-compose.hardware.yml up --build -d`
   (or native backend with `RFID_HARDWARE=true` if USB passthrough fails)
3. Load seed:
   `Get-Content database/seed/03-bluffet-2026-hardware.sql | docker compose exec -T postgres psql -U timing_user -d keweenaw_timing`
4. From `frontend/`: `npm run test:e2e:bluffet-hardware`
5. Artifact dir: `e2e-artifacts/bluffet-hardware/<runId>/`

## Monitor every 60 seconds

Read `status.json` + `issues.md`:

- **critical** (Proxmark dead, data loss, sync never recovers, harness hung) → stop test, fix, commit, restart from step 2
- **minor/idea** → note; let run finish; after run fix all, commit, restart
- Success = full run + empty issues list + `side-by-side-1440p.mp4` present

## Chaos windows (owned by harness)

Late signups, DNFs, reader crash + manual entry, 5‑min client offline, kids race at T+20.
```

- [ ] **Step 2: Commit**

```powershell
git add frontend/e2e/hardware-bluffet/README.md .cursor/rules/bluffet-hardware-e2e.mdc
git commit -m "docs: agent runbook for East Bluffet hardware e2e monitor loop"
```

---

### Task 12: First full run + iterate

- [ ] **Step 1: Agent starts the test** per README  
- [ ] **Step 2: Wake every 1 minute**; append triage notes if harness missed something  
- [ ] **Step 3: On completion**, run ffmpeg script; review `issues.md`  
- [ ] **Step 4: Fix → commit → restart** until issues list is empty  

No code placeholder here — execution follows the design’s iterate policy.

---

## Self-review (plan vs spec)

| Spec requirement | Task |
|---|---|
| Real Proxmark write/read | 2, 8 |
| Single-tag rewrite per racer | 1, 6 |
| 100 seed, 9 no-shows, 3 late, ~10 DNF | 4, 6, 9 |
| start in 2 min; 30/15 + kids@T+20 | 4, 9 |
| Lap distribution 30s–3m ~1m mean | 5–6 |
| Reader carousel bias + page switches | 9 |
| Spectators laptop + iPhone 13 | 7, 9 |
| Reader crash + manual entry | 9 |
| 5‑min client API outage + catch-up | 9 |
| 3×1080p → 1440p | 7, 10 |
| Agent start + 1‑min monitor | 11–12 |
| Iterate until clean | 12 |

**Known product gap called out in plan:** `AssociateTag` reassignment (Task 1); mock-only reader (Task 2); empty `HOSTED_API_URL` outage = browser offline vs dual hosted (documented in Task 9).

---

## Execution handoff

Plan complete and saved to `docs/superpowers/plans/2026-07-15-hardware-bluffet-e2e.md`. Two execution options:

**1. Subagent-Driven (recommended)** — dispatch a fresh subagent per task, review between tasks, fast iteration  

**2. Inline Execution** — execute tasks in this session using executing-plans, batch execution with checkpoints  

Which approach?
