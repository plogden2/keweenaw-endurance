# Quickstart: RFID Race Scanner

**Feature**: `002-rfid-race-scanner` | **Date**: 2026-07-12  
**Status**: Spec ready — start with `/speckit-implement` (e2e-first per `tasks.md`)

## Prerequisites

- Docker Desktop / Docker Compose
- Proxmark3 optional (use mock: `PROXMARK3_ENABLED=true` / `RFID_INJECT=true` / `GO_ENV=test`)

## Approved UI prototypes

Reference (already approved): `frontend/prototypes/002-rfid-race-scanner/README.md`

## Boot local stack

```bash
docker compose up --build
```

- Frontend: http://localhost:3000  
- Backend: http://localhost:8080  

Optional: set `ORGANIZER_PIN=1738`, `HOSTED_API_URL=…`, `RFID_INJECT=true`.

## Load AYCEB 2026 demo seed

```bash
python database/seed/generate_bluffet_seed.py
docker compose exec -T postgres psql -U timing_user -d keweenaw_timing < database/seed/03-bluffet-2026.sql
```

Expect: **All You Can East Bluffet**, 3 races, clarified categories, **100** racers with category assignment + demo tags.

## Organizer PIN

Default: **`1738`**

```bash
curl -s -X POST http://localhost:8080/api/auth/pin \
  -H "Content-Type: application/json" \
  -d "{\"pin\":\"1738\"}"
```

## Arm finish reader

1. PIN → Station config → Bluffet event → mode **Finish**
2. Keep SPA open (WebSocket reader on all routes)
3. Live view: tabs, overall boards, countdown pre-start; auto-start at race `start_time`

## Simulate a tap

```bash
curl -s -X POST http://localhost:8080/api/rfid/inject \
  -H "Content-Type: application/json" \
  -d "{\"tag_uid\":\"DEMO-TAG-0001\"}"
```

Expect: popup (name, overall place, laps), sound without label; karaoke button → “recorded” after one click; test-read if race still scheduled.

## Live CSV

After any lap/racer change, station file updates automatically (path per deploy, e.g. under `data/events/{id}/live-snapshot.csv`). Copy that file for USB backup; import on a replacement laptop — **no manual export step**.

```bash
# Optional copy of live snapshot
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/events/$EVENT_ID/live-csv -o bluffet-live.csv

curl -H "Authorization: Bearer $TOKEN" \
  -F file=@bluffet-live.csv \
  http://localhost:8080/api/events/$EVENT_ID/import.csv
```

## Offline / sync

Local Postgres keeps accepting taps. IndexedDB is WAQ only. When `HOSTED_API_URL` is set and network returns, push/pull pending.

## Tests (TDD order)

```bash
docker compose -f docker-compose.test.yml run --rm backend go test ./...
docker compose -f docker-compose.test.yml run --rm frontend npm test
npx playwright test --config=frontend/e2e/playwright.config.ts
```

Coverage fail-under is **100%** for new packages/modules in this feature (FR-029). Document any hardware-only exclusions here if CI cannot instrument them (prefer MockReader coverage instead).

## Next

`/speckit-implement` — execute `tasks.md` starting Phase 1–3 (setup, foundation, failing e2e), then US1 MVP.
