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
