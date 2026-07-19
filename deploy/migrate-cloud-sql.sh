#!/usr/bin/env bash
# Apply database/init + database/migrations against Cloud SQL via Auth Proxy.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

: "${PROJECT_ID:?PROJECT_ID is required}"
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
