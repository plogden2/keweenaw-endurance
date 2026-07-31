---
name: deploy-prod
description: >-
  Redeploy keweenaw-endurance to GCP production (Artifact Registry + Cloud Run).
  Use when the user asks to deploy, redeploy, ship to prod, cut over Cloud Run,
  or push images for keweenawendurance.com.
---

# Deploy prod (keweenaw-endurance)

## Defaults

| Item | Value |
|------|-------|
| GCP project | `keweenaw-endurance` |
| Region | `us-central1` |
| Cloud SQL | `keweenaw-prod` |
| Registry | `us-central1-docker.pkg.dev/keweenaw-endurance/keweenaw` |
| Services | `keweenaw-backend`, `keweenaw-frontend` |
| Domain | `https://keweenawendurance.com` / `https://www.keweenawendurance.com` |
| Direct URLs | `https://keweenaw-backend-opdxtbtb2q-uc.a.run.app`, `https://keweenaw-frontend-opdxtbtb2q-uc.a.run.app` |

Canonical docs: `deploy/README.md`. This skill captures **operator learnings** that are easy to miss.

## When to use which path

1. **Preferred ongoing CD:** push to `main` with `[deploy]` in the commit message (or `workflow_dispatch` with `deploy=true`). Needs green `test` + `e2e` jobs first.
2. **Use manual cutover when:** Actions e2e flakes block CD, or you need a hotfix now. Prior prod ships used **local Docker build/push + `deploy/deploy-cloud-run.sh`**.
3. **Do not assume Cloud Build works.** `gcloud builds submit` failed with `cloudbuild.googleapis.com` **disabled** / `PERMISSION_DENIED`. Prefer local Docker unless the API is known-enabled. Do not enable APIs without asking.

## Manual redeploy (Windows)

Prereqs: `gcloud` authenticated (`plogden2@gmail.com` historically), Docker Desktop running, Git Bash at `C:\Program Files\Git\bin\bash.exe`.

```powershell
$env:PROJECT_ID = "keweenaw-endurance"
$env:REGION = "us-central1"
$IMAGE_TAG = git rev-parse --short HEAD

gcloud auth configure-docker "$env:REGION-docker.pkg.dev" --quiet

# Backend (COPY step can take several minutes — do not kill as hung)
$IMG = "$env:REGION-docker.pkg.dev/$env:PROJECT_ID/keweenaw/backend"
docker build -t "${IMG}:${IMAGE_TAG}" -t "${IMG}:latest" ./backend
docker push "${IMG}:${IMAGE_TAG}"
docker push "${IMG}:latest"

# Frontend
$IMG = "$env:REGION-docker.pkg.dev/$env:PROJECT_ID/keweenaw/frontend"
docker build -t "${IMG}:${IMAGE_TAG}" -t "${IMG}:latest" ./frontend
docker push "${IMG}:${IMAGE_TAG}"
docker push "${IMG}:latest"

# Apply Cloud Run (bash scripts; envsubst)
& "C:\Program Files\Git\bin\bash.exe" -lc 'export PROJECT_ID=keweenaw-endurance REGION=us-central1 CLOUD_SQL_INSTANCE=keweenaw-prod IMAGE_TAG=$(git rev-parse --short HEAD); bash deploy/deploy-cloud-run.sh'
```

### Windows pitfalls

- Run `deploy/*.sh` via **Git Bash**, not raw PowerShell.
- Under Git Bash, `gcloud` may print `python3: Permission denied` (WindowsApps stub). Deploy can still succeed; confirm with smoke checks. If gcloud truly fails, set `CLOUDSDK_PYTHON` to a real Python and retry, or run `gcloud run services replace` from PowerShell after `render-cloud-run.sh`.
- Backend image context transfer/COPY is slow on this machine; allow ~5–10+ minutes.
- Tag with **short SHA** (`git rev-parse --short HEAD`) to match existing Artifact Registry tags. CI uses full `github.sha` — either works if build and deploy use the **same** tag.

## Migrations / seed

- **App deploy ≠ DB migrate.** Cloud Run replace does not run SQL.
- Production schema often lands via **GORM AutoMigrate** on backend boot (e.g. bib-tag association). Read migration comments before assuming SQL must be applied.
- `deploy/migrate-cloud-sql.sh` re-runs `database/init` + all `database/migrations/*.sql`. **Do not run casually** on a live DB; only when intentional and you understand idempotency/risk.
- Seeds (e.g. `database/seed/03-bluffet-2026.sql`) are **not** part of redeploy unless the user asks.

## Smoke checks (required)

```powershell
curl.exe -sS -o NUL -w "backend_health=%{http_code}`n" https://keweenaw-backend-opdxtbtb2q-uc.a.run.app/health
curl.exe -sS -o NUL -w "frontend=%{http_code}`n" https://keweenaw-frontend-opdxtbtb2q-uc.a.run.app/
curl.exe -sS -o NUL -w "prod_apex=%{http_code}`n" https://keweenawendurance.com/
curl.exe -sS -o NUL -w "prod_www=%{http_code}`n" https://www.keweenawendurance.com/
curl.exe -sS -o NUL -w "results_xlsx=%{http_code}`n" "https://keweenaw-backend-opdxtbtb2q-uc.a.run.app/api/events/00000000-0000-0000-0000-000000000000/results.xlsx"
gcloud run services describe keweenaw-backend --region=us-central1 --format="value(spec.template.spec.containers[0].image)"
gcloud run services describe keweenaw-frontend --region=us-central1 --format="value(spec.template.spec.containers[0].image)"
```

Expect: health/frontend/domain `200`; `results.xlsx` `401` without PIN (route present); images end with the new `IMAGE_TAG`.

## Out of scope for app redeploy

- Reader laptop / device-bridge / Proxmark — native, not Cloud Run.
- Arming `PUT /api/stations/current` — separate race-day ops; a cold backend loses in-memory station binding.
- Enabling Cloud Build API, changing DNS/LB, rotating secrets — ask first.
- Committing local dumps, `frontend/test-results/`, RFID dump bins/json.

## Rollback

```bash
gcloud run revisions list --service=keweenaw-backend --region=us-central1
gcloud run services update-traffic keweenaw-backend --to-revisions=REVISION=100 --region=us-central1
# repeat for keweenaw-frontend, or redeploy previous IMAGE_TAG
```

Hosted data corruption → Cloud SQL PITR/backup, not CSV import (emergency only).

## Checklist

```
- [ ] On intended git SHA (usually origin/main)
- [ ] Docker + gcloud auth OK
- [ ] Backend + frontend images pushed with IMAGE_TAG
- [ ] deploy-cloud-run.sh applied both services
- [ ] Smoke checks green; image tags match
- [ ] Migrations only if explicitly needed
```
