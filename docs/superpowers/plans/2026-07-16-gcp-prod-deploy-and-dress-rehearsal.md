# GCP Prod Deploy + Prod-like Dress Rehearsal — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship GCP deploy artifacts for `keweenawendurance.com`, a local prod-like Compose stack with an on-laptop Proxmark **device-bridge** that keeps scoring to a **local live CSV while offline and auto-syncs** when back online, plus dress rehearsal `--prod` and outage chaos that proves Offline → syncing → online · synced with **no manual import**.

**Architecture:** Hosted Cloud Run–style origin (UI + API + Postgres) is the online authority. A local **device-bridge** owns USB Proxmark; when online it relays over WS/HTTPS; when offline it continues write/poll into a **local live CSV + durable queue**, then **automatically flushes** to hosted on reconnect. Reader UI shows **Offline → Syncing → Online · Synced**. Manual `import.csv` is emergency-only. Dress rehearsal partitions the network for 5 minutes, keeps scoring via bridge, and asserts automatic catch-up.

**Tech Stack:** Go (backend + `cmd/device-bridge`), Vue/nginx frontend, Docker Compose, Playwright, Cloud Run YAML / Cloud Build, Proxmark3 CLI

**Spec:** `docs/superpowers/specs/2026-07-16-gcp-prod-deploy-and-dress-rehearsal-design.md`

**Note on commits:** Do not commit unless the user explicitly asks. Skip commit steps during execution unless requested.

---

## File map

| File | Responsibility |
|---|---|
| `frontend/nginx.conf` | SPA + `/health` on `$PORT` / 8080 for prod frontend image |
| `frontend/Dockerfile` | Fix port, copy nginx.conf, healthcheck |
| `backend/Dockerfile` | Writable `DATA_DIR` for `appuser` |
| `backend/cmd/device-bridge/main.go` | Laptop process: pm3 owner, bridge WS client |
| `backend/internal/handlers/bridge.go` | Hosted bridge WS + write-command dispatch |
| `backend/internal/services/rfid_service.go` | WriteTag via bridge when no local hardware |
| `backend/internal/services/csv_export.go` | Optional mirror dir / GCS sink hook |
| `docker-compose.prod.yml` | Prod-like stack (no USB on hosted backend) |
| `docker-compose.bridge.yml` | Overlay notes / env for bridge against prod-like |
| `nginx/nginx.prod.conf` | `server_name keweenawendurance.com` edge |
| `deploy/*` | Cloud Build, Cloud Run YAMLs, README |
| `.env.prod.example` | Prod env placeholders |
| `scripts/run-bluffet-hardware.mjs` | `--prod [url]` parsing + safety gates |
| `frontend/src/components/SyncStatus.vue` (+ EventLive chip) | Offline / Syncing / Online · Synced; auto-sync on reconnect |
| `frontend/e2e/hardware-bluffet/lib/chaos.ts` | Network partition helpers; assert auto-sync (no import.csv) |
| `frontend/e2e/hardware-bluffet/dress-rehearsal.spec.ts` | Outage: keep scoring via bridge; assert Offline→Synced |
| `frontend/e2e/hardware-bluffet/README.md` | Prod-like + `--prod` runbook |
| `frontend/e2e/fixtures/rfid.ts` | API base from `E2E_BASE_URL` when set |

---

### Task 1: Fix frontend production image

**Files:**
- Create: `frontend/nginx.conf`
- Modify: `frontend/Dockerfile`

- [ ] **Step 1: Add SPA nginx config**

```nginx
# frontend/nginx.conf
worker_processes auto;
error_log /var/log/nginx/error.log warn;
pid /tmp/nginx.pid;

events { worker_connections 1024; }

http {
  include /etc/nginx/mime.types;
  default_type application/octet-stream;
  sendfile on;
  client_body_temp_path /tmp/client_temp;
  proxy_temp_path /tmp/proxy_temp;
  fastcgi_temp_path /tmp/fastcgi_temp;
  uwsgi_temp_path /tmp/uwsgi_temp;
  scgi_temp_path /tmp/scgi_temp;

  server {
    listen 8080;
    server_name _;
    root /usr/share/nginx/html;
    index index.html;

    location /health {
      access_log off;
      default_type text/plain;
      return 200 'ok';
    }

    location / {
      try_files $uri $uri/ /index.html;
    }
  }
}
```

- [ ] **Step 2: Update Dockerfile** to copy that config, EXPOSE 8080, HEALTHCHECK against `http://localhost:8080/health`, keep non-root user with writable `/tmp` nginx paths.

- [ ] **Step 3: Build smoke**

```powershell
docker build -t ke-frontend:prod ./frontend
docker run --rm -d -p 8088:8080 --name ke-fe-smoke ke-frontend:prod
curl -s http://localhost:8088/health
docker rm -f ke-fe-smoke
```

Expected: `ok`

---

### Task 2: Fix backend production DATA_DIR writability

**Files:**
- Modify: `backend/Dockerfile`

- [ ] **Step 1: Change final stage** so `WORKDIR /app`, create `/app/data` owned by `appuser`, copy binary to `/app/main`, default env `DATA_DIR=/app/data`.

```dockerfile
WORKDIR /app
RUN mkdir -p /app/data && chown -R appuser:appuser /app
COPY --from=builder /app/main .
RUN chown appuser:appuser /app/main
USER appuser
ENV DATA_DIR=/app/data
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1
CMD ["./main"]
```

- [ ] **Step 2: Build smoke**

```powershell
docker build -t ke-backend:prod ./backend
```

Expected: build succeeds.

---

### Task 3: Edge nginx prod config + Compose prod-like stack

**Files:**
- Create: `nginx/nginx.prod.conf`
- Create: `docker-compose.prod.yml`
- Create: `.env.prod.example`

- [ ] **Step 1: Write `nginx/nginx.prod.conf`**

```nginx
# Edge for local prod-like — server_name keweenawendurance.com
# Map via hosts: 127.0.0.1 keweenawendurance.com
worker_processes auto;
events { worker_connections 1024; }
http {
  include /etc/nginx/mime.types;
  map $http_upgrade $connection_upgrade {
    default upgrade;
    '' close;
  }
  server {
    listen 80;
    server_name keweenawendurance.com localhost;

    location /health {
      proxy_pass http://frontend:8080/health;
    }

    location /api/ {
      proxy_pass http://backend:8080/api/;
      proxy_http_version 1.1;
      proxy_set_header Host $host;
      proxy_set_header Upgrade $http_upgrade;
      proxy_set_header Connection $connection_upgrade;
      proxy_read_timeout 3600s;
    }

    location / {
      proxy_pass http://frontend:8080/;
      proxy_set_header Host $host;
    }
  }
}
```

- [ ] **Step 2: Write `docker-compose.prod.yml`**

Services: `frontend` (prod Dockerfile, no source mount), `backend` (prod Dockerfile, `GO_ENV=production`, `RFID_HARDWARE=false`, `RFID_INJECT=false`, `PROXMARK3_ENABLED=true` for scoring path when bridge feeds reads, `DATA_DIR=/app/data`, `LIVE_CSV_MIRROR_DIR=/app/csv-mirror`, `CORS_ORIGINS=http://keweenawendurance.com,http://localhost`), `postgres`, `redis`, `nginx` (ports `80:80`, mount `nginx.prod.conf`), volumes for postgres + `backend_data` + `csv_mirror`.

Hosted backend must **not** map USB devices.

- [ ] **Step 3: Write `.env.prod.example`** with `JWT_SECRET`, `DB_PASSWORD`, `ORGANIZER_PIN`, `DATA_DIR=/app/data`, `LIVE_CSV_MIRROR_DIR`, `GCS_LIVE_CSV_BUCKET=`, `BRIDGE_TOKEN=`, `CORS_ORIGINS`.

- [ ] **Step 4: Boot smoke (no hardware)**

```powershell
docker compose -f docker-compose.prod.yml up --build -d
curl -s -H "Host: keweenawendurance.com" http://127.0.0.1/health
curl -s -H "Host: keweenawendurance.com" http://127.0.0.1/api/health
```

Expected: both healthy (adjust `/api/health` path if project uses `/health` on backend only — use existing backend health route).

---

### Task 4: Live CSV mirror directory (durable stand-in for GCS)

**Files:**
- Modify: `backend/internal/config/config.go` — add `LiveCSVMirrorDir string` from `LIVE_CSV_MIRROR_DIR`
- Modify: `backend/internal/services/csv_export.go` — after `WriteLiveSnapshot`, copy file to mirror dir if set
- Test: `backend/internal/services/csv_export_test.go` — assert mirror copy appears

- [ ] **Step 1: Failing test** — when `LiveCSVMirrorDir` is set, `WriteLiveSnapshot` leaves a copy at `{mirror}/events/{id}/live-snapshot.csv`.

- [ ] **Step 2: Implement copy** using `os.MkdirAll` + `io.Copy` (same-machine atomic write via temp+rename).

- [ ] **Step 3: Run tests**

```powershell
cd backend; go test ./internal/services/ -run CSV -count=1
```

Expected: PASS. Document in deploy README that Cloud Run should set mirror to a GCS FUSE path or use a later GCS uploader; prod-like uses the compose volume.

---

### Task 5: Hosted bridge protocol (WS + write dispatch)

**Files:**
- Create: `backend/internal/handlers/bridge.go`
- Create: `backend/internal/services/bridge_hub.go` (in-memory hub: one connection per `device_id`)
- Modify: `backend/cmd/server/main.go` — register routes
- Modify: `backend/internal/services/rfid_service.go` — WriteTag uses hub when `!config.RFID.Hardware`
- Test: `backend/internal/handlers/bridge_test.go`

**Protocol (JSON over WS `/api/rfid/bridge`):**

Auth: `Authorization: Bearer <JWT>` (PIN-derived admin token) **or** `X-Bridge-Token: <BRIDGE_TOKEN>` matching env. Query: `device_id=laptop-finish-1`.

| Direction | Message |
|---|---|
| Server → Bridge | `{"type":"write","request_id":"...","logical_uuid":"..."}` |
| Bridge → Server | `{"type":"write_ack","request_id":"...","ok":true}` or `ok:false,"error":"..."` |
| Bridge → Server | `{"type":"read","logical_uuid":"...","ts":"..."}` |

On `read`, hosted calls the same scoring path as local Poll (reuse inject/score helper).

WriteTag when hardware off:

1. Ensure logical UUID for participant.
2. `hub.RequestWrite(deviceID, uuid, timeout)` — waits for matching `write_ack`.
3. Return participant.

If no bridge connected: return clear error `bridge unavailable`.

- [ ] **Step 1: Write hub unit tests** (request/ack round-trip with fake conn).

- [ ] **Step 2: Implement hub + handler + WriteTag branch**.

- [ ] **Step 3: Wire routes** under `/api/rfid/bridge` (WS upgrade). Keep existing `/api/rfid/write-tag` public shape unchanged for harness/UI.

- [ ] **Step 4: Run**

```powershell
cd backend; go test ./internal/handlers/ ./internal/services/ -count=1
```

Expected: PASS.

**Also expose bridge status for the UI** (Task 6b): hosted endpoint `GET /api/rfid/bridge/status?device_id=...` returning `{ connected, pending_count, syncing, last_sync_at }` so the reader chip can show Offline/syncing/synced. Bridge reports pending queue depth on each reconnect flush.

**Outage model (locked):** Short blips and 5‑minute outages use the **same** path — keep scoring to local CSV/queue, auto-flush on reconnect. No harness-fabricated `import.csv`.

---

### Task 6: Device-bridge binary (offline CSV + auto-sync)

**Files:**
- Create: `backend/cmd/device-bridge/main.go`
- Create: `backend/cmd/device-bridge/README.md`
- Create: `backend/internal/bridge/localcsv.go` (append-only local live CSV + pending lap queue on disk under `BRIDGE_DATA_DIR`)
- Create: `backend/internal/bridge/sync.go` (flush pending → hosted read/score API; idempotent)
- Document native Windows run (preferred — COM port)

- [ ] **Step 1: Implement `main.go` + local CSV queue**

Env:
- `HOSTED_API_URL=http://keweenawendurance.com`
- `BRIDGE_TOKEN` or `ORGANIZER_PIN`
- `DEVICE_ID=laptop-finish-1`
- `BRIDGE_DATA_DIR` (default `./bridge-data`) — local live CSV + pending queue
- `RFID_HARDWARE=true`, `PROXMARK3_CLI`, `PROXMARK3_PORT`
- `EVENT_ID` (Bluffet event id for CSV path)

Behavior:
1. Auth to hosted; dial `ws(s)://{host}/api/rfid/bridge?device_id=...`.
2. On `write` while online → local `WriteTag` → `write_ack`.
3. On poll UUID while online → send `read` to hosted.
4. **While offline (WS down / hosted unreachable):** still accept local write requests from a **local HTTP loopback** (or queue write commands received before disconnect) and continue poll; append each scored lap to `BRIDGE_DATA_DIR/events/{EVENT_ID}/live-snapshot.csv` + `pending.jsonl`. Publish status `offline`.
5. **On reconnect:** set status `syncing`; flush `pending.jsonl` to hosted (POST bridge read / batch sync endpoint) idempotently; clear queue; set status `online_synced`.
6. Never require operator to call `import.csv` for this path.

**Write-tag while offline:** Hosted `POST /api/rfid/write-tag` will fail if bridge WS is down. For dress rehearsal / race day, the harness and reader station must either (a) call a **local bridge write HTTP** (`http://127.0.0.1:8091/write-tag`) that the bridge exposes, or (b) the UI detects offline and routes write-tag to the local bridge. Prefer (a)+(b): bridge listens on `BRIDGE_LOCAL_ADDR=127.0.0.1:8091` for write-tag; frontend uses that when sync chip is Offline.

- [ ] **Step 2: Unit-test local CSV append + flush idempotency** (`go test ./internal/bridge/`).

- [ ] **Step 3: Smoke with prod-like stack**; `BRIDGE_MOCK=true` for no-hardware; real pm3 for dress rehearsal.

- [ ] **Step 4: Document native Windows launch**

```powershell
$env:HOSTED_API_URL="http://keweenawendurance.com"
$env:ORGANIZER_PIN="1738"
$env:DEVICE_ID="laptop-finish-1"
$env:EVENT_ID="1441674d-a011-471a-a601-722b88b117f5"
$env:BRIDGE_DATA_DIR="$PWD\bridge-data"
$env:RFID_HARDWARE="true"
cd backend; go run ./cmd/device-bridge
```

---

### Task 6b: Reader sync status UX (Offline / Syncing / Online · Synced)

**Files:**
- Modify: `frontend/src/components/SyncStatus.vue`
- Modify: `frontend/src/views/EventLive.vue` (sync chip)
- Modify: `frontend/src/services/api.ts` / types — bridge status fields
- Test: `frontend/src/views/EventLive.test.ts`, `SyncStatus` tests if present

- [ ] **Step 1: Failing tests** for chip text `Offline`, `Syncing`, `Online · Synced` from bridge/sync status polling.

- [ ] **Step 2: Implement** — poll `GET /api/rfid/bridge/status` (and/or enhance sync-status). On `navigator.onLine` false **or** bridge reports disconnected with pending > 0 → Offline. When flush in progress → Syncing. When connected and pending == 0 → Online · Synced. **Auto-call sync** on reconnect (no button required for happy path); keep manual button as fallback only.

- [ ] **Step 3: When Offline, route `rfidApi.writeTag` to local bridge** `http://127.0.0.1:8091/write-tag` (configurable `VITE_BRIDGE_LOCAL_URL`).

---

### Task 7: GCP deploy kit

**Files:**
- Create: `deploy/cloudbuild.yaml`
- Create: `deploy/cloud-run-backend.yaml`
- Create: `deploy/cloud-run-frontend.yaml`
- Create: `deploy/README.md`

- [ ] **Step 1: Cloud Build** — build `frontend` + `backend` images, push to `REGION-docker.pkg.dev/$PROJECT_ID/keweenaw/...`.

- [ ] **Step 2: Cloud Run backend YAML** — Cloud SQL annotation, secrets, `DATA_DIR=/app/data`, `RFID_HARDWARE=false`, `LIVE_CSV_MIRROR_DIR` or `GCS_LIVE_CSV_BUCKET`, `BRIDGE_TOKEN` secret, minScale 1, CPU always allocated if WS needed.

- [ ] **Step 3: Cloud Run frontend YAML** — port 8080, domain mapping note for `keweenawendurance.com`.

- [ ] **Step 4: README** covering: project APIs, Artifact Registry, Cloud SQL (+ **automated backups / PITR as primary DR**), GCS bucket for hosted CSV mirror, secrets, domain mapping, migrate/seed, bridge install on laptop (local CSV dir + auto-sync), rollback. Note: normal outages use **automatic bridge sync**; destructive `import.csv` is **emergency-only**, all-stop, single-operator.

---

### Task 8: Harness `--prod` flag + API base

**Files:**
- Modify: `scripts/run-bluffet-hardware.mjs`
- Modify: `frontend/e2e/hardware-bluffet/playwright.config.ts`
- Modify: `frontend/e2e/fixtures/rfid.ts`
- Modify: `frontend/package.json` (script docs only if needed)

- [ ] **Step 1: Parse argv in run script**

```js
// --prod [optionalUrl]
const args = process.argv.slice(2)
let mode = 'prod-like'
let baseURL = process.env.E2E_BASE_URL || 'http://keweenawendurance.com'
const prodIdx = args.indexOf('--prod')
if (prodIdx !== -1) {
  mode = 'prod'
  const next = args[prodIdx + 1]
  baseURL = next && !next.startsWith('-') ? next : 'https://keweenawendurance.com'
  if (/keweenawendurance\.com/i.test(baseURL) && process.env.BLUFFET_HW_ALLOW_PROD !== '1') {
    console.error('Refusing --prod against live domain without BLUFFET_HW_ALLOW_PROD=1')
    process.exit(2)
  }
}
// pass remaining args to playwright; set E2E_BASE_URL, E2E_API_URL (= same origin + '' or derive), BLUFFET_HW_MODE
```

For same-origin prod-like: set both `E2E_BASE_URL` and `E2E_API_URL` to `http://keweenawendurance.com` (fixtures currently default API to `:8080` — change to prefer `E2E_BASE_URL` when `E2E_API_URL` unset).

- [ ] **Step 2: Update `API_BASE`**

```ts
export const API_BASE =
  process.env.E2E_API_URL ??
  process.env.E2E_BASE_URL ??
  'http://localhost:8080'
```

- [ ] **Step 3: Manual dry-run**

```powershell
node scripts/run-bluffet-hardware.mjs --prod
# expect exit 2 without allow flag
$env:BLUFFET_HW_ALLOW_PROD='1'; node scripts/run-bluffet-hardware.mjs --help 2>$null
```

---

### Task 9: Outage chaos — keep scoring offline, assert automatic sync

**Files:**
- Modify: `frontend/e2e/hardware-bluffet/lib/chaos.ts` — `partitionFromHosted` / `healPartition` (block browser contexts **and** bridge→hosted path, e.g. pause bridge outbound by env signal file or docker network disconnect of nginx/backend while bridge keeps local loopback)
- Modify: `frontend/e2e/hardware-bluffet/lib/proxmark.ts` — when offline, use local bridge write URL
- Modify: `frontend/e2e/hardware-bluffet/dress-rehearsal.spec.ts`
- Modify: `frontend/e2e/hardware-bluffet/README.md`

- [ ] **Step 1: Replace outage block** so that during `[OUTAGE_START, OUTAGE_START+5m]`:
  - Assert reader chip shows **Offline**
  - Lap engine **keeps** calling write→read (via local bridge); count `outageLapsScored`
  - Assert hosted `serverLapsTotal` does **not** increase
  - Assert local bridge pending/CSV grew (read `BRIDGE_DATA_DIR` or bridge status API once healed — or poll local `http://127.0.0.1:8091/status`)
  - On heal: **do not** call `import.csv`
  - Assert chip **Syncing** then **Online · Synced**
  - Assert hosted totals ≥ pre + `outageLapsScored`; spectators `awaitCatchUp`

- [ ] **Step 2: Update README** for prod-like bring-up + bridge + auto-sync expectations (no manual CSV steps).

```powershell
# hosts: 127.0.0.1 keweenawendurance.com
docker compose -f docker-compose.prod.yml up --build -d
# seed hardware SQL
# start device-bridge natively
npm run test:e2e:bluffet-hardware
```

---

### Task 10: Seed prod-like DB + run dress rehearsal

- [ ] **Step 1: Ensure hosts entry** for `keweenawendurance.com` → `127.0.0.1`.

- [ ] **Step 2: Up prod stack, load `database/seed/03-bluffet-2026-hardware.sql`**.

- [ ] **Step 3: Start device-bridge** with real Proxmark.

- [ ] **Step 4: Run**

```powershell
cd frontend
npm run test:e2e:bluffet-hardware
```

- [ ] **Step 5: Monitor** `e2e-artifacts/bluffet-hardware/<runId>/status.json` every 60s; on critical failures, stop and fix.

- [ ] **Step 6: Mark design status Approved / Implemented** in the spec header when green.

---

## Self-review (plan vs spec)

| Spec requirement | Task |
|---|---|
| GCP deploy kit | Task 7 |
| Frontend/backend prod image fixes | Tasks 1–2 |
| Prod-like Compose + nginx domain | Task 3 |
| Hosted CSV mirror (GCS/volume) | Task 4 + Task 7 |
| Device-bridge USB owner | Tasks 5–6 |
| Offline keep scoring → local CSV | Task 6 |
| Auto-sync on reconnect (no manual import) | Task 6 + Task 9 |
| UI Offline / Syncing / Online · Synced | Task 6b |
| Write-tag reverse channel + local offline write | Tasks 5–6 |
| Bridge auth | Task 5 |
| `--prod` + safety gates | Task 8 |
| Outage chaos proves automatic path | Task 9 |
| Run prod-like rehearsal | Task 10 |
| Emergency import.csv only | Task 7 README |
| Cloud SQL PITR primary hosted DR | Task 7 README |

**Placeholder scan:** none intentional.  
**Type consistency:** bridge messages use `logical_uuid`, `request_id`, `type`; UI states use `offline` / `syncing` / `online_synced`.

---
