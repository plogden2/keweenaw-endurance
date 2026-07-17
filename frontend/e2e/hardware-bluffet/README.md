# East Bluffet hardware e2e — agent runbook

Wall-clock dress rehearsal of All You Can East Bluffet using a real Proxmark3, three Playwright contexts (reader, spectator laptop, spectator iPhone 13), and chaos scenarios (reader crash, 5‑minute **hosted partition** with offline bridge scoring + automatic sync).

**Prerequisite:** Complete the Proxmark logical UUID plan first — `docs/superpowers/plans/2026-07-15-proxmark3-tag-uuid.md` (design: `docs/superpowers/specs/2026-07-15-proxmark3-tag-uuid-design.md`). The harness writes each racer's permanent logical UUID onto a single physical chip before every lap; silicon UID is never the scoring key.

**Design:** `docs/superpowers/specs/2026-07-15-hardware-bluffet-e2e-design.md`

## Control model

The **harness owns the timeline** — lap sequencing, Proxmark write→read, chaos windows, assertions, screenshots, and video recording. The **agent starts the test and monitors only**; do not invent mid-race taps or change the schedule ad hoc.

## Prod-like bring-up (dress rehearsal topology)

The rehearsal mimics production: **hosted stack** (UI + API + Postgres) plus a **native device-bridge** on the reader laptop that owns USB Proxmark. Normal outages never use `import.csv` — the bridge queues laps locally and **auto-flushes** on reconnect. The reader UI shows **Offline → Syncing → Online · Synced**.

### 1. Hosts entry

Add to `C:\Windows\System32\drivers\etc\hosts`:

```text
127.0.0.1 keweenawendurance.com
```

### 2. Hosted stack

From the repo root:

```powershell
docker compose -f docker-compose.prod.yml up --build -d
```

### 3. Seed

Load the compressed-duration Bluffet hardware seed:

```powershell
# e.g. psql or your usual seed loader
database/seed/03-bluffet-2026-hardware.sql
```

### 4. Device-bridge (native on reader laptop)

The bridge **must run outside Docker** for COM port access. See `backend/cmd/device-bridge/README.md`.

```powershell
$env:HOSTED_API_URL="http://keweenawendurance.com"
$env:ORGANIZER_PIN="1738"
$env:DEVICE_ID="laptop-finish-1"
$env:EVENT_ID="1441674d-a011-471a-a601-722b88b117f5"
$env:BRIDGE_DATA_DIR="$PWD\bridge-data"
# Optional — defaults to %TEMP%\keweenaw-bridge-partition.signal (same path the harness uses)
# $env:BRIDGE_PARTITION_SIGNAL="$PWD\bridge-data\partition.signal"
$env:RFID_HARDWARE="true"
$env:PROXMARK3_PORT="COM3"
cd backend
go run ./cmd/device-bridge
```

Loopback HTTP (`http://127.0.0.1:8091`) serves `/status` and `/write-tag` while hosted is unreachable. Offline laps land in `bridge-data/events/{EVENT_ID}/pending.jsonl` and `live-snapshot.csv`.

### 5. Run harness

From `frontend/`:

```powershell
npm run test:e2e:bluffet-hardware
```

For the prod-like URL explicitly:

```powershell
node ../scripts/run-bluffet-hardware.mjs --prod http://keweenawendurance.com
```

This npm script invokes `scripts/run-bluffet-hardware.mjs`, which **creates the run directory** under `e2e-artifacts/bluffet-hardware/<runId>/`, seeds empty `issues.jsonl` / `issues.md`, exports `BLUFFET_HW_ARTIFACT_DIR`, then spawns Playwright. All artifacts for one run land in that single directory.

### 6. Monitor every 60s

Read `e2e-artifacts/bluffet-hardware/<runId>/status.json` and `issues.md`. Summarize phase, health, laps scored, pending sync, chaos flags, and any new issues.

### 7. Iterate policy

- **Critical** → stop the harness early, fix, commit, restart from step 1.
- **Minor / ideas only** → let the run finish, fix all listed items, commit, restart until a run ends with an empty issues list.

Exit when a full run completes with **zero** remaining bugs, limitations, or improvement ideas.

## Outage chaos (T+14:00 → T+19:00)

At T+14 the harness:

1. Creates `BRIDGE_PARTITION_SIGNAL` (cuts bridge→hosted WebSocket; loopback HTTP + poll continue).
2. Takes **spectator** browser contexts offline (stale UI).
3. Keeps the **reader** context online so the sync chip can show **Offline** via local bridge status.
4. Routes lap writes to `http://127.0.0.1:8091/write-tag` and asserts hosted lap totals **do not** increase.
5. Asserts local bridge `pending_count` grows with offline laps scored.

On heal (T+19:00):

- Removes the partition signal — **no** `import.csv`.
- Asserts reader chip **Syncing** then **Online · Synced**.
- Asserts hosted totals catch up to pre-outage + offline laps scored.
- Spectators `awaitCatchUp`.

## Legacy dev stack (non-prod)

For quick local dev without prod-like nginx/domain:

```powershell
docker compose -f docker-compose.yml -f docker-compose.hardware.yml up --build -d
# seed hardware SQL
# start device-bridge with HOSTED_API_URL=http://localhost:8080
npm run test:e2e:bluffet-hardware
```

## Artifact layout

Per run: `e2e-artifacts/bluffet-hardware/<runId>/`

| File | Purpose |
|---|---|
| `status.json` | Live phase, wall clocks, laps scored, pending sync, chaos flags, last Proxmark op — polled every minute |
| `issues.jsonl` | Machine-readable issue stream (timestamped, with severity) |
| `issues.md` | Human-readable rendered issue list for the agent |
| `reader.webm` | Reader context recording (1920×1080) |
| `spectator-laptop.webm` | Desktop spectator recording (1920×1080) |
| `spectator-iphone.webm` | iPhone 13 profile spectator recording (1920×1080) |
| `side-by-side-1440p.mp4` | Composed deliverable (see below) |
| `playwright-report.json` | Playwright JSON report for the run |

Screenshots referenced from issue entries are saved alongside these files when failures occur.

### Side-by-side video compose

After a run finishes (or when reviewing artifacts), compose the three 1080p source videos into one 1440p side-by-side MP4 with Reader | Laptop | iPhone labels:

```powershell
.\scripts\compose-bluffet-hardware-video.ps1 -RunDir "e2e-artifacts\bluffet-hardware\<runId>"
```

Requires `ffmpeg` on PATH. Inputs: `reader.webm`, `spectator-laptop.webm`, `spectator-iphone.webm`. Output: `side-by-side-1440p.mp4` in the same run directory.

## What the agent does not do

- Drive individual RFID taps or lap timing (harness lap engine handles this).
- Skip prerequisite Proxmark / logical UUID work.
- Call `import.csv` during normal outage recovery (emergency-only operator path).
- Ignore critical issues and wait for a "clean" status at the end.
