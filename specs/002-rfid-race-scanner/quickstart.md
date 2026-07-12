# Quickstart: RFID Race Scanner

**Feature**: `002-rfid-race-scanner` | **Date**: 2026-07-12  
**Status**: Implementation complete — use this guide for local demo, CI parity, and manual acceptance

## Prerequisites

- Docker Desktop / Docker Compose
- Proxmark3 optional (prefer mock: `RFID_INJECT=true` — already set in `docker-compose.yml`)

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

Expect: **All You Can East Bluffet** (`1441674d-a011-471a-a601-722b88b117f5`), 3 lap races (12h / 6h / 90-min kids), clarified categories, **100** racers with `DEMO-TAG-0001`…`0100`.

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
  -d '{"tag_uid":"DEMO-TAG-0001"}'
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

## Proxmark3 Docker USB device passthrough (optional hardware)

CI and local demos use **mock inject** (`RFID_INJECT=true` / `MockReader`). Real Proxmark3 USB is optional.

### Linux / WSL2 with USBIPD

1. Plug in the Proxmark3 and note the host serial device (often `/dev/ttyACM0` or `/dev/ttyUSB0`).
2. Pass the device into the backend container. Example override (create `docker-compose.proxmark.yml`):

```yaml
services:
  backend:
    devices:
      - /dev/ttyACM0:/dev/ttyACM0
    group_add:
      - dialout
    environment:
      - PROXMARK3_ENABLED=true
      - RFID_INJECT=false
      # Optional: set if your binary expects a port path
      - PROXMARK3_PORT=/dev/ttyACM0
```

```bash
docker compose -f docker-compose.yml -f docker-compose.proxmark.yml up --build backend
```

3. Confirm the container can open the TTY (`ls -l /dev/ttyACM0` inside the container) and that the host user/group has dialout access.

### Docker Desktop (Windows / macOS)

- **Windows**: Use USBIPD-WIN to attach the Proxmark3 into WSL2, then pass `/dev/ttyACM*` as above. Native Windows Docker Desktop does not reliably expose COM ports as Linux TTYs without WSL2 USBIPD.
- **macOS**: Docker Desktop generally cannot pass arbitrary USB serial devices into Linux containers; run the backend natively on the Mac with `PROXMARK3_ENABLED=true`, or use mock inject for containerized demos.

### Coverage exclusion (FR-029)

Proxmark3 USB I/O shims that talk to real hardware are covered by `MockReader` unit tests under `backend/internal/rfid`. Prefer mock + inject for CI; do not require a physical reader for the 100% gate.

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
