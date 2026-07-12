#!/bin/sh
# Fail CI if backend/internal/services/scan packages with tests are under 100% coverage.
# Skips when the package dir is missing or has no *_test.go files yet (T013a / FR-029).
set -eu

THRESHOLD="${COVERAGE_FAIL_UNDER:-100.0}"

# Prefer running with backend as cwd (Docker); fall back to repo-root layout.
if [ -d "./internal/services/scan" ]; then
  BACKEND_DIR="."
  SCAN_DIR="./internal/services/scan"
elif [ -d "./backend/internal/services/scan" ]; then
  BACKEND_DIR="./backend"
  SCAN_DIR="./backend/internal/services/scan"
else
  echo "coverage-fail-under: scan package not present yet — skipping"
  exit 0
fi

TEST_FILES="$(find "$SCAN_DIR" -type f -name '*_test.go' 2>/dev/null || true)"
if [ -z "$TEST_FILES" ]; then
  echo "coverage-fail-under: no tests under services/scan yet — skipping"
  exit 0
fi

PROFILE="$(mktemp)"
trap 'rm -f "$PROFILE"' EXIT

echo "coverage-fail-under: go test ./internal/services/scan/... (fail-under ${THRESHOLD}%)"
(
  cd "$BACKEND_DIR"
  go test ./internal/services/scan/... -coverprofile="$PROFILE" -covermode=atomic
)

PCT="$(go tool cover -func="$PROFILE" | awk '/^total:/ { gsub(/%/, "", $3); print $3 }')"
if [ -z "$PCT" ]; then
  echo "coverage-fail-under: could not parse coverage total"
  exit 1
fi

echo "coverage-fail-under: scan package coverage ${PCT}%"
awk -v p="$PCT" -v t="$THRESHOLD" 'BEGIN { if ((p + 0) < (t + 0)) exit 1; exit 0 }' || {
  echo "coverage-fail-under: coverage ${PCT}% is below ${THRESHOLD}%"
  exit 1
}

echo "coverage-fail-under: OK"
