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
DB_PASSWORD_SECRET_NEW=0
DB_PASSWORD_PROVIDED=0
if [[ -n "${DB_PASSWORD+x}" ]]; then
  DB_PASSWORD_PROVIDED=1
else
  DB_PASSWORD="$(openssl rand -base64 32)"
fi

ensure_secret() {
  local name="$1" value="$2"
  local newly_created=0
  if gcloud secrets describe "${name}" >/dev/null 2>&1; then
    echo "Secret ${name} already exists — skipping create"
  else
    printf '%s' "${value}" | gcloud secrets create "${name}" --data-file=-
    newly_created=1
  fi
  if [[ "${name}" == "keweenaw-db-password" ]]; then
    DB_PASSWORD_SECRET_NEW=${newly_created}
  fi
  gcloud secrets add-iam-policy-binding "${name}" \
    --member="serviceAccount:keweenaw-backend@${PROJECT_ID}.iam.gserviceaccount.com" \
    --role="roles/secretmanager.secretAccessor" || true
}

JWT_SECRET="${JWT_SECRET:-$(openssl rand -hex 32)}"
ORGANIZER_PIN="${ORGANIZER_PIN:-1738}"
BRIDGE_TOKEN="${BRIDGE_TOKEN:-$(openssl rand -hex 32)}"

ensure_secret keweenaw-db-password "${DB_PASSWORD}"
ensure_secret keweenaw-jwt-secret "${JWT_SECRET}"
ensure_secret keweenaw-organizer-pin "${ORGANIZER_PIN}"
ensure_secret keweenaw-bridge-token "${BRIDGE_TOKEN}"

if [[ ${DB_PASSWORD_SECRET_NEW} -eq 1 || ${DB_PASSWORD_PROVIDED} -eq 1 ]]; then
  gcloud sql users create timing_user --instance="${CLOUD_SQL_INSTANCE}" \
    --password="${DB_PASSWORD}" 2>/dev/null \
    || gcloud sql users set-password timing_user --instance="${CLOUD_SQL_INSTANCE}" \
         --password="${DB_PASSWORD}"
else
  echo "NOTE: keweenaw-db-password secret already existed and DB_PASSWORD was not set — keeping existing DB password unchanged."
fi

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
