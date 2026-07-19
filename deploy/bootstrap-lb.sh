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
       --protocol=HTTP --timeout=3600
gcloud compute backend-services update keweenaw-backend-bes --global --timeout=3600 || true
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
