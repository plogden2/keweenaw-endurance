# Implementation Plan: RFID Race Scanner

**Branch**: `002-rfid-race-scanner` | **Date**: 2026-07-12 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/002-rfid-race-scanner/spec.md`

**Status**: Ready for `/speckit-implement` — HTML prototypes approved; clarifications complete.

## Summary

Deliver full race-day RFID timing for Proxmark3: event-scoped reader stations (finish default, optional checkpoints), continuous background reads via **WebSocket**, 1-minute cooldown, lap popup + Mario Kart sound (no sound label), karaoke one-click (shows recorded state after use), Racers page (debounced search, click-to-edit bib, inline tag program), PIN `1738` management with public live view, pre-start countdown + auto-start at `start_time`, live overall leaderboards with category colors/legend, race tabs + overlap chart + fullscreen rotator, **local Postgres authority** + IndexedDB WAQ + hosted sync (`HOSTED_API_URL`), **continuously maintained live CSV** + import recovery, multi-station (≥3), AYCEB 2026 seed (100 racers / 3 races), e2e-first TDD with **100%** coverage fail-under for new packages.

Technical approach: extend Vue + Go + PostgreSQL Docker stack; implement approved prototypes in `frontend/prototypes/002-rfid-race-scanner/`; mock Proxmark3 for CI.

## Technical Context

**Language/Version**: Go 1.22+ (backend), TypeScript 5.x / Vue 3 (frontend)

**Primary Dependencies**: Gin, GORM, PostgreSQL driver, Redis (leaderboard cache); Vue 3 + Pinia + Vue Router + Vite; Playwright for e2e; Proxmark3 via `backend/internal/rfid` (extended poll + WebSocket stream)

**Storage**: PostgreSQL 14 — **per-station Docker Postgres is authority**; hosted Postgres via sync; IndexedDB only as short WAQ/UI cache when local API blips; **live CSV file** updated on every relevant mutation for disaster recovery

**Testing**: Go `testing` + Testify; Vitest + Vue Test Utils; Playwright e2e; **100%** coverage fail-under for new packages; Proxmark3 `MockReader` + inject endpoint

**Target Platform**: Dockerized web app (race-day laptops + GCP Cloud Run / Cloud SQL)

**Project Type**: Web application (frontend + backend) with hardware-attached reader stations

**Performance Goals**: Tap → UI ≤ 2s; countdown drift ≤ 1s; ≥ 3 concurrent reader stations; search debounce ~200–300ms

**Constraints**: Offline-capable; continuous read across SPA routes; constitution TDD + **approved** HTML prototypes; PIN management; no tag revoke; live CSV always current

**Scale/Scope**: ~100 demo racers, 3 races, multi-category; arbitrary readers (validate ≥ 3)

## Constitution Check

| Gate | Status | Notes |
|------|--------|-------|
| I. TDD | PASS | US9 e2e red suite before implementation; FR-029 100% coverage gate |
| II. Stack | PASS | Vue + Go + Postgres + Docker + GCP |
| III. Race data focus | PASS | Core timing |
| IV. Container-first | PASS | Station stack in Compose; Proxmark3 via container device passthrough when used |
| V. UI HTML prototypes | PASS | Prototypes created **and user-approved** 2026-07-12 |
| VI. GCP | PASS | Hosted sync target Cloud SQL–compatible |
| VII. Comprehensive testing | PASS | Unit + integration + e2e + **100%** coverage CI fail-under for new packages |

## Project Structure

### Documentation (this feature)

```text
specs/002-rfid-race-scanner/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── api-rfid-scanner.md
│   └── csv-race-export.md
└── tasks.md
```

### Source Code

```text
backend/internal/{rfid,models,services,handlers,database}/
frontend/src/{views,components,composables,services,stores,assets/audio}/
frontend/prototypes/002-rfid-race-scanner/   # approved HTML (reference)
frontend/e2e/                               # Playwright
database/{migrations,seed}/
docker-compose.yml / docker-compose.test.yml
```

**Structure Decision**: Extend existing `frontend/` + `backend/` app. No new top-level service.

## Complexity Tracking

| Item | Why Needed | Simpler Alternative Rejected Because |
|------|------------|--------------------------------------|
| Local Postgres + hosted sync | Offline + multi-station | IndexedDB-only insufficient |
| Live CSV always on | Recovery without export | Manual export fails if forgotten offline |
| Finish + checkpoint modes | Clarified product need | Finish-only blocks course layouts |
| Overall + category colors | Approved live UX | Category-only boards hide cross-field competition |
