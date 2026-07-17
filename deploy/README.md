# GCP production deploy — keweenawendurance.com

Deploy kit for the hosted authority stack: Cloud Run (frontend + backend), Cloud SQL, Artifact Registry, GCS live CSV mirror, and Secret Manager. The **device-bridge** runs natively on the reader laptop (not in Cloud Run).

For local prod-like rehearsal before cutover, use `docker-compose.prod.yml` at the repo root.

## Architecture

```text
Reader laptop                          Cloud (keweenawendurance.com)
┌──────────────────────────┐          ┌──────────────────────────────┐
│ Reader UI                │──HTTPS──▶│ keweenaw-frontend (static)   │
│ Device-bridge (native)   │──WSS────▶│ keweenaw-backend (Go API)    │
│  pm3 USB + local CSV     │          │   └─ Cloud SQL (authority)   │
└──────────────────────────┘          │   └─ GCS live CSV mirror     │
Spectators ──────────────────────────▶└──────────────────────────────┘
```

**Normal outages:** bridge keeps scoring to local CSV; on reconnect it **automatically syncs** to hosted DB. UI: Offline → Syncing → Online · Synced.

**Emergency only:** `POST /api/events/:id/import.csv` is a destructive full replace — all-stop, single-operator, not the outage path.

---

## 1. Enable APIs

```bash
export PROJECT_ID=your-gcp-project
export REGION=us-central1

gcloud config set project $PROJECT_ID

gcloud services enable \
  artifactregistry.googleapis.com \
  cloudbuild.googleapis.com \
  run.googleapis.com \
  sqladmin.googleapis.com \
  secretmanager.googleapis.com \
  storage.googleapis.com \
  iam.googleapis.com \
  compute.googleapis.com
```

---

## 2. Artifact Registry

```bash
gcloud artifacts repositories create keweenaw \
  --repository-format=docker \
  --location=$REGION \
  --description="Keweenaw Endurance container images"

gcloud auth configure-docker ${REGION}-docker.pkg.dev
```

Build and push (from repo root):

```bash
gcloud builds submit --config=deploy/cloudbuild.yaml \
  --substitutions=_REGION=$REGION,_ARTIFACT_REPO=keweenaw
```

Images land at:

- `${REGION}-docker.pkg.dev/${PROJECT_ID}/keweenaw/backend:TAG`
- `${REGION}-docker.pkg.dev/${PROJECT_ID}/keweenaw/frontend:TAG`

---

## 3. Cloud SQL (primary hosted DR)

Create a PostgreSQL 14 instance with **automated backups** and **point-in-time recovery (PITR)** enabled. PITR is the **primary disaster-recovery path** for hosted data.

```bash
export CLOUD_SQL_INSTANCE=keweenaw-prod

gcloud sql instances create $CLOUD_SQL_INSTANCE \
  --database-version=POSTGRES_14 \
  --tier=db-custom-2-7680 \
  --region=$REGION \
  --storage-auto-increase \
  --backup-start-time=04:00 \
  --enable-point-in-time-recovery

gcloud sql databases create keweenaw_timing --instance=$CLOUD_SQL_INSTANCE

gcloud sql users create timing_user \
  --instance=$CLOUD_SQL_INSTANCE \
  --password='GENERATE_STRONG_PASSWORD'
```

Store the DB password in Secret Manager (see §5). Grant the backend service account `roles/cloudsql.client`.

**Restore / DR:** use Cloud SQL backup restore or PITR to a new instance or timestamp. Do not rely on Cloud Run ephemeral `DATA_DIR=/app/data`.

---

## 4. GCS bucket (hosted live CSV mirror)

Secondary DR artifact: backend mirrors live CSV snapshots to GCS when `GCS_LIVE_CSV_BUCKET` is set (recommended on Cloud Run).

```bash
export GCS_LIVE_CSV_BUCKET=${PROJECT_ID}-keweenaw-live-csv

gsutil mb -l $REGION gs://$GCS_LIVE_CSV_BUCKET

# Backend SA needs objectAdmin on this bucket
gsutil iam ch \
  serviceAccount:keweenaw-backend@${PROJECT_ID}.iam.gserviceaccount.com:objectAdmin \
  gs://$GCS_LIVE_CSV_BUCKET
```

On Cloud Run, prefer `GCS_LIVE_CSV_BUCKET` over `LIVE_CSV_MIRROR_DIR` (no persistent disk). Local prod-like Compose uses a volume for `LIVE_CSV_MIRROR_DIR`.

---

## 5. Secrets (Secret Manager)

Create secrets and grant `roles/secretmanager.secretAccessor` to `keweenaw-backend@...`:

| Secret name | Purpose |
|---|---|
| `keweenaw-db-password` | Cloud SQL `timing_user` password |
| `keweenaw-jwt-secret` | JWT signing key |
| `keweenaw-organizer-pin` | PIN exchange for admin JWT |
| `keweenaw-bridge-token` | Bridge WebSocket auth (`X-Bridge-Token`) |

```bash
echo -n 'YOUR_DB_PASSWORD' | gcloud secrets create keweenaw-db-password --data-file=-
echo -n 'YOUR_JWT_SECRET'  | gcloud secrets create keweenaw-jwt-secret --data-file=-
echo -n 'YOUR_PIN'           | gcloud secrets create keweenaw-organizer-pin --data-file=-
openssl rand -hex 32 | gcloud secrets create keweenaw-bridge-token --data-file=-
```

Copy the bridge token to the reader laptop env (`BRIDGE_TOKEN`). Never commit secrets.

---

## 6. Service accounts

```bash
gcloud iam service-accounts create keweenaw-backend \
  --display-name="Keweenaw Cloud Run backend"

gcloud iam service-accounts create keweenaw-frontend \
  --display-name="Keweenaw Cloud Run frontend"

gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:keweenaw-backend@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role="roles/cloudsql.client"
```

Optional: Memorystore Redis for leaderboard cache (`REDIS_HOST` in backend YAML). The app degrades gracefully without Redis.

---

## 7. Deploy Cloud Run services

Edit placeholders in `deploy/cloud-run-backend.yaml` and `deploy/cloud-run-frontend.yaml` (`$PROJECT_ID`, `$PROJECT_NUMBER`, `$REGION`, `$CLOUD_SQL_INSTANCE`, image tags), then:

```bash
export PROJECT_NUMBER=$(gcloud projects describe $PROJECT_ID --format='value(projectNumber)')
```

`$PROJECT_NUMBER` is used in the backend's `run.googleapis.com/secrets` annotation to bind `secretKeyRef` aliases to Secret Manager paths.

```bash
gcloud run services replace deploy/cloud-run-backend.yaml \
  --region=$REGION --project=$PROJECT_ID

gcloud run services replace deploy/cloud-run-frontend.yaml \
  --region=$REGION --project=$PROJECT_ID
```

Backend highlights (see YAML):

- Cloud SQL annotation + `DB_HOST=/cloudsql/...`
- `RFID_HARDWARE=false` (Proxmark stays on laptop bridge)
- `DATA_DIR=/app/data` (ephemeral)
- `GCS_LIVE_CSV_BUCKET` for hosted CSV mirror
- `BRIDGE_TOKEN` from Secret Manager
- `minScale: 1`, `cpu-throttling: false`, session affinity for bridge WebSocket

Frontend listens on **port 8080** (nginx in `frontend/Dockerfile`).

Allow unauthenticated invoke on both services if using public HTTPS (or front with IAP / LB as needed).

---

## 8. Domain mapping — keweenawendurance.com

Two common patterns:

### A. Single origin via Cloud Load Balancer (recommended)

- HTTPS load balancer with managed certificate for `keweenawendurance.com` and `www.keweenawendurance.com`
- URL map:
  - `/api/*` → serverless NEG → `keweenaw-backend`
  - `/*` → serverless NEG → `keweenaw-frontend`
- WebSocket upgrade headers for `/api/rfid/bridge` (backend timeout 3600s in YAML)

### B. Separate Cloud Run domain mappings

```bash
gcloud run domain-mappings create \
  --service=keweenaw-frontend \
  --domain=keweenawendurance.com \
  --region=$REGION

# API subdomain example if not using path-based routing:
gcloud run domain-mappings create \
  --service=keweenaw-backend \
  --domain=api.keweenawendurance.com \
  --region=$REGION
```

Point DNS A/AAAA or CNAME records at Google's targets. For same-origin cookies and CORS, prefer pattern A so UI and API share `https://keweenawendurance.com`.

Update backend `CORS_ORIGINS=https://keweenawendurance.com`.

---

## 9. Migrate and seed database

Run from a machine with Cloud SQL Auth Proxy (or Cloud Shell):

```bash
cloud-sql-proxy ${PROJECT_ID}:${REGION}:${CLOUD_SQL_INSTANCE} &
export PGPASSWORD='...'

psql -h 127.0.0.1 -U timing_user -d keweenaw_timing -f database/init/01-init.sql
for f in database/migrations/*.sql; do
  psql -h 127.0.0.1 -U timing_user -d keweenaw_timing -f "$f"
done

# Production event seed (adjust for your race):
psql -h 127.0.0.1 -U timing_user -d keweenaw_timing \
  -f database/seed/03-bluffet-2026.sql
```

Re-run seeds only on fresh databases; they are not idempotent for production cutover.

---

## 10. Reader laptop — device-bridge install

The bridge **must run natively** on Windows (COM port access). See `backend/cmd/device-bridge/README.md`.

```powershell
$env:HOSTED_API_URL="https://keweenawendurance.com"
$env:BRIDGE_TOKEN="<same as keweenaw-bridge-token secret>"
$env:DEVICE_ID="laptop-finish-1"
$env:EVENT_ID="<production event UUID>"
$env:BRIDGE_DATA_DIR="C:\keweenaw\bridge-data"
$env:RFID_HARDWARE="true"
$env:PROXMARK3_PORT="COM3"

cd backend
go build -o device-bridge.exe ./cmd/device-bridge
.\device-bridge.exe
```

Local layout:

```text
bridge-data/events/{EVENT_ID}/live-snapshot.csv   # offline continuity
bridge-data/events/{EVENT_ID}/pending.jsonl       # auto-flush queue
```

When hosted is unreachable, scoring continues into local CSV. On reconnect, the bridge flushes automatically — **no operator import step**.

---

## 11. Rollback

### Application

```bash
# List revisions
gcloud run revisions list --service=keweenaw-backend --region=$REGION

# Route 100% traffic to a known-good revision
gcloud run services update-traffic keweenaw-backend \
  --to-revisions=keweenaw-backend-00042-abc=100 \
  --region=$REGION
```

Redeploy a previous image tag from Artifact Registry if needed:

```bash
gcloud run services update keweenaw-backend \
  --image=${REGION}-docker.pkg.dev/${PROJECT_ID}/keweenaw/backend:PREVIOUS_SHA \
  --region=$REGION
```

Repeat for `keweenaw-frontend`.

### Database

Use **Cloud SQL PITR** or backup restore — not CSV import — for hosted DB corruption or bad deploys affecting data.

### Emergency CSV import (last resort)

`POST /api/events/:id/import.csv` replaces timing data destructively. Requirements:

- All scoring stopped (bridge disconnected, no active race writes)
- Single operator
- Documented incident; not used for connectivity blips

Normal outages rely on **automatic bridge sync** only.

---

## 12. Pre-cutover checklist

- [ ] Cloud SQL backups + PITR verified
- [ ] Secrets rotated from dev defaults
- [ ] `RFID_HARDWARE=false` on backend
- [ ] Bridge token on laptop matches Secret Manager
- [ ] GCS bucket IAM for backend SA
- [ ] Domain + TLS live on `keweenawendurance.com`
- [ ] Migrations applied; production seed loaded
- [ ] Bridge offline → online dress rehearsal against staging or prod-like Compose
- [ ] Rollback revision/image tag documented

---

## Files in this directory

| File | Purpose |
|---|---|
| `cloudbuild.yaml` | Build/push frontend + backend to Artifact Registry |
| `cloud-run-backend.yaml` | Backend service (Cloud SQL, secrets, bridge WS) |
| `cloud-run-frontend.yaml` | Frontend static nginx on port 8080 |
| `README.md` | This guide |
