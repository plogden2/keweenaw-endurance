# Hardware East Bluffet e2e Dress Rehearsal — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver a Proxmark3-backed, wall-clock East Bluffet dress-rehearsal harness that an agent can start and monitor every minute, including chaos (reader crash, 5‑min API outage), 1080p×3 video → 1440p side-by-side, and an iterate-until-clean issues loop.

**Architecture:** **Prerequisite:** complete `docs/superpowers/plans/2026-07-15-proxmark3-tag-uuid.md` first (real Proxmark driver + logical UUID on tag). This plan adds a compressed-duration Bluffet seed, then a Playwright harness under `frontend/e2e/hardware-bluffet/` that owns the timeline and writes `status.json` / `issues.md` for the monitoring agent. With one physical chip, each lap is `WriteTag(participant)` (overwrite user memory with that racer’s permanent logical UUID) → real `Poll`/WS read → score. Spectators use desktop + iPhone 13 contexts; API outage uses Playwright offline while the backend keeps polling Proxmark into local Postgres.

**Tech Stack:** Vue/Playwright, Python seed generator, ffmpeg, Docker Compose, Cursor agent runbook (Proxmark/Go identity work lives in the prerequisite plan)

**Spec:** `docs/superpowers/specs/2026-07-15-hardware-bluffet-e2e-design.md`  
**Prerequisite plan:** `docs/superpowers/plans/2026-07-15-proxmark3-tag-uuid.md`  
**Prerequisite design:** `docs/superpowers/specs/2026-07-15-proxmark3-tag-uuid-design.md`

---

## File map

| File | Responsibility |
|---|---|
| `database/seed/generate_bluffet_seed.py` | `--durations=30,15,5` + race display names (logical UUIDs from Proxmark plan) |
| `database/seed/03-bluffet-2026-hardware.sql` | Compressed-duration seed artifact |
| `docker-compose.hardware.yml` | Already from Proxmark plan — reuse |
| `frontend/e2e/hardware-bluffet/playwright.config.ts` | Video 1080p, long timeout |
| `frontend/e2e/hardware-bluffet/dress-rehearsal.spec.ts` | Single orchestrated ~32 min test |
| `frontend/e2e/hardware-bluffet/lib/*.ts` | status, issues, clock, lap-engine, roster, chaos, spectators |
| `frontend/e2e/hardware-bluffet/README.md` | Agent entrypoint + 1‑min monitor loop |
| `scripts/compose-bluffet-hardware-video.ps1` | ffmpeg 3×1080p → 1440p side-by-side |
| `frontend/package.json` | `test:e2e:bluffet-hardware` script |
| `.gitignore` | `e2e-artifacts/` |

**Not in this plan:** Proxmark CLI driver, Reader interface, logical UUID codec, WriteTag API/UI, silicon-UID reassignment (deleted as unnecessary).

---

### Task 0: Verify prerequisite

- [ ] **Step 1: Confirm Proxmark plan is merged/green**

```powershell
# RFID_HARDWARE smoke: write logical UUID for one seeded racer, Poll returns same UUID, scan popup works
cd backend; go test ./internal/rfid/ ./internal/services/ -count=1
```

Expected: all pass; hardware smoke from Proxmark Task 7 documented as OK on this laptop.

- [ ] **Step 2: If not done, stop and implement `2026-07-15-proxmark3-tag-uuid.md` first**

---

### Task 1: Compressed Bluffet seed durations

**Files:**
- Modify: `database/seed/generate_bluffet_seed.py`
- Create: `database/seed/03-bluffet-2026-hardware.sql`

- [ ] **Step 1: Add CLI flag** (keep default 720/360/90 for CI)

```python
import argparse
parser = argparse.ArgumentParser()
parser.add_argument("--durations", default="720,360,90")
parser.add_argument("--names", default="12 Hour,6 Hour,90-Minute Kids")
parser.add_argument("--output", default=str(OUTPUT_SQL))
args = parser.parse_args()
durs = [int(x) for x in args.durations.split(",")]
names = [x.strip() for x in args.names.split(",")]
assert len(durs) == 3 and len(names) == 3
# apply durs[i], names[i] into races list; keep stable event/race UUIDs
```

Logical tag UUIDs already come from the Proxmark seed work (`stable_uuid(f"tag:...")`).

- [ ] **Step 2: Generate hardware SQL**

```powershell
python database/seed/generate_bluffet_seed.py --durations=30,15,5 --names="30 Minute,15 Minute,5-Minute Kids" --output=database/seed/03-bluffet-2026-hardware.sql
```

- [ ] **Step 3: Commit**

```powershell
git add database/seed/generate_bluffet_seed.py database/seed/03-bluffet-2026-hardware.sql
git commit -m "feat(seed): add compressed-duration Bluffet seed for hardware e2e"
```

---

### Task 2: Artifact helpers (status + issues)

**Files:**
- Create: `frontend/e2e/hardware-bluffet/lib/artifacts.ts`
- Create: `frontend/e2e/hardware-bluffet/lib/clock.ts`
- Modify: `.gitignore` — add `e2e-artifacts/`

- [ ] **Step 1: Implement**

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
  const u = rng()
  const ms = 30_000 + u * u * 150_000
  return Math.min(180_000, Math.max(30_000, Math.round(ms)))
}

export function sleep(ms: number) {
  return new Promise((r) => setTimeout(r, ms))
}
```

- [ ] **Step 2: Commit**

```powershell
git add frontend/e2e/hardware-bluffet/lib .gitignore
git commit -m "feat(e2e): add hardware Bluffet status and issues artifact helpers"
```

---

### Task 3: Roster + lap engine (logical UUID rewrite)

**Files:**
- Create: `frontend/e2e/hardware-bluffet/lib/roster.ts`
- Create: `frontend/e2e/hardware-bluffet/lib/lapEngine.ts`
- Create: `frontend/e2e/hardware-bluffet/lib/proxmark.ts`

- [ ] **Step 1: Roster**

```typescript
import type { APIRequestContext } from '@playwright/test'
import { API_BASE, BLUFFET, pinToken } from '../../fixtures/rfid'

export type Racer = {
  id: string
  raceId: string
  bib: string
  firstName: string
  lastName: string
  logicalTagUuid: string // permanent racer RFID UUID (from seed/association)
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
        logicalTagUuid: p.rfid_tag_uid ?? p.rfidTagUid ?? (p.tag_uids?.[0] ?? ''),
      })
    }
  }
  return racers
}

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

- [ ] **Step 2: Proxmark helper — write logical UUID, await real read**

```typescript
import type { APIRequestContext, Page } from '@playwright/test'
import { API_BASE, pinToken } from '../../fixtures/rfid'

/**
 * Single physical chip dress rehearsal:
 * WriteTag(participant) overwrites chip user memory with that racer's permanent logical UUID,
 * then a real Proxmark Poll/WS read scores the lap. No silicon UID reassignment.
 */
export async function programRacerAndAwaitLap(opts: {
  request: APIRequestContext
  readerPage: Page
  participantId: string
  timeoutMs?: number
}) {
  const token = await pinToken(opts.request)
  const write = await opts.request.post(`${API_BASE}/api/rfid/write-tag`, {
    headers: { Authorization: `Bearer ${token}` },
    data: { participant_id: opts.participantId },
  })
  if (!write.ok()) {
    throw new Error(`write-tag failed: ${write.status()} ${await write.text()}`)
  }
  await opts.readerPage.getByTestId('scan-popup').waitFor({
    state: 'visible',
    timeout: opts.timeoutMs ?? 30_000,
  })
}
```

- [ ] **Step 3: Lap engine**

```typescript
import { sampleLapDelayMs } from './clock'

export type LapState = {
  nextDue: Map<string, number>
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

**Throughput note:** One chip serializes laps. Eligibility uses 30s–3m (mean ~1m); realized gaps may be longer — log a `minor`/`idea` if mean ≫ 90s.

- [ ] **Step 4: Commit**

```powershell
git add frontend/e2e/hardware-bluffet/lib
git commit -m "feat(e2e): Bluffet hardware roster and logical-UUID lap engine"
```

---

### Task 4: Playwright hardware config + video

**Files:**
- Create: `frontend/e2e/hardware-bluffet/playwright.config.ts`
- Modify: `frontend/package.json`

- [ ] **Step 1: Single-project config; spectators via extra contexts**

```typescript
import { defineConfig, devices } from '@playwright/test'
import path from 'node:path'

const artifactDir =
  process.env.BLUFFET_HW_ARTIFACT_DIR ??
  path.join('..', '..', 'e2e-artifacts', 'bluffet-hardware', 'current')

export default defineConfig({
  testDir: './',
  fullyParallel: false,
  workers: 1,
  timeout: 45 * 60 * 1000,
  expect: { timeout: 15_000 },
  reporter: [['list'], ['json', { outputFile: path.join(artifactDir, 'playwright-report.json') }]],
  use: {
    baseURL: process.env.E2E_BASE_URL ?? 'http://localhost:3000',
    trace: 'retain-on-failure',
    screenshot: 'on',
    video: { mode: 'on', size: { width: 1920, height: 1080 } },
  },
  projects: [{ name: 'hardware-bluffet', use: { ...devices['Desktop Chrome'] } }],
})
```

Inside the dress-rehearsal spec, create laptop + iPhone contexts with `recordVideo: { dir, size: { width: 1920, height: 1080 } }` (iPhone project device descriptor for UA/viewport).

- [ ] **Step 2: npm script**

```json
"test:e2e:bluffet-hardware": "playwright test --config=e2e/hardware-bluffet/playwright.config.ts"
```

- [ ] **Step 3: Commit**

```powershell
git add frontend/e2e/hardware-bluffet/playwright.config.ts frontend/package.json
git commit -m "feat(e2e): hardware Bluffet Playwright config with 1080p video"
```

---

### Task 5: Dress rehearsal orchestrator

**Files:**
- Create: `frontend/e2e/hardware-bluffet/dress-rehearsal.spec.ts`
- Create: `frontend/e2e/hardware-bluffet/lib/chaos.ts`
- Create: `frontend/e2e/hardware-bluffet/lib/spectators.ts`
- Create: `frontend/e2e/hardware-bluffet/lib/setup.ts`

- [ ] **Step 1: Setup — start times + late signup**

```typescript
import type { APIRequestContext } from '@playwright/test'
import { API_BASE, BLUFFET, pinToken } from '../../fixtures/rfid'

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
    data: { first_name: name.first, last_name: name.last, category_id: categoryId },
  })
  if (!res.ok()) throw new Error(`late signup: ${await res.text()}`)
  return res.json()
}
```

Late signup creates a participant; first lap calls `programRacerAndAwaitLap` which `ensureLogicalTagUUID` + writes that UUID to the chip.

- [ ] **Step 2: Chaos + spectators** (same as prior design)

```typescript
// chaos.ts
export async function startApiOutage(contexts: import('@playwright/test').BrowserContext[]) {
  for (const ctx of contexts) await ctx.setOffline(true)
}
export async function endApiOutage(contexts: import('@playwright/test').BrowserContext[]) {
  for (const ctx of contexts) await ctx.setOffline(false)
}
```

```typescript
// spectators.ts — search/track 5 friends, rotate race tabs, dwell on live/leaderboards
```

**Outage (empty HOSTED_API_URL):** offline spectator (+ optional reader UI) contexts for 5 minutes; backend continues Proxmark→Postgres; on restore assert UI shows outage-window laps (compare to always-online `request` API counts).

- [ ] **Step 3: Main timeline**

Phases: T−2m setup → 2 late signups → T+0 lap engine (`programRacerAndAwaitLap` per due racer) + reader page rotation (bias `fullscreen-rotator`) → T+2m late signup → ~10 DNFs → reader crash + manual entry → 5‑min API outage → T+20 kids → T+30 finalize.

Update `writeStatus` at least every 10s.

**Pre-race:** optionally walk roster calling `write-tag` once each to prove programming (overwrites same chip); associations already hold permanent UUIDs from seed.

- [ ] **Step 4: Commit**

```powershell
git add frontend/e2e/hardware-bluffet
git commit -m "feat(e2e): East Bluffet hardware dress rehearsal orchestrator"
```

---

### Task 6: ffmpeg side-by-side compose

**Files:**
- Create: `scripts/compose-bluffet-hardware-video.ps1`

- [ ] **Step 1: Script**

```powershell
param([Parameter(Mandatory = $true)][string]$RunDir)
$reader = Join-Path $RunDir "reader.webm"
$laptop = Join-Path $RunDir "spectator-laptop.webm"
$iphone = Join-Path $RunDir "spectator-iphone.webm"
$out = Join-Path $RunDir "side-by-side-1440p.mp4"

ffmpeg -y `
  -i $reader -i $laptop -i $iphone `
  -filter_complex "[0:v]scale=853:1440:force_original_aspect_ratio=decrease,pad=853:1440:(ow-iw)/2:(oh-ih)/2,setsar=1[v0];[1:v]scale=853:1440:force_original_aspect_ratio=decrease,pad=853:1440:(ow-iw)/2:(oh-ih)/2,setsar=1[v1];[2:v]scale=853:1440:force_original_aspect_ratio=decrease,pad=853:1440:(ow-iw)/2:(oh-ih)/2,setsar=1[v2];[v0][v1][v2]hstack=inputs=3[v]" `
  -map "[v]" -an -c:v libx264 -crf 20 -pix_fmt yuv420p $out
```

- [ ] **Step 2: Commit**

```powershell
git add scripts/compose-bluffet-hardware-video.ps1
git commit -m "feat(scripts): compose three 1080p Bluffet videos into 1440p side-by-side"
```

---

### Task 7: Agent runbook

**Files:**
- Create: `frontend/e2e/hardware-bluffet/README.md`

- [ ] **Step 1: Document entrypoint**

When asked to **start the East Bluffet e2e test**:

1. Confirm Proxmark plan smoke is green; tag on antenna; `pm3` works  
2. `docker compose -f docker-compose.yml -f docker-compose.hardware.yml up --build -d`  
3. Load `database/seed/03-bluffet-2026-hardware.sql`  
4. `cd frontend; npm run test:e2e:bluffet-hardware`  
5. Every 60s read `e2e-artifacts/bluffet-hardware/<runId>/status.json` + `issues.md`  
6. Critical → stop, fix, commit, restart; minor → finish, fix all, commit, restart until issues empty  

- [ ] **Step 2: Commit**

```powershell
git add frontend/e2e/hardware-bluffet/README.md
git commit -m "docs: agent runbook for East Bluffet hardware e2e"
```

---

### Task 8: First full run + iterate

- [ ] **Step 1:** Agent starts the test per README  
- [ ] **Step 2:** Monitor every 1 minute  
- [ ] **Step 3:** Compose video; review issues  
- [ ] **Step 4:** Fix → commit → restart until clean  

---

## Self-review (plan vs spec)

| Spec requirement | Task |
|---|---|
| Real Proxmark write/read | **Prerequisite plan** |
| Logical UUID on tag (not silicon) | **Prerequisite plan** |
| Single-chip rewrite per racer lap | 3, 5 (`WriteTag(participant)` only) |
| 100 seed, 9 no-shows, 3 late, ~10 DNF | 1, 3, 5 |
| start in 2 min; 30/15 + kids@T+20 | 1, 5 |
| Lap distribution 30s–3m ~1m mean | 2–3 |
| Reader carousel + page switches | 5 |
| Spectators laptop + iPhone 13 | 4–5 |
| Reader crash + manual entry | 5 |
| 5‑min client API outage + catch-up | 5 |
| 3×1080p → 1440p | 4, 6 |
| Agent start + 1‑min monitor | 7–8 |

---

## Execution order

1. **First:** `docs/superpowers/plans/2026-07-15-proxmark3-tag-uuid.md`  
2. **Then:** this plan  

Two execution options for the **Proxmark** plan (do that now):

**1. Subagent-Driven (recommended)** — fresh subagent per task  

**2. Inline Execution** — execute in this session with checkpoints  

Which approach?
