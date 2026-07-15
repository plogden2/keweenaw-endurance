# East Bluffet hardware e2e — agent runbook

Wall-clock dress rehearsal of All You Can East Bluffet using a real Proxmark3, three Playwright contexts (reader, spectator laptop, spectator iPhone 13), and chaos scenarios (reader crash, 5‑minute API outage).

**Prerequisite:** Complete the Proxmark logical UUID plan first — `docs/superpowers/plans/2026-07-15-proxmark3-tag-uuid.md` (design: `docs/superpowers/specs/2026-07-15-proxmark3-tag-uuid-design.md`). The harness writes each racer's permanent logical UUID onto a single physical chip before every lap; silicon UID is never the scoring key.

**Design:** `docs/superpowers/specs/2026-07-15-hardware-bluffet-e2e-design.md`

## Control model

The **harness owns the timeline** — lap sequencing, Proxmark write→read, chaos windows, assertions, screenshots, and video recording. The **agent starts the test and monitors only**; do not invent mid-race taps or change the schedule ad hoc.

## Start the East Bluffet e2e test

When asked to **start the East Bluffet e2e test**:

1. **Proxmark ready** — Confirm the Proxmark plan smoke is green; place the tag on the antenna; verify `pm3` works on this machine.
2. **Stack up** — From the repo root:
   ```powershell
   docker compose -f docker-compose.yml -f docker-compose.hardware.yml up --build -d
   ```
3. **Seed** — Load the compressed-duration Bluffet seed:
   ```powershell
   # e.g. psql or your usual seed loader
   database/seed/03-bluffet-2026-hardware.sql
   ```
4. **Run harness** — From `frontend/`:
   ```powershell
   npm run test:e2e:bluffet-hardware
   ```
   This npm script invokes `scripts/run-bluffet-hardware.mjs`, which **creates the run directory** under `e2e-artifacts/bluffet-hardware/<runId>/`, seeds empty `issues.jsonl` / `issues.md`, exports `BLUFFET_HW_ARTIFACT_DIR`, then spawns Playwright. All artifacts for one run land in that single directory.
5. **Monitor every 60s** — Read `e2e-artifacts/bluffet-hardware/<runId>/status.json` and `issues.md`. Summarize phase, health, laps scored, chaos flags, and any new issues.
6. **Iterate policy**
   - **Critical** → stop the harness early, fix, commit, restart from step 1.
   - **Minor / ideas only** → let the run finish, fix all listed items, commit, restart until a run ends with an empty issues list.

Exit when a full run completes with **zero** remaining bugs, limitations, or improvement ideas.

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
- Ignore critical issues and wait for a "clean" status at the end.
