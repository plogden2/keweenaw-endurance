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
