# Quickstart: RFID Race Scanner

**Feature**: `002-rfid-race-scanner` | **Date**: 2026-07-12  
**Status**: Implementation complete — use this guide for local demo, CI parity, and manual acceptance

## Prerequisites

- Docker Desktop / Docker Compose
- Proxmark3 optional (prefer mock: `RFID_INJECT=true` — already set in `docker-compose.yml`)

## Logical tag UUID identity

Scoring keys off a **logical UUID** written into tag user memory — **not** the chip’s silicon UID.

| Concept | Role |
|---------|------|
| Silicon UID | Hardware serial only — never used for racer identity or `tag_uid` lookups |
| Logical RFID UUID | Stable UUID per racer (`rfid_tag_associations.tag_uid`, `participants.rfid_tag_uid`). Generated at registration/seed and **never changes** for that racer |
| Program tag | Write the racer’s logical UUID onto whatever chip is on the reader (Racers UI or `POST /api/rfid/write-tag`) |
| Read / Poll | Read user-memory payload → return logical UUID on WebSocket / scan path |
| Lost / replacement tag | Place new blank chip → Program tag again → **same** logical UUID is written; no DB reassignment of silicon IDs |

## Approved UI prototypes

Reference (already approved): `frontend/prototypes/002-rfid-race-scanner/README.md`

## Boot local stack

```bash
docker compose up --build
```

- Frontend: http://localhost:3000  
- Backend: http://localhost:8080 / health: http://localhost:8080/health  

Compose defaults include `ORGANIZER_PIN=1738`, `RFID_INJECT=true`, and `PROXMARK3_ENABLED=true` (mock reader). Optional: set `HOSTED_API_URL` for hosted sync.

## Load AYCEB 2026 demo seed

```bash
python database/seed/generate_bluffet_seed.py
docker compose exec -T postgres psql -U timing_user -d keweenaw_timing < database/seed/03-bluffet-2026.sql
```

On PowerShell, pipe via stdin:

```powershell
python database/seed/generate_bluffet_seed.py
Get-Content database/seed/03-bluffet-2026.sql | docker compose exec -T postgres psql -U timing_user -d keweenaw_timing
```

Expect: **All You Can East Bluffet** (`1441674d-a011-471a-a601-722b88b117f5`), 3 lap races (12h / 6h / 90-min kids), clarified categories, **100** racers with deterministic logical RFID UUIDs (`uuid5` per `tag:{race}:{n}` — see `frontend/e2e/fixtures/rfid.ts`).

The generator uses **deterministic UUIDs** (fixed event/race IDs matching `frontend/e2e/fixtures/rfid.ts` `BLUFFET`; child rows via uuid5). Regenerating SQL does not break e2e fixtures.

## Organizer PIN

Default: **`1738`**

```bash
curl -s -X POST http://localhost:8080/api/auth/pin \
  -H "Content-Type: application/json" \
  -d '{"pin":"1738"}'
```

## Arm finish reader

1. PIN → Station config → Bluffet event → mode **Finish**
2. Keep SPA open (WebSocket reader on all routes)
3. Live view: tabs, overall boards, countdown pre-start; auto-start at race `start_time`

## Simulate a tap (mock RFID)

```bash
curl -s -X POST http://localhost:8080/api/rfid/inject \
  -H "Content-Type: application/json" \
  -d '{"tag_uid":"23657b2d-aa08-5fe8-8553-e9e3affb4678"}'
```

Expect: popup (name, overall place, laps), sound without on-screen sound label; karaoke button → “recorded” after one click; test-read if race still scheduled.

## Live CSV

After any lap/racer change, station file updates automatically (path per deploy, e.g. under `data/events/{id}/live-snapshot.csv`). Copy that file for USB backup; import on a replacement laptop — **no manual export step**.

```bash
# Optional copy of live snapshot (requires PIN JWT)
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/events/$EVENT_ID/live-csv" -o bluffet-live.csv

curl -H "Authorization: Bearer $TOKEN" \
  -F file=@bluffet-live.csv \
  "http://localhost:8080/api/events/$EVENT_ID/import.csv"
```

## Offline / sync

Local Postgres keeps accepting taps. IndexedDB is WAQ only. When `HOSTED_API_URL` is set and network returns, push/pull pending.

## Proxmark3 hardware (optional)

CI and local demos use **mock inject** (`RFID_INJECT=true` / `MockReader`). Real Proxmark3 USB is optional and **never required for CI**.

### Enable hardware reader

| Variable | Purpose |
|----------|---------|
| `RFID_HARDWARE=true` | Select `CLIProxmarkReader` (pm3 CLI) instead of `MockReader` |
| `RFID_INJECT=false` | Disable mock inject endpoint (set automatically in hardware overlay) |
| `PROXMARK3_ENABLED=true` | Reader subsystem on (mock or hardware depending on flags above) |
| `PROXMARK3_CLI` | pm3 binary name/path (default `pm3`) |
| `PROXMARK3_PORT` | Serial port passed to pm3 `-p` when auto-detect fails (e.g. `/dev/ttyACM0`) |

Compose overlay (disables inject, enables hardware CLI):

```bash
docker compose -f docker-compose.yml -f docker-compose.hardware.yml up --build
```

Override CLI/port from the host shell:

```bash
PROXMARK3_CLI=pm3 PROXMARK3_PORT=/dev/ttyACM0 \
  docker compose -f docker-compose.yml -f docker-compose.hardware.yml up --build backend
```

On **Windows without WSL2 USBIPD**, run the backend **natively** on the host with the same env vars instead of the container overlay.

### USB passthrough (Linux / WSL2)

1. Plug in Proxmark3; note serial device (`/dev/ttyACM0` or `/dev/ttyUSB0`).
2. **Windows + WSL2**: use [USBIPD-WIN](https://github.com/dorssel/usbipd-win) — `usbipd list`, `usbipd bind --busid <BUSID>`, `usbipd attach --wsl --busid <BUSID>`.
3. Uncomment `devices` / `group_add` in `docker-compose.hardware.yml`, set `PROXMARK3_PORT`, and confirm TTY inside the container (`ls -l /dev/ttyACM0`).
4. **macOS**: Docker Desktop cannot pass arbitrary USB serial into Linux containers; run backend natively with `RFID_HARDWARE=true` or use mock inject.

### Program tag (hardware)

1. Place tag on reader.
2. Organizer PIN → Racers → **Program tag** for a seeded racer (writes that racer’s logical UUID to chip user memory), or:

```bash
curl -s -X POST http://localhost:8080/api/rfid/write-tag \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"participant_id":"<seeded-participant-uuid>"}'
```

3. Poll / live WebSocket should deliver the racer’s **logical UUID** (not silicon UID).
4. Expect lap or test-read popup when the race is armed.

Replacement tag: program the **same** racer again — logical UUID is rewritten onto the new chip; DB identity unchanged.

### Hardware smoke test (skip-by-default)

```bash
cd backend
RFID_HARDWARE=true go test ./internal/rfid -run TestHardwareProxmark3Smoke -v
```

Skipped when `RFID_HARDWARE` is unset/false. Optional: `PROXMARK3_SMOKE_UUID` for the round-trip UUID.

### Coverage exclusion (FR-029)

Proxmark3 USB I/O is covered by `MockReader` unit tests under `backend/internal/rfid`. Prefer mock + inject for CI; do not require a physical reader for the 100% gate.

## Tests (TDD / CI parity)

```bash
# Backend (preferred — matches CI)
docker compose -f docker-compose.test.yml run --rm backend-test

# Scan package 100% coverage fail-under (FR-029)
docker compose -f docker-compose.test.yml run --rm --user root \
  -v "${PWD}/scripts:/scripts:ro" \
  -e COVERAGE_FAIL_UNDER=100.0 \
  backend-test sh /scripts/coverage-fail-under.sh

# Frontend unit (Vitest; excludes Playwright e2e/)
docker compose -f docker-compose.test.yml run --rm frontend-test
# or: cd frontend && npm test -- --run

# Playwright e2e (stack up + seed + mock inject)
docker compose up --build -d
# load seed as above, then:
cd frontend
npm ci
npx playwright install --with-deps chromium
npm run test:e2e
```

Coverage fail-under is **100%** for new packages/modules in this feature (`backend/internal/services/scan/...` and frontend thresholds in `vitest.config.ts`). Hardware-only Proxmark3 USB paths are excluded via MockReader coverage (see above).

## Manual acceptance checklist (not automated)

Do **not** treat these as CI gates. Record pass/fail in PR notes when accepting a release:

| Criterion | What to verify manually |
|-----------|-------------------------|
| **SC-001** | Known tag outside cooldown → stored lap + popup (name, place, laps) + sound within **≤2s** |
| **SC-003** | Karaoke bonus records within **≤3s** of one click; second click does not duplicate |
| **SC-012** | Pre-start countdown ticks at **1s** resolution and auto-starts at `start_time` |
| **SC-013** | Racers search filters within **~300ms** after typing pause (no Search button) |
| **SC-005** | **4-hour offline soak** (or shortened soak with documented waiver): taps accepted offline, sync when network returns |

## Next

Feature polish tasks live in `tasks.md` Phase 12. Day-of ops: boot stack → seed → PIN → arm Finish → inject or Proxmark3.
