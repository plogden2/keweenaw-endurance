# CI/CD + Google Cloud production deploy — design

**Date:** 2026-07-19  
**Status:** Approved for implementation planning  
**Related:** [GCP prod deploy + dress rehearsal](./2026-07-16-gcp-prod-deploy-and-dress-rehearsal-design.md), `deploy/README.md`

## Goal

Make the repo **ready to deploy** to a greenfield Google Cloud project at `keweenawendurance.com`, with:

1. Existing GitHub Actions CI unchanged in purpose (tests + e2e).
2. Automatic **CD only when the commit message contains `[deploy]`** (push to `main` after CI passes).
3. One-time bootstrap for Cloud Run, Cloud SQL, Artifact Registry, Secret Manager, GCS, and HTTPS load balancer.
4. GitHub Actions authenticated via a **service account JSON key** (`GCP_SA_KEY`).

## Decisions (locked)

| Topic | Choice |
|-------|--------|
| GCP project | Greenfield — create project, then bootstrap |
| Environments | Production only |
| CD gate | Commit message must include `[deploy]` |
| Auth to GCP | Service account JSON key in GitHub secrets |
| Domain | Registered; include global HTTPS LB + managed cert |
| Pipeline owner | GitHub Actions builds images, pushes to Artifact Registry, deploys Cloud Run |
| Region default | `us-central1` (overridable via `GCP_REGION`) |

## §1 — Pipeline flow

### CI (every PR + every push to `main`)

Keep current `.github/workflows/ci.yml` behavior:

- Backend unit tests via `docker-compose.test.yml`
- 100% coverage fail-under for RFID scan packages
- Frontend tests
- Playwright e2e against Compose stack with mock RFID

CI never deploys.

### CD (push to `main` only)

A `deploy` job that:

1. Runs **after** CI succeeds (same workflow or a sibling that `needs` the test jobs).
2. Runs **only if** `contains(github.event.head_commit.message, '[deploy]')`.
3. Authenticates with GitHub secrets: `GCP_SA_KEY`, `GCP_PROJECT_ID`, `GCP_REGION`.
4. Builds and pushes `backend` + `frontend` images to Artifact Registry, tagged with the git SHA and `latest`.
5. Deploys both Cloud Run services to that SHA (substituting placeholders in existing `deploy/cloud-run-*.yaml` or equivalent `gcloud run deploy` / `services replace`).
6. Does **not** recreate Cloud SQL, load balancer, or secrets on every deploy.

**Non-triggers:** PRs never deploy. Pushes without `[deploy]` never deploy.

**Escape hatch:** Optional `workflow_dispatch` that deploys the selected ref (same auth and steps), for emergency / missed tag cases.

## §2 — One-time GCP bootstrap

Run once after the project exists and credentials are available. Prefer a `deploy/bootstrap.sh` (document Windows/`gcloud` equivalents in `deploy/README.md`).

Bootstrap creates/configures:

1. **APIs** — Artifact Registry, Cloud Run, Cloud SQL Admin, Secret Manager, Storage, IAM, Compute, Certificate Manager (as required for managed certs on the LB).
2. **Artifact Registry** — Docker repo `keweenaw` in `$REGION`.
3. **Service accounts**
   - `keweenaw-backend` — `roles/cloudsql.client`, Secret Manager accessor on app secrets, GCS objectAdmin on live-CSV bucket.
   - `keweenaw-frontend` — Cloud Run runtime only.
   - `keweenaw-ci` — used by GitHub Actions (JSON key → `GCP_SA_KEY`): push to Artifact Registry, deploy/update Cloud Run services, minimal IAM for those actions.
4. **Cloud SQL** — Postgres 14 instance `keweenaw-prod`, database `keweenaw_timing`, user `timing_user`, automated backups + PITR enabled.
5. **Secret Manager** — `keweenaw-db-password`, `keweenaw-jwt-secret`, `keweenaw-organizer-pin`, `keweenaw-bridge-token` (generate or prompt; never commit values).
6. **GCS** — `${PROJECT_ID}-keweenaw-live-csv` for hosted live CSV mirror.
7. **Cloud Run** — initial `keweenaw-backend` + `keweenaw-frontend` from existing YAML templates with placeholders filled (`PROJECT_ID`, `PROJECT_NUMBER`, `REGION`, instance name, image tags).
8. **Migrations** — apply `database/init` + `database/migrations` via Cloud SQL Auth Proxy. Production seed is **explicit/optional**, not automatic on every bootstrap.

Reuse and extend the existing kit in `deploy/` (`cloudbuild.yaml` may remain for manual `gcloud builds submit`; CD path is GitHub Actions).

## §3 — Domain + HTTPS load balancer

Single production origin:

| Host / path | Target |
|-------------|--------|
| `keweenawendurance.com` + `www` | Global HTTPS LB |
| `/api/*` | Serverless NEG → `keweenaw-backend` |
| `/*` | Serverless NEG → `keweenaw-frontend` |

Requirements:

- Google-managed certificate for `keweenawendurance.com` and `www.keweenawendurance.com`
- HTTP → HTTPS redirect
- WebSocket-friendly path for device-bridge (`/api/rfid/bridge`); Cloud Run backend already uses long `timeoutSeconds` and session affinity
- Backend env `CORS_ORIGINS=https://keweenawendurance.com`
- Bootstrap prints DNS records (A/AAAA or LB IPs) for the operator to set at the registrar
- After cert is ACTIVE, production URL for UI, API, and bridge is `https://keweenawendurance.com`

## §4 — Secrets, rollback, out of scope

### GitHub secrets (after bootstrap)

| Secret | Purpose |
|--------|---------|
| `GCP_SA_KEY` | JSON key for `keweenaw-ci` |
| `GCP_PROJECT_ID` | GCP project id |
| `GCP_REGION` | Deploy region (default `us-central1`) |

App runtime secrets stay in **Secret Manager only** (not mirrored to GitHub).

### Rollback

- **App:** route traffic to a previous Cloud Run revision, or redeploy a known-good Artifact Registry SHA.
- **Data:** Cloud SQL PITR / backup restore — not emergency CSV import for normal deploys.
- Document in `deploy/README.md`.

### Out of scope

- Staging environment
- Workload Identity Federation
- Device-bridge laptop install (already documented separately)
- Product/feature changes unrelated to deploy

## Deliverables

| Item | Purpose |
|------|---------|
| Update `.github/workflows/ci.yml` (or add deploy workflow) | Gated `[deploy]` job after CI |
| `deploy/bootstrap.sh` (+ README notes for Windows) | One-time GCP provision |
| LB / DNS bootstrap in `deploy/` | HTTPS LB, NEGs, managed cert |
| Deploy script / templating | Substitute project/region/SHA into Cloud Run manifests |
| Updated `deploy/README.md` | Checklist: project → bootstrap → GitHub secrets → DNS → first `[deploy]` |

## Operator sequence

1. Create GCP project; enable billing.
2. Create a temporary owner/user credentials (or run bootstrap with user `gcloud` auth).
3. Run bootstrap; note printed DNS records and generated secrets (especially bridge token).
4. Add `GCP_SA_KEY`, `GCP_PROJECT_ID`, `GCP_REGION` to GitHub repo secrets.
5. Point domain DNS at the load balancer; wait for managed cert ACTIVE.
6. Push to `main` with `[deploy]` in the commit message (after CI green).
7. Configure reader laptop device-bridge with `HOSTED_API_URL=https://keweenawendurance.com` and matching `BRIDGE_TOKEN`.

## Success criteria

- [ ] Push to `main` without `[deploy]` runs CI only.
- [ ] Push to `main` with `[deploy]` runs CI then builds, pushes, and updates both Cloud Run services.
- [ ] `https://keweenawendurance.com` serves the frontend; a known `/api/*` route reaches the backend; Cloud Run service URL `/health` is healthy on both services.
- [ ] Secrets are not in the repo; CI key is only in GitHub secrets.
- [ ] Bootstrap is idempotent enough to re-run safely for missing pieces, or clearly documents “create once” vs “update” steps.
`)