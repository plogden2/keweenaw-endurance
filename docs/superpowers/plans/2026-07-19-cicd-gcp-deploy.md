# CI/CD + GCP Production Deploy — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `[deploy]`-gated GitHub Actions CD plus one-time GCP bootstrap (Cloud Run, Cloud SQL, secrets, GCS, HTTPS LB) for `keweenawendurance.com`.

**Architecture:** Keep existing CI jobs. Add a `deploy` job that runs only on `main` pushes whose commit message contains `[deploy]`. That job authenticates with `GCP_SA_KEY`, builds/pushes images to Artifact Registry, and updates Cloud Run. One-time `deploy/bootstrap.sh` (+ LB script) provisions greenfield infra; CD never recreates SQL/LB/secrets.

**Tech Stack:** GitHub Actions, Docker, `gcloud`, Artifact Registry, Cloud Run, Cloud SQL (Postgres 14), Secret Manager, GCS, global HTTPS Load Balancer + managed cert, bash scripts with `envsubst`.

**Spec:** `docs/superpowers/specs/2026-07-19-cicd-gcp-deploy-design.md`

---

## File map

| File | Responsibility |
|------|----------------|
| `deploy/cloud-run-backend.yaml` | Template with `${PROJECT_ID}`-style placeholders for `envsubst` |
| `deploy/cloud-run-frontend.yaml` | Same for frontend |
| `deploy/render-cloud-run.sh` | Render templates → `deploy/.generated/` |
| `deploy/deploy-cloud-run.sh` | Render + `gcloud run services replace` for both services |
| `deploy/bootstrap.sh` | One-time APIs, AR, SAs, Cloud SQL, secrets, GCS, CI key |
| `deploy/bootstrap-lb.sh` | Serverless NEGs, URL map, HTTPS proxy, forwarding rules, managed cert |
| `deploy/migrate-cloud-sql.sh` | Cloud SQL Auth Proxy + apply init/migrations |
| `deploy/README.md` | Operator checklist (project → bootstrap → secrets → DNS → `[deploy]`) |
| `.github/workflows/ci.yml` | Add gated `deploy` job after `test` + `e2e` |

---

### Task 1: Convert Cloud Run YAML placeholders for envsubst

**Files:**
- Modify: `deploy/cloud-run-backend.yaml`
- Modify: `deploy/cloud-run-frontend.yaml`

- [ ] **Step 1: Normalize placeholders**

Replace shell-style `$PROJECT_ID`, `$REGION`, `$PROJECT_NUMBER`, `$CLOUD_SQL_INSTANCE` with braced forms that `envsubst` expects, and use an explicit image tag placeholder:

Backend image line becomes:

```yaml
image: ${REGION}-docker.pkg.dev/${PROJECT_ID}/keweenaw/backend:${IMAGE_TAG}
```

Frontend:

```yaml
image: ${REGION}-docker.pkg.dev/${PROJECT_ID}/keweenaw/frontend:${IMAGE_TAG}
```

Cloud SQL annotation:

```yaml
run.googleapis.com/cloudsql-instances: ${PROJECT_ID}:${REGION}:${CLOUD_SQL_INSTANCE}
```

Secrets annotation and all other `$VAR` occurrences → `${VAR}`. Keep `DB_HOST` as:

```yaml
value: /cloudsql/${PROJECT_ID}:${REGION}:${CLOUD_SQL_INSTANCE}
```

GCS bucket:

```yaml
value: ${PROJECT_ID}-keweenaw-live-csv
```

Service account:

```yaml
serviceAccountName: keweenaw-backend@${PROJECT_ID}.iam.gserviceaccount.com
```

(and frontend SA likewise).

Update file header comments to say: render via `deploy/render-cloud-run.sh` (do not apply raw templates).

- [ ] **Step 2: Sanity-check required vars list**

Document in comments at top of each YAML the required env vars:

```text
PROJECT_ID PROJECT_NUMBER REGION CLOUD_SQL_INSTANCE IMAGE_TAG
```

- [ ] **Step 3: Commit**

```bash
git add deploy/cloud-run-backend.yaml deploy/cloud-run-frontend.yaml
git commit -m "chore(deploy): use envsubst placeholders in Cloud Run YAMLs"
```

---

### Task 2: Add render + deploy scripts

**Files:**
- Create: `deploy/render-cloud-run.sh`
- Create: `deploy/deploy-cloud-run.sh`

- [ ] **Step 1: Create `deploy/render-cloud-run.sh`**

```bash
#!/usr/bin/env bash
# Render Cloud Run service YAMLs with envsubst.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${OUT_DIR:-${ROOT}/deploy/.generated}"
mkdir -p "${OUT_DIR}"

: "${PROJECT_ID:?PROJECT_ID is required}"
: "${PROJECT_NUMBER:?PROJECT_NUMBER is required}"
: "${REGION:?REGION is required}"
: "${CLOUD_SQL_INSTANCE:?CLOUD_SQL_INSTANCE is required}"
: "${IMAGE_TAG:?IMAGE_TAG is required}"

VARS='${PROJECT_ID} ${PROJECT_NUMBER} ${REGION} ${CLOUD_SQL_INSTANCE} ${IMAGE_TAG}'

envsubst "${VARS}" < "${ROOT}/deploy/cloud-run-backend.yaml" \
  > "${OUT_DIR}/cloud-run-backend.yaml"
envsubst "${VARS}" < "${ROOT}/deploy/cloud-run-frontend.yaml" \
  > "${OUT_DIR}/cloud-run-frontend.yaml"

echo "Rendered:"
echo "  ${OUT_DIR}/cloud-run-backend.yaml"
echo "  ${OUT_DIR}/cloud-run-frontend.yaml"
```

- [ ] **Step 2: Create `deploy/deploy-cloud-run.sh`**

```bash
#!/usr/bin/env bash
# Render and apply Cloud Run services (app deploy only — no SQL/LB).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${OUT_DIR:-${ROOT}/deploy/.generated}"

: "${PROJECT_ID:?}"
: "${REGION:=us-central1}"
: "${CLOUD_SQL_INSTANCE:=keweenaw-prod}"
: "${IMAGE_TAG:?IMAGE_TAG is required (git SHA)}"

if [[ -z "${PROJECT_NUMBER:-}" ]]; then
  PROJECT_NUMBER="$(gcloud projects describe "${PROJECT_ID}" --format='value(projectNumber)')"
  export PROJECT_NUMBER
fi

export PROJECT_ID PROJECT_NUMBER REGION CLOUD_SQL_INSTANCE IMAGE_TAG
bash "${ROOT}/deploy/render-cloud-run.sh"

gcloud run services replace "${OUT_DIR}/cloud-run-backend.yaml" \
  --region="${REGION}" --project="${PROJECT_ID}"
gcloud run services replace "${OUT_DIR}/cloud-run-frontend.yaml" \
  --region="${REGION}" --project="${PROJECT_ID}"

echo "Deployed keweenaw-backend and keweenaw-frontend @ ${IMAGE_TAG}"
```

- [ ] **Step 3: Make executable + ignore generated YAMLs**

Append to `.gitignore`:

```gitignore
deploy/.generated/
```

```bash
chmod +x deploy/render-cloud-run.sh deploy/deploy-cloud-run.sh
```

On Windows, CI should call `bash deploy/deploy-cloud-run.sh`.

- [ ] **Step 4: Dry-run render locally (no gcloud apply)**

```bash
export PROJECT_ID=example-proj PROJECT_NUMBER=123456789012
export REGION=us-central1 CLOUD_SQL_INSTANCE=keweenaw-prod IMAGE_TAG=abc1234
bash deploy/render-cloud-run.sh
grep -E "abc1234|example-proj|123456789012" deploy/.generated/cloud-run-backend.yaml
grep -E "abc1234|example-proj" deploy/.generated/cloud-run-frontend.yaml
```

Expected: both files contain substituted values; no literal `${PROJECT_ID}` left in image/SA lines.

- [ ] **Step 5: Commit**

```bash
git add deploy/render-cloud-run.sh deploy/deploy-cloud-run.sh .gitignore
git commit -m "chore(deploy): add Cloud Run render and deploy scripts"
```

---

### Task 3: Add Cloud SQL migrate helper

**Files:**
- Create: `deploy/migrate-cloud-sql.sh`

- [ ] **Step 1: List real init/migration paths**

```bash
ls database/init database/migrations
```

Point the script at the actual init SQL file(s).

- [ ] **Step 2: Write script**

```bash
#!/usr/bin/env bash
# Apply database/init + database/migrations against Cloud SQL via Auth Proxy.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

: "${PROJECT_ID:?}"
: "${REGION:=us-central1}"
: "${CLOUD_SQL_INSTANCE:=keweenaw-prod}"
: "${DB_USER:=timing_user}"
: "${DB_NAME:=keweenaw_timing}"
: "${DB_PASSWORD:?DB_PASSWORD is required}"

CONNECTION="${PROJECT_ID}:${REGION}:${CLOUD_SQL_INSTANCE}"
PORT="${PROXY_PORT:-5433}"

if ! command -v cloud-sql-proxy >/dev/null 2>&1 && ! command -v cloud_sql_proxy >/dev/null 2>&1; then
  echo "Install Cloud SQL Auth Proxy: https://cloud.google.com/sql/docs/postgres/connect-auth-proxy"
  exit 1
fi

PROXY_BIN="$(command -v cloud-sql-proxy || command -v cloud_sql_proxy)"
"${PROXY_BIN}" "${CONNECTION}" --port "${PORT}" &
PROXY_PID=$!
trap 'kill ${PROXY_PID} 2>/dev/null || true' EXIT
sleep 3

export PGPASSWORD="${DB_PASSWORD}"
psql -h 127.0.0.1 -p "${PORT}" -U "${DB_USER}" -d "${DB_NAME}" -v ON_ERROR_STOP=1 \
  -f "${ROOT}/database/init/01-init.sql"

shopt -s nullglob
for f in "${ROOT}/database/migrations/"*.sql; do
  echo "Applying ${f}"
  psql -h 127.0.0.1 -p "${PORT}" -U "${DB_USER}" -d "${DB_NAME}" -v ON_ERROR_STOP=1 -f "${f}"
done

echo "Migrations complete. Seed separately if needed (see deploy/README.md)."
```

Adjust `-f` init path if the repo uses a different filename (from Step 1).

- [ ] **Step 3: chmod + commit**

```bash
chmod +x deploy/migrate-cloud-sql.sh
git add deploy/migrate-cloud-sql.sh
git commit -m "chore(deploy): add Cloud SQL migration helper"
```

---

### Task 4: One-time bootstrap script (core GCP resources)

**Files:**
- Create: `deploy/bootstrap.sh`

- [ ] **Step 1: Write `deploy/bootstrap.sh`**

Implement creates that skip when resources already exist:

```bash
#!/usr/bin/env bash
# One-time greenfield bootstrap for Keweenaw hosted stack (no LB — see bootstrap-lb.sh).
set -euo pipefail

: "${PROJECT_ID:?Set PROJECT_ID}"
: "${REGION:=us-central1}"
: "${CLOUD_SQL_INSTANCE:=keweenaw-prod}"
: "${ARTIFACT_REPO:=keweenaw}"

gcloud config set project "${PROJECT_ID}"

echo "==> Enabling APIs"
gcloud services enable \
  artifactregistry.googleapis.com \
  run.googleapis.com \
  sqladmin.googleapis.com \
  secretmanager.googleapis.com \
  storage.googleapis.com \
  iam.googleapis.com \
  compute.googleapis.com \
  certificatemanager.googleapis.com \
  --project="${PROJECT_ID}"

echo "==> Artifact Registry"
gcloud artifacts repositories describe "${ARTIFACT_REPO}" --location="${REGION}" \
  || gcloud artifacts repositories create "${ARTIFACT_REPO}" \
       --repository-format=docker --location="${REGION}" \
       --description="Keweenaw Endurance container images"

echo "==> Service accounts"
for SA in keweenaw-backend keweenaw-frontend keweenaw-ci; do
  gcloud iam service-accounts describe "${SA}@${PROJECT_ID}.iam.gserviceaccount.com" \
    || gcloud iam service-accounts create "${SA}" --display-name="${SA}"
done

gcloud projects add-iam-policy-binding "${PROJECT_ID}" \
  --member="serviceAccount:keweenaw-backend@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role="roles/cloudsql.client" --condition=None || true

gcloud projects add-iam-policy-binding "${PROJECT_ID}" \
  --member="serviceAccount:keweenaw-ci@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role="roles/artifactregistry.writer" --condition=None || true
gcloud projects add-iam-policy-binding "${PROJECT_ID}" \
  --member="serviceAccount:keweenaw-ci@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role="roles/run.admin" --condition=None || true
gcloud iam service-accounts add-iam-policy-binding \
  "keweenaw-backend@${PROJECT_ID}.iam.gserviceaccount.com" \
  --member="serviceAccount:keweenaw-ci@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role="roles/iam.serviceAccountUser" || true
gcloud iam service-accounts add-iam-policy-binding \
  "keweenaw-frontend@${PROJECT_ID}.iam.gserviceaccount.com" \
  --member="serviceAccount:keweenaw-ci@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role="roles/iam.serviceAccountUser" || true

echo "==> Cloud SQL (may take several minutes)"
if ! gcloud sql instances describe "${CLOUD_SQL_INSTANCE}" >/dev/null 2>&1; then
  gcloud sql instances create "${CLOUD_SQL_INSTANCE}" \
    --database-version=POSTGRES_14 \
    --tier=db-custom-2-7680 \
    --region="${REGION}" \
    --storage-auto-increase \
    --backup-start-time=04:00 \
    --enable-point-in-time-recovery
fi

gcloud sql databases describe keweenaw_timing --instance="${CLOUD_SQL_INSTANCE}" \
  || gcloud sql databases create keweenaw_timing --instance="${CLOUD_SQL_INSTANCE}"

echo "==> Secrets"
ensure_secret() {
  local name="$1" value="$2"
  if gcloud secrets describe "${name}" >/dev/null 2>&1; then
    echo "Secret ${name} already exists — skipping create"
  else
    printf '%s' "${value}" | gcloud secrets create "${name}" --data-file=-
  fi
  gcloud secrets add-iam-policy-binding "${name}" \
    --member="serviceAccount:keweenaw-backend@${PROJECT_ID}.iam.gserviceaccount.com" \
    --role="roles/secretmanager.secretAccessor" || true
}

DB_PASSWORD="${DB_PASSWORD:-$(openssl rand -base64 32)}"
JWT_SECRET="${JWT_SECRET:-$(openssl rand -hex 32)}"
ORGANIZER_PIN="${ORGANIZER_PIN:-1738}"
BRIDGE_TOKEN="${BRIDGE_TOKEN:-$(openssl rand -hex 32)}"

ensure_secret keweenaw-db-password "${DB_PASSWORD}"
ensure_secret keweenaw-jwt-secret "${JWT_SECRET}"
ensure_secret keweenaw-organizer-pin "${ORGANIZER_PIN}"
ensure_secret keweenaw-bridge-token "${BRIDGE_TOKEN}"

gcloud sql users create timing_user --instance="${CLOUD_SQL_INSTANCE}" \
  --password="${DB_PASSWORD}" 2>/dev/null \
  || gcloud sql users set-password timing_user --instance="${CLOUD_SQL_INSTANCE}" \
       --password="${DB_PASSWORD}"

echo "==> GCS live CSV bucket"
BUCKET="${PROJECT_ID}-keweenaw-live-csv"
gsutil ls -b "gs://${BUCKET}" >/dev/null 2>&1 || gsutil mb -l "${REGION}" "gs://${BUCKET}"
gsutil iam ch \
  "serviceAccount:keweenaw-backend@${PROJECT_ID}.iam.gserviceaccount.com:objectAdmin" \
  "gs://${BUCKET}" || true

echo "==> CI key (create once; do NOT commit)"
KEY_FILE="${CI_KEY_FILE:-${TMPDIR:-/tmp}/keweenaw-ci-key.json}"
if [[ ! -f "${KEY_FILE}" ]]; then
  gcloud iam service-accounts keys create "${KEY_FILE}" \
    --iam-account="keweenaw-ci@${PROJECT_ID}.iam.gserviceaccount.com"
fi

PROJECT_NUMBER="$(gcloud projects describe "${PROJECT_ID}" --format='value(projectNumber)')"
echo ""
echo "=== Bootstrap core complete ==="
echo "PROJECT_ID=${PROJECT_ID}"
echo "PROJECT_NUMBER=${PROJECT_NUMBER}"
echo "REGION=${REGION}"
echo "CI key file: ${KEY_FILE}"
echo "Add GitHub secrets: GCP_SA_KEY (contents of key file), GCP_PROJECT_ID, GCP_REGION=${REGION}"
echo "BRIDGE_TOKEN (save for laptop): ${BRIDGE_TOKEN}"
echo "Next: initial image push + deploy/deploy-cloud-run.sh,"
echo "      then deploy/migrate-cloud-sql.sh, then deploy/bootstrap-lb.sh"
echo "NOTE: default ORGANIZER_PIN is 1738 unless ORGANIZER_PIN was set — rotate for real events."
```

- [ ] **Step 2: chmod + commit**

```bash
chmod +x deploy/bootstrap.sh
git add deploy/bootstrap.sh
git commit -m "chore(deploy): add greenfield GCP bootstrap script"
```

---

### Task 5: HTTPS load balancer bootstrap

**Files:**
- Create: `deploy/bootstrap-lb.sh`

- [ ] **Step 1: Write LB script**

```bash
#!/usr/bin/env bash
# Create HTTPS LB + managed cert for keweenawendurance.com (/api/* → backend, /* → frontend).
set -euo pipefail

: "${PROJECT_ID:?}"
: "${REGION:=us-central1}"
: "${DOMAIN:=keweenawendurance.com}"
: "${WWW_DOMAIN:=www.keweenawendurance.com}"

gcloud config set project "${PROJECT_ID}"

gcloud compute network-endpoint-groups describe keweenaw-backend-neg --region="${REGION}" \
  || gcloud compute network-endpoint-groups create keweenaw-backend-neg \
       --region="${REGION}" --network-endpoint-type=serverless \
       --cloud-run-service=keweenaw-backend

gcloud compute network-endpoint-groups describe keweenaw-frontend-neg --region="${REGION}" \
  || gcloud compute network-endpoint-groups create keweenaw-frontend-neg \
       --region="${REGION}" --network-endpoint-type=serverless \
       --cloud-run-service=keweenaw-frontend

gcloud compute backend-services describe keweenaw-backend-bes --global \
  || gcloud compute backend-services create keweenaw-backend-bes \
       --global --load-balancing-scheme=EXTERNAL_MANAGED \
       --protocol=HTTP
gcloud compute backend-services add-backend keweenaw-backend-bes --global \
  --network-endpoint-group=keweenaw-backend-neg \
  --network-endpoint-group-region="${REGION}" 2>/dev/null || true

gcloud compute backend-services describe keweenaw-frontend-bes --global \
  || gcloud compute backend-services create keweenaw-frontend-bes \
       --global --load-balancing-scheme=EXTERNAL_MANAGED \
       --protocol=HTTP
gcloud compute backend-services add-backend keweenaw-frontend-bes --global \
  --network-endpoint-group=keweenaw-frontend-neg \
  --network-endpoint-group-region="${REGION}" 2>/dev/null || true

if ! gcloud compute url-maps describe keweenaw-url-map >/dev/null 2>&1; then
  gcloud compute url-maps create keweenaw-url-map \
    --default-service=keweenaw-frontend-bes
fi
# Verify flags against: gcloud compute url-maps add-path-matcher --help
gcloud compute url-maps add-path-matcher keweenaw-url-map \
  --path-matcher-name=keweenaw-paths \
  --default-service=keweenaw-frontend-bes \
  --backend-service-path-rules="/api/*=keweenaw-backend-bes" \
  --new-hosts="${DOMAIN},${WWW_DOMAIN}" 2>/dev/null || true

gcloud compute ssl-certificates describe keweenaw-cert --global \
  || gcloud compute ssl-certificates create keweenaw-cert \
       --domains="${DOMAIN},${WWW_DOMAIN}" --global

gcloud compute target-https-proxies describe keweenaw-https-proxy \
  || gcloud compute target-https-proxies create keweenaw-https-proxy \
       --ssl-certificates=keweenaw-cert --url-map=keweenaw-url-map

gcloud compute addresses describe keweenaw-lb-ip --global \
  || gcloud compute addresses create keweenaw-lb-ip --global

IP="$(gcloud compute addresses describe keweenaw-lb-ip --global --format='value(address)')"

gcloud compute forwarding-rules describe keweenaw-https-rule --global \
  || gcloud compute forwarding-rules create keweenaw-https-rule \
       --global --address=keweenaw-lb-ip --target-https-proxy=keweenaw-https-proxy \
       --ports=443 --load-balancing-scheme=EXTERNAL_MANAGED

gcloud run services add-iam-policy-binding keweenaw-backend \
  --region="${REGION}" --member="allUsers" --role="roles/run.invoker" || true
gcloud run services add-iam-policy-binding keweenaw-frontend \
  --region="${REGION}" --member="allUsers" --role="roles/run.invoker" || true

echo ""
echo "=== LB bootstrap complete ==="
echo "Static IP: ${IP}"
echo "Create DNS A records:"
echo "  ${DOMAIN}     → ${IP}"
echo "  ${WWW_DOMAIN} → ${IP}"
echo "Wait until ssl cert managed.status == ACTIVE:"
echo "  gcloud compute ssl-certificates describe keweenaw-cert --global --format='value(managed.status)'"
```

When implementing, verify `add-path-matcher` flags against current gcloud help and adjust if needed. Prefer consistent `EXTERNAL_MANAGED` across address, forwarding rule, and backend services.

- [ ] **Step 2: chmod + commit**

```bash
chmod +x deploy/bootstrap-lb.sh
git add deploy/bootstrap-lb.sh
git commit -m "chore(deploy): add HTTPS load balancer bootstrap"
```

---

### Task 6: GitHub Actions `[deploy]` job

**Files:**
- Modify: `.github/workflows/ci.yml`

- [ ] **Step 1: Extend `on:` with workflow_dispatch**

```yaml
on:
  push:
    branches: [main, master]
  pull_request:
    branches: [main, master]
  workflow_dispatch:
    inputs:
      deploy:
        description: 'Set to "true" to deploy after CI'
        required: false
        default: 'false'
```

- [ ] **Step 2: Add deploy job after existing `test` and `e2e` jobs**

```yaml
  # CD runs only when the tip commit message contains [deploy]
  # (for merges, put [deploy] on the merge commit message).
  deploy:
    name: Deploy to Cloud Run
    needs: [test, e2e]
    if: >
      (github.ref == 'refs/heads/main' || github.ref == 'refs/heads/master') &&
      (
        (github.event_name == 'push' && contains(github.event.head_commit.message, '[deploy]')) ||
        (github.event_name == 'workflow_dispatch' && github.event.inputs.deploy == 'true')
      )
    runs-on: ubuntu-latest
    permissions:
      contents: read
    env:
      PROJECT_ID: ${{ secrets.GCP_PROJECT_ID }}
      REGION: ${{ secrets.GCP_REGION }}
      CLOUD_SQL_INSTANCE: keweenaw-prod
      IMAGE_TAG: ${{ github.sha }}
    steps:
      - uses: actions/checkout@v4

      - id: auth
        uses: google-github-actions/auth@v2
        with:
          credentials_json: ${{ secrets.GCP_SA_KEY }}

      - uses: google-github-actions/setup-gcloud@v2

      - name: Default region
        run: |
          if [ -z "${REGION}" ]; then
            echo "REGION=us-central1" >> "$GITHUB_ENV"
          fi

      - name: Configure Docker for Artifact Registry
        run: gcloud auth configure-docker "${REGION}-docker.pkg.dev" --quiet

      - name: Build and push backend
        run: |
          IMG="${REGION}-docker.pkg.dev/${PROJECT_ID}/keweenaw/backend"
          docker build -t "${IMG}:${IMAGE_TAG}" -t "${IMG}:latest" ./backend
          docker push "${IMG}:${IMAGE_TAG}"
          docker push "${IMG}:latest"

      - name: Build and push frontend
        run: |
          IMG="${REGION}-docker.pkg.dev/${PROJECT_ID}/keweenaw/frontend"
          docker build -t "${IMG}:${IMAGE_TAG}" -t "${IMG}:latest" ./frontend
          docker push "${IMG}:${IMAGE_TAG}"
          docker push "${IMG}:latest"

      - name: Deploy Cloud Run services
        run: |
          export PROJECT_ID REGION CLOUD_SQL_INSTANCE IMAGE_TAG
          bash deploy/deploy-cloud-run.sh
```

- [ ] **Step 3: Validate YAML**

```bash
actionlint .github/workflows/ci.yml
```

If `actionlint` is unavailable, visually confirm indentation and that `if:` / `needs:` are valid.

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: add [deploy]-gated Cloud Run deploy job"
```

---

### Task 7: Update deploy README checklist

**Files:**
- Modify: `deploy/README.md`

- [ ] **Step 1: Add CI/CD + greenfield bootstrap section** (near top, after architecture)

Operator sequence:

1. Create GCP project + enable billing.
2. `gcloud auth login` and `export PROJECT_ID=...`
3. `bash deploy/bootstrap.sh` — save printed `BRIDGE_TOKEN` and CI key path.
4. Initial images + Cloud Run:

```bash
export REGION=us-central1 IMAGE_TAG=bootstrap
# docker build/push backend+frontend to Artifact Registry, then:
bash deploy/deploy-cloud-run.sh
```

5. `DB_PASSWORD=... bash deploy/migrate-cloud-sql.sh`
6. `bash deploy/bootstrap-lb.sh` → set DNS A records to printed IP.
7. GitHub → Settings → Secrets: `GCP_SA_KEY`, `GCP_PROJECT_ID`, `GCP_REGION`.
8. Wait for managed cert `ACTIVE`.
9. Push to `main` with `[deploy]` in the commit message.

Windows: run bash scripts via Git Bash or WSL; need `gcloud` + Docker Desktop.

Mark existing Cloud Build section as an optional alternative to Actions image builds. Keep bridge laptop + rollback sections.

- [ ] **Step 2: Commit**

```bash
git add deploy/README.md
git commit -m "docs(deploy): document [deploy] CD and bootstrap sequence"
```

---

### Task 8: Operator handoff — credentials + first deploy (manual)

**Files:** none (runtime)

Executed with the user once scripts exist.

- [ ] **Step 1: Collect** — `PROJECT_ID`, billing on, PIN preference, DNS access ready
- [ ] **Step 2: Run** `bash deploy/bootstrap.sh`
- [ ] **Step 3: Add GitHub secrets** `GCP_SA_KEY`, `GCP_PROJECT_ID`, `GCP_REGION`
- [ ] **Step 4: Initial image + migrate + LB + DNS**
- [ ] **Step 5: Verify** frontend origin + Cloud Run `/health` URLs
- [ ] **Step 6: Smoke** a `[deploy]` commit on `main` and confirm Actions updates revisions

---

## Self-review (spec coverage)

| Spec requirement | Task |
|------------------|------|
| CI unchanged in purpose | Task 6 |
| CD only with `[deploy]` | Task 6 |
| SA JSON key auth | Tasks 4, 6, 8 |
| Bootstrap APIs/AR/SQL/secrets/GCS/SAs | Task 4 |
| Cloud Run templated deploy | Tasks 1–2 |
| Migrations helper | Task 3 |
| HTTPS LB + DNS | Tasks 5, 7 |
| Prod-only / no WIF | Honored |
| README checklist | Task 7 |
| Operator credentials | Task 8 |

---

## Execution handoff

Plan complete and saved to `docs/superpowers/plans/2026-07-19-cicd-gcp-deploy.md`.

**Two execution options:**

1. **Subagent-Driven (recommended)** — fresh subagent per task, review between tasks
2. **Inline Execution** — execute tasks in this session with checkpoints

Which approach?
