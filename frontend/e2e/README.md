# RFID Race Scanner — Playwright e2e

End-to-end suite for Speckit feature `002-rfid-race-scanner` (US1–US8 acceptance).

## Status / gate

The suite is **intentionally red** until the corresponding user stories land. Specs under `frontend/e2e/*.spec.ts` are written first (TDD); they fail against missing routes, APIs, and UI until implementation catches up.

| Phase | Expectation |
| --- | --- |
| Stories not yet implemented | Suite fails (allowed in CI via `continue-on-error: true`) |
| Stories complete | Suite must be **green** and the CI e2e job becomes **required** (remove `continue-on-error`) |

## Prerequisites

1. Docker Desktop / Docker Compose
2. Local (or test) stack with seed loaded — see `specs/002-rfid-race-scanner/quickstart.md`
3. Backend with mock RFID inject enabled (`RFID_INJECT=true` and/or `GO_ENV=test`)
4. Frontend reachable at `http://localhost:3000` (override with `E2E_BASE_URL`)
5. API at `http://localhost:8080` (override with `E2E_API_URL`)

### Boot stack + seed

```bash
# From repo root
docker compose up --build

python database/seed/generate_bluffet_seed.py
docker compose exec -T postgres psql -U timing_user -d keweenaw_timing < database/seed/03-bluffet-2026.sql
```

Unit/integration compose (backend/frontend tests only — not a full browser stack):

```bash
docker compose -f docker-compose.test.yml run --rm backend-test
docker compose -f docker-compose.test.yml run --rm frontend-test
```

## Run e2e

From `frontend/`:

```bash
npm run test:e2e
```

Equivalent:

```bash
npx playwright test --config=e2e/playwright.config.ts
```

Install browsers once if needed: `npx playwright install chromium`.

## Mock RFID inject

Hardware is simulated via the backend mock reader:

```bash
curl -s -X POST http://localhost:8080/api/rfid/inject \
  -H "Content-Type: application/json" \
  -d "{\"tag_uid\":\"DEMO-TAG-0001\"}"
```

Helpers in `e2e/fixtures/rfid.ts`:

- `ORGANIZER_PIN` — default `1738`
- `API_BASE` — `http://localhost:8080`
- `inject(request, tagUid)` — POST `/api/rfid/inject`
- `pinLogin(page)` — unlock management UI
- `BLUFFET` — seeded event/race/tag assumptions

## Spec map

| Spec | Story |
| --- | --- |
| `live-lap-timing.spec.ts` | US1 |
| `karaoke-bonus.spec.ts` | US2 |
| `racers-page.spec.ts` | US3 |
| `race-crud.spec.ts` | US4 |
| `offline-sync.spec.ts` | US5 |
| `csv-recovery.spec.ts` | US6 |
| `multi-station.spec.ts` | US7 |
| `demo-seed.spec.ts` | US8 |
