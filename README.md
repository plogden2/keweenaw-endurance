# Keweenaw Endurance Syndicate

Race timing and event management system for endurance events in the Keweenaw area.

## Stack

- **Frontend**: Vue 3 + Vite
- **Backend**: Go (Gin) + GORM
- **Database**: PostgreSQL 14
- **Cache**: Redis
- **Containers**: Docker Compose

## Development

```bash
docker compose up
```

- Frontend: http://localhost:3000
- Backend API: http://localhost:8080
- Health check: http://localhost:8080/health

### Run backend tests

```bash
docker compose -f docker-compose.test.yml run --rm backend-test
```

## API (MVP)

Base URL: `http://localhost:8080/api`

**Authentication**: Not enabled in the MVP. All endpoints below are unauthenticated. JWT middleware and role-based access are planned for a later phase; do not expose this API to the public internet without auth.

### Events

| Method | Path | Notes |
|--------|------|-------|
| GET | `/events` | Query: `page`, `limit` |
| POST | `/events` | Requires `name`, `event_date` (YYYY-MM-DD) |
| GET | `/events/:id` | |
| PUT | `/events/:id` | Partial update: send only fields to change |
| DELETE | `/events/:id` | Hard delete (cascades to races) |

### Races

| Method | Path | Notes |
|--------|------|-------|
| GET | `/races` | Query: `page`, `limit`, `event_id` |
| POST | `/races` | Requires `event_id`, `name`, `race_type` (`time_based` or `lap_based`) |
| GET | `/races/:id` | |
| PUT | `/races/:id` | Partial update |
| DELETE | `/races/:id` | |

### Participants

| Method | Path | Notes |
|--------|------|-------|
| GET | `/participants` | Query: `page`, `limit`, `race_id` |
| POST | `/participants` | Requires `race_id`, `bib_number`, `first_name`, `last_name` |
| GET | `/participants/:id` | |
| PUT | `/participants/:id` | Partial update; `rfid_tag_uid` must be unique when set |
| DELETE | `/participants/:id` | |

### Not yet implemented (501)

- `/timing/*` — live timing, results, leaderboard
- `/rfid/*` — tag read/write, manual entry, sync status

## Spec Kit

Feature specs and tasks live under `specs/001-race-timing/`. See `.specify/feature.json` for the active feature directory.
