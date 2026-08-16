# Redeploy keweenawendurance.com after the 2026-08-16 shutdown

Production in GCP project `keweenaw-endurance` was fully torn down on **2026-08-16** so the monthly bill would be **$0**. Application data and secrets were saved locally. Use this playbook to bring the site back.

Canonical greenfield steps (APIs, SQL, LB, Cloud Run) are in [`deploy/README.md`](../deploy/README.md). This file covers **restore-from-backup** differences.

## Backup location

```
C:\Users\gener\Documents\keweenaw-endurance-backups\2026-08-16\
```

That folder is **not in git**. It contains the SQL dump and Secret Manager values.

## What was running when we shut down

| Resource | Name / value |
|----------|----------------|
| Project | `keweenaw-endurance` |
| Region | `us-central1` |
| Cloud SQL | `keweenaw-prod` (Postgres 14, `db-custom-2-7680`, public IP) |
| Database | `keweenaw_timing` / user `timing_user` |
| Cloud Run | `keweenaw-backend`, `keweenaw-frontend` (image tag **`0682ea2`**) |
| GCS | `keweenaw-endurance-keweenaw-live-csv` (empty except the shutdown dump) |
| Artifact Registry | `us-central1-docker.pkg.dev/keweenaw-endurance/keweenaw` |
| HTTPS LB | static IP `136.69.101.114` (`keweenaw-lb-ip`) |
| DNS | zone `keweenawendurance-com` — apex + `www` A → that IP |
| Domain | Cloud Domains `keweenawendurance.com`, paid through **2027-07-19** |
| Secrets | `keweenaw-db-password`, `keweenaw-jwt-secret`, `keweenaw-organizer-pin`, `keweenaw-bridge-token` |

The **domain registration was kept** (active until **2027-07-19**). Cloud DNS, the load balancer, Cloud Run, Cloud SQL, buckets, images, and secrets were deleted. **Billing is unlinked** (`billingEnabled: false`), which is what keeps the GCP monthly bill at $0.

Domain auto-renew could not be flipped after billing was removed (writes require billing). GCP cannot charge renewal with no billing account. If Squarespace also manages the name, confirm auto-renew is off there so you do not get a yearly invoice in 2027.

If the project has no billing account, re-link it first (console: Billing → Account management → My projects, or):

```bash
# billingAccounts/01E372-A52791-D6FEA2 was the account in use on 2026-08-16
curl -X PUT \
  -H "Authorization: Bearer $(gcloud auth print-access-token)" \
  -H "Content-Type: application/json" \
  -d '{"billingAccountName":"billingAccounts/01E372-A52791-D6FEA2"}' \
  "https://cloudbilling.googleapis.com/v1/projects/keweenaw-endurance/billingInfo"
```

## 1. Relink billing and set gcloud

```bash
export PROJECT_ID=keweenaw-endurance
export REGION=us-central1
export CLOUD_SQL_INSTANCE=keweenaw-prod
gcloud config set project "$PROJECT_ID"
gcloud auth login   # if needed; historically plogden2@gmail.com
```

Confirm billing is enabled before creating SQL or a load balancer.

## 2. Bootstrap infrastructure (do not generate new secrets yet)

From the repo root, in **Git Bash / WSL**:

```bash
export PROJECT_ID=keweenaw-endurance
export REGION=us-central1
bash deploy/bootstrap.sh
```

That recreates Artifact Registry, Cloud SQL (`keweenaw-prod` + `keweenaw_timing` + `timing_user`), GCS, and service accounts.

`bootstrap.sh` will **create new random secrets** if they do not exist. After it finishes, **overwrite them** with the shutdown backup so the reader laptop bridge token and organizer PIN still match:

```bash
BACKUP="/c/Users/gener/Documents/keweenaw-endurance-backups/2026-08-16"

# PowerShell equivalent also works; values must be the raw file bytes with no newline/BOM.
gcloud secrets versions add keweenaw-db-password --data-file="$BACKUP/secrets/keweenaw-db-password.txt"
gcloud secrets versions add keweenaw-jwt-secret --data-file="$BACKUP/secrets/keweenaw-jwt-secret.txt"
gcloud secrets versions add keweenaw-organizer-pin --data-file="$BACKUP/secrets/keweenaw-organizer-pin.txt"
gcloud secrets versions add keweenaw-bridge-token --data-file="$BACKUP/secrets/keweenaw-bridge-token.txt"
```

Then reset the Cloud SQL user password to the restored secret (bootstrap may have set a different one):

```bash
DB_PASSWORD=$(cat "$BACKUP/secrets/keweenaw-db-password.txt")
gcloud sql users set-password timing_user \
  --instance=keweenaw-prod \
  --password="$DB_PASSWORD"
```

Save the new CI key `bootstrap.sh` prints and put it in GitHub secret `GCP_SA_KEY` (the old key is gone).

## 3. Restore the database (do not re-seed)

Do **not** run `database/seed/*.sql` on top of this dump. Restore the dump, then skip `deploy/migrate-cloud-sql.sh` unless you are moving forward to a newer schema than 2026-08-16.

Using Cloud SQL import (upload dump to the live-csv bucket first):

```bash
BACKUP_WIN="C:\\Users\\gener\\Documents\\keweenaw-endurance-backups\\2026-08-16"
gsutil cp "$BACKUP_WIN\\sql\\keweenaw_timing-2026-08-16.sql.gz" \
  gs://keweenaw-endurance-keweenaw-live-csv/restore/keweenaw_timing.sql.gz

SQL_SA=$(gcloud sql instances describe keweenaw-prod --format='value(serviceAccountEmailAddress)')
gsutil iam ch "serviceAccount:${SQL_SA}:objectAdmin" gs://keweenaw-endurance-keweenaw-live-csv

gcloud sql import sql keweenaw-prod \
  gs://keweenaw-endurance-keweenaw-live-csv/restore/keweenaw_timing.sql.gz \
  --database=keweenaw_timing
```

If import complains the schema already exists, drop and recreate `keweenaw_timing` (empty) and import again:

```bash
gcloud sql databases delete keweenaw_timing --instance=keweenaw-prod --quiet
gcloud sql databases create keweenaw_timing --instance=keweenaw-prod
# then repeat import
```

## 4. Build images and deploy Cloud Run

Preferred: current `main`. To recreate the exact 2026-08-16 app, check out git SHA **`0682ea2`** first.

Windows operator steps: [`.cursor/skills/deploy-prod/SKILL.md`](../.cursor/skills/deploy-prod/SKILL.md).

```bash
export PROJECT_ID=keweenaw-endurance REGION=us-central1
export CLOUD_SQL_INSTANCE=keweenaw-prod
export IMAGE_TAG=$(git rev-parse --short HEAD)   # or 0682ea2

# Build/push backend + frontend to Artifact Registry, then:
bash deploy/deploy-cloud-run.sh
```

Allow unauthenticated invoke if the site is public:

```bash
gcloud run services add-iam-policy-binding keweenaw-backend \
  --region=us-central1 --member=allUsers --role=roles/run.invoker
gcloud run services add-iam-policy-binding keweenaw-frontend \
  --region=us-central1 --member=allUsers --role=roles/run.invoker
```

## 5. HTTPS load balancer + DNS

```bash
export PROJECT_ID=keweenaw-endurance REGION=us-central1
bash deploy/bootstrap-lb.sh
```

Note the **new static IP** (the old `136.69.101.114` was released). Recreate Cloud DNS:

```bash
gcloud dns managed-zones create keweenawendurance-com \
  --dns-name=keweenawendurance.com. \
  --description="DNS zone for domain: keweenawendurance.com"

# Point apex + www at the IP printed by bootstrap-lb.sh
NEW_IP=REPLACE_ME
gcloud dns record-sets create keweenawendurance.com. --zone=keweenawendurance-com --type=A --ttl=300 --rrdatas="$NEW_IP"
gcloud dns record-sets create www.keweenawendurance.com. --zone=keweenawendurance-com --type=A --ttl=300 --rrdatas="$NEW_IP"
```

If the domain still uses Google Cloud DNS name servers (`ns-cloud-e1` … `e4.googledomains.com`), you only need the A records. Wait until:

```bash
gcloud compute ssl-certificates describe keweenaw-cert --global --format='value(managed.status)'
# expect ACTIVE (or use keweenaw-cert-2 if bootstrap created the first cert before DNS was visible)
```

## 6. Smoke checks

```powershell
curl.exe -sS -o NUL -w "backend_health=%{http_code}`n" https://keweenaw-backend-opdxtbtb2q-uc.a.run.app/health
curl.exe -sS -o NUL -w "frontend=%{http_code}`n" https://keweenaw-frontend-opdxtbtb2q-uc.a.run.app/
curl.exe -sS -o NUL -w "prod_apex=%{http_code}`n" https://keweenawendurance.com/
curl.exe -sS -o NUL -w "prod_www=%{http_code}`n" https://www.keweenawendurance.com/
```

Reader laptop: `HOSTED_API_URL=https://keweenawendurance.com` and `BRIDGE_TOKEN` from the restored `keweenaw-bridge-token` secret. See `docs/reader-laptop-setup.md`.

## Cost note

The previous Cloud SQL tier `db-custom-2-7680` (2 vCPU / 7.5 GB) plus two global forwarding rules and `minScale: 1` on the backend is what drove the bill. For a cheaper return, change the SQL `--tier` in `deploy/bootstrap.sh` (for example `db-f1-micro` or `db-custom-1-3840`) and drop backend `minScale` to `0` in `deploy/cloud-run-backend.yaml` before deploying.
