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

## CI/CD + greenfield bootstrap

**Recommended operator sequence** for a new GCP project. Bash scripts live in `deploy/`; rendered Cloud Run YAML lands in `deploy/.generated/` (gitignored).

1. **Create GCP project** and enable billing.
2. **`gcloud auth login`** and set context:

   ```bash
   export PROJECT_ID=your-gcp-project
   export REGION=us-central1
   ```

3. **`bash deploy/bootstrap.sh`** — provisions APIs, Artifact Registry, Cloud SQL, secrets, GCS bucket, and service accounts. **Save the printed `BRIDGE_TOKEN` and CI key file path** (for the reader laptop and GitHub secrets).
4. **Initial images + Cloud Run** — build and push backend + frontend to Artifact Registry (see §2 for an optional Cloud Build path), then:

   ```bash
   export IMAGE_TAG=bootstrap   # or git SHA
   bash deploy/deploy-cloud-run.sh
   ```

   `deploy-cloud-run.sh` calls `render-cloud-run.sh` (envsubst on the templates) and applies `deploy/.generated/cloud-run-*.yaml`.
5. **Database migrations:**

   ```bash
   DB_PASSWORD='...' bash deploy/migrate-cloud-sql.sh
   ```

   Uses Cloud SQL Auth Proxy; applies `database/init` and `database/migrations`. Seed production data separately (see §9).
6. **HTTPS load balancer + DNS:**

   ```bash
   bash deploy/bootstrap-lb.sh
   ```

   Create **DNS A records** for apex and `www` to the printed static IP. Wait until the managed certificate is `ACTIVE`:

   ```bash
   gcloud compute ssl-certificates describe keweenaw-cert --global --format='value(managed.status)'
   ```

7. **GitHub repo secrets** (Settings → Secrets and variables → Actions):

   | Secret | Value |
   |---|---|
   | `GCP_SA_KEY` | Full JSON contents of the CI key file from bootstrap |
   | `GCP_PROJECT_ID` | Your GCP project ID |
   | `GCP_REGION` | e.g. `us-central1` |

8. **Ongoing deploys:** push to `main` with **`[deploy]` in the commit message**. GitHub Actions builds/pushes images and runs `deploy/deploy-cloud-run.sh`. Pushes without `[deploy]` run CI only.

**Windows:** run bash scripts via **Git Bash** or **WSL**. Install `gcloud` CLI and **Docker Desktop** for local image builds.

**HTTP → HTTPS redirect:** not yet automated in `bootstrap-lb.sh` (Task 5 deferred). Operators can add a classic HTTP redirect URL map later if apex HTTP should redirect to HTTPS.

---

## 1. Enable APIs

`deploy/bootstrap.sh` enables the required APIs. To enable manually:

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
  compute.googleapis.com \
  certificatemanager.googleapis.com
```

---

## 2. Artifact Registry *(optional — Cloud Build image builds)*

Use this path for **manual or one-off** image builds. Ongoing deploys normally use **GitHub Actions** on `[deploy]` pushes (see CI/CD section above).

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

`deploy/bootstrap.sh` creates the instance, database, and `timing_user`. Manual reference:

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

`deploy/bootstrap.sh` creates secrets and grants accessor to the backend SA. Manual reference:

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

`deploy/bootstrap.sh` creates `keweenaw-backend`, `keweenaw-frontend`, and `keweenaw-ci`. Manual reference:

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

Templates `deploy/cloud-run-backend.yaml` and `deploy/cloud-run-frontend.yaml` hold `$PROJECT_ID`, `$PROJECT_NUMBER`, `$REGION`, `$CLOUD_SQL_INSTANCE`, and `$IMAGE_TAG` placeholders. **Do not hand-edit and apply the templates directly.**

Render and deploy:

```bash
export PROJECT_ID=your-gcp-project
export REGION=us-central1
export CLOUD_SQL_INSTANCE=keweenaw-prod
export IMAGE_TAG=your-git-sha-or-tag

bash deploy/deploy-cloud-run.sh
```

This runs `render-cloud-run.sh` → `deploy/.generated/cloud-run-*.yaml`, then `gcloud run services replace` on the rendered files.

`$PROJECT_NUMBER` is resolved automatically if unset (used in the backend's `run.googleapis.com/secrets` annotation to bind `secretKeyRef` aliases to Secret Manager paths).

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

Run **`bash deploy/bootstrap-lb.sh`** for the HTTPS LB, managed cert, path routing (`/api/*` → backend, `/*` → frontend), and static IP. Point DNS A records at the printed IP.

Manual reference:

- HTTPS load balancer with managed certificate for `keweenawendurance.com` and `www.keweenawendurance.com`
- URL map:
  - `/api/*` → serverless NEG → `keweenaw-backend`
  - `/*` → serverless NEG → `keweenaw-frontend`
- WebSocket upgrade headers for `/api/rfid/bridge` (backend timeout 3600s in YAML)

**HTTP → HTTPS redirect** is not created by `bootstrap-lb.sh` yet; add a classic HTTP redirect URL map separately if needed.

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

**Preferred:** `DB_PASSWORD=... bash deploy/migrate-cloud-sql.sh` (Cloud SQL Auth Proxy + `database/init` + migrations).

Manual reference from a machine with Cloud SQL Auth Proxy (or Cloud Shell):

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

- [ ] `bash deploy/bootstrap.sh` complete; `BRIDGE_TOKEN` saved
- [ ] GitHub secrets: `GCP_SA_KEY`, `GCP_PROJECT_ID`, `GCP_REGION`
- [ ] Initial images pushed; `deploy/deploy-cloud-run.sh` applied
- [ ] `deploy/migrate-cloud-sql.sh` run; production seed loaded if needed
- [ ] `deploy/bootstrap-lb.sh` run; DNS A records set; managed cert `ACTIVE`
- [ ] Cloud SQL backups + PITR verified
- [ ] Secrets rotated from dev defaults (especially `ORGANIZER_PIN`)
- [ ] `RFID_HARDWARE=false` on backend
- [ ] Bridge token on laptop matches Secret Manager
- [ ] GCS bucket IAM for backend SA
- [ ] Domain + TLS live on `keweenawendurance.com`
- [ ] Bridge offline → online dress rehearsal against staging or prod-like Compose
- [ ] Rollback revision/image tag documented
- [ ] Smoke: `[deploy]` push on `main` updates Cloud Run revisions

---

## Files in this directory

| File | Purpose |
|---|---|
| `bootstrap.sh` | One-time greenfield: APIs, AR, SQL, secrets, GCS, service accounts, CI key |
| `bootstrap-lb.sh` | HTTPS LB, managed cert, path routing, static IP + DNS instructions |
| `deploy-cloud-run.sh` | Render templates and `gcloud run services replace` |
| `render-cloud-run.sh` | envsubst templates → `deploy/.generated/` |
| `migrate-cloud-sql.sh` | Auth Proxy + `database/init` + migrations |
| `.generated/` | Rendered Cloud Run YAML (gitignored; created by render script) |
| `cloudbuild.yaml` | Optional: build/push frontend + backend to Artifact Registry |
| `cloud-run-backend.yaml` | Backend service template (Cloud SQL, secrets, bridge WS) |
| `cloud-run-frontend.yaml` | Frontend service template (static nginx on port 8080) |
| `README.md` | This guide |
