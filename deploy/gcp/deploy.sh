#!/usr/bin/env bash

set -euo pipefail

PROJECT_ID="${PROJECT_ID:-aimpowered}"
REGION="${REGION:-asia-east1}"
SERVICE_NAME="${SERVICE_NAME:-aimpowered-newapi}"
SQL_INSTANCE="${SQL_INSTANCE:-aimpowered-newapi-db}"
SQL_DATABASE="${SQL_DATABASE:-new_api}"
SQL_USER="${SQL_USER:-new_api_app}"
SERVICE_ACCOUNT_NAME="${SERVICE_ACCOUNT_NAME:-aimpowered-newapi}"
ARTIFACT_REPOSITORY="${ARTIFACT_REPOSITORY:-cloud-run-images}"
RELEASE_VERSION="${RELEASE_VERSION:-v1.0.0-rc.24}"
UPSTREAM_IMAGE="calciumion/new-api:${RELEASE_VERSION}"
TARGET_IMAGE_REPOSITORY="${REGION}-docker.pkg.dev/${PROJECT_ID}/${ARTIFACT_REPOSITORY}/${SERVICE_NAME}"
TARGET_IMAGE="${TARGET_IMAGE_REPOSITORY}:${RELEASE_VERSION}"
SERVICE_ACCOUNT="${SERVICE_ACCOUNT_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"

DB_ADMIN_PASSWORD_SECRET="${SERVICE_NAME}-db-admin-password"
DB_APP_PASSWORD_SECRET="${SERVICE_NAME}-db-app-password"
SQL_DSN_SECRET="${SERVICE_NAME}-sql-dsn"
SESSION_SECRET_NAME="${SERVICE_NAME}-session-secret"
ADMIN_PASSWORD_SECRET="${SERVICE_NAME}-admin-password"

require_command() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Required command is missing: $1" >&2
    exit 1
  }
}

secret_exists() {
  gcloud secrets describe "$1" --project="$PROJECT_ID" >/dev/null 2>&1
}

create_generated_secret() {
  local secret_name="$1"
  local byte_count="$2"

  if secret_exists "$secret_name"; then
    return
  fi

  openssl rand -hex "$byte_count" \
    | gcloud secrets create "$secret_name" \
        --project="$PROJECT_ID" \
        --replication-policy=automatic \
        --data-file=- \
        --quiet >/dev/null
}

require_command gcloud
require_command docker
require_command curl
require_command jq
require_command openssl

echo "Enabling required Google Cloud APIs..."
gcloud services enable \
  artifactregistry.googleapis.com \
  run.googleapis.com \
  sqladmin.googleapis.com \
  secretmanager.googleapis.com \
  --project="$PROJECT_ID" \
  --quiet

if ! gcloud iam service-accounts describe "$SERVICE_ACCOUNT" --project="$PROJECT_ID" >/dev/null 2>&1; then
  echo "Creating dedicated runtime service account..."
  gcloud iam service-accounts create "$SERVICE_ACCOUNT_NAME" \
    --project="$PROJECT_ID" \
    --display-name="AIMPowered New API runtime" \
    --quiet
fi

gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:${SERVICE_ACCOUNT}" \
  --role="roles/cloudsql.client" \
  --condition=None \
  --quiet >/dev/null

create_generated_secret "$DB_ADMIN_PASSWORD_SECRET" 32
create_generated_secret "$DB_APP_PASSWORD_SECRET" 32
create_generated_secret "$SESSION_SECRET_NAME" 64
create_generated_secret "$ADMIN_PASSWORD_SECRET" 24

DB_ADMIN_PASSWORD="$(gcloud secrets versions access latest --secret="$DB_ADMIN_PASSWORD_SECRET" --project="$PROJECT_ID")"
DB_APP_PASSWORD="$(gcloud secrets versions access latest --secret="$DB_APP_PASSWORD_SECRET" --project="$PROJECT_ID")"

if ! gcloud sql instances describe "$SQL_INSTANCE" --project="$PROJECT_ID" >/dev/null 2>&1; then
  echo "Creating development-sized PostgreSQL instance..."
  gcloud sql instances create "$SQL_INSTANCE" \
    --project="$PROJECT_ID" \
    --database-version=POSTGRES_15 \
    --region="$REGION" \
    --tier=db-f1-micro \
    --availability-type=zonal \
    --storage-type=SSD \
    --storage-size=10 \
    --storage-auto-increase \
    --backup-start-time=18:00 \
    --deletion-protection \
    --root-password="$DB_ADMIN_PASSWORD" \
    --quiet
fi

if ! gcloud sql databases describe "$SQL_DATABASE" --instance="$SQL_INSTANCE" --project="$PROJECT_ID" >/dev/null 2>&1; then
  echo "Creating application database..."
  gcloud sql databases create "$SQL_DATABASE" \
    --instance="$SQL_INSTANCE" \
    --project="$PROJECT_ID" \
    --quiet
fi

if ! gcloud sql users list --instance="$SQL_INSTANCE" --project="$PROJECT_ID" --format='value(name)' | grep -Fxq "$SQL_USER"; then
  echo "Creating application database user..."
  gcloud sql users create "$SQL_USER" \
    --instance="$SQL_INSTANCE" \
    --project="$PROJECT_ID" \
    --password="$DB_APP_PASSWORD" \
    --quiet
fi

CONNECTION_NAME="$(gcloud sql instances describe "$SQL_INSTANCE" --project="$PROJECT_ID" --format='value(connectionName)')"
SQL_DSN="postgresql://${SQL_USER}:${DB_APP_PASSWORD}@/${SQL_DATABASE}?host=/cloudsql/${CONNECTION_NAME}&sslmode=disable"

if ! secret_exists "$SQL_DSN_SECRET"; then
  printf '%s' "$SQL_DSN" \
    | gcloud secrets create "$SQL_DSN_SECRET" \
        --project="$PROJECT_ID" \
        --replication-policy=automatic \
        --data-file=- \
        --quiet >/dev/null
fi

for runtime_secret in "$SQL_DSN_SECRET" "$SESSION_SECRET_NAME"; do
  gcloud secrets add-iam-policy-binding "$runtime_secret" \
    --project="$PROJECT_ID" \
    --member="serviceAccount:${SERVICE_ACCOUNT}" \
    --role="roles/secretmanager.secretAccessor" \
    --quiet >/dev/null
done

echo "Mirroring the pinned upstream image into Artifact Registry..."
gcloud auth configure-docker "${REGION}-docker.pkg.dev" --quiet >/dev/null
docker pull --platform=linux/amd64 "$UPSTREAM_IMAGE"
docker tag "$UPSTREAM_IMAGE" "$TARGET_IMAGE"
docker push "$TARGET_IMAGE"

IMAGE_DIGEST="$(gcloud artifacts docker images describe "$TARGET_IMAGE" --project="$PROJECT_ID" --format='value(image_summary.digest)')"
DEPLOY_IMAGE="${TARGET_IMAGE_REPOSITORY}@${IMAGE_DIGEST}"

echo "Deploying privately for safe first-run initialization..."
gcloud run deploy "$SERVICE_NAME" \
  --project="$PROJECT_ID" \
  --region="$REGION" \
  --platform=managed \
  --image="$DEPLOY_IMAGE" \
  --service-account="$SERVICE_ACCOUNT" \
  --execution-environment=gen2 \
  --port=3000 \
  --cpu=1 \
  --memory=512Mi \
  --concurrency=20 \
  --min=0 \
  --max=2 \
  --timeout=300 \
  --ingress=all \
  --startup-probe="initialDelaySeconds=0,timeoutSeconds=5,periodSeconds=5,failureThreshold=24,httpGet.path=/api/status,httpGet.port=3000" \
  --liveness-probe="initialDelaySeconds=15,timeoutSeconds=5,periodSeconds=30,failureThreshold=3,httpGet.path=/api/status,httpGet.port=3000" \
  --set-cloudsql-instances="$CONNECTION_NAME" \
  --set-secrets="SQL_DSN=${SQL_DSN_SECRET}:latest,SESSION_SECRET=${SESSION_SECRET_NAME}:latest" \
  --set-env-vars="TZ=Asia/Taipei,NODE_NAME=${SERVICE_NAME},ERROR_LOG_ENABLED=true,BATCH_UPDATE_ENABLED=true,TRUSTED_PROXIES=169.254.0.0/16" \
  --no-allow-unauthenticated \
  --quiet

SERVICE_URL="$(gcloud run services describe "$SERVICE_NAME" --project="$PROJECT_ID" --region="$REGION" --format='value(status.url)')"
IDENTITY_TOKEN="$(gcloud auth print-identity-token)"
ADMIN_PASSWORD="$(gcloud secrets versions access latest --secret="$ADMIN_PASSWORD_SECRET" --project="$PROJECT_ID")"

echo "Waiting for the private health endpoint..."
for attempt in $(seq 1 30); do
  if curl --silent --fail \
      --header "Authorization: Bearer ${IDENTITY_TOKEN}" \
      "${SERVICE_URL}/api/status" >/dev/null; then
    break
  fi

  if [[ "$attempt" == "30" ]]; then
    echo "Service did not become healthy in time." >&2
    exit 1
  fi

  sleep 5
done

SETUP_STATUS="$(curl --silent --fail \
  --header "Authorization: Bearer ${IDENTITY_TOKEN}" \
  "${SERVICE_URL}/api/setup")"

if [[ "$(jq -r '.data.status' <<<"$SETUP_STATUS")" != "true" ]]; then
  echo "Creating the initial administrator while the service is private..."
  SETUP_PAYLOAD="$(jq -cn \
    --arg username "admin" \
    --arg password "$ADMIN_PASSWORD" \
    '{username:$username,password:$password,confirmPassword:$password,SelfUseModeEnabled:false,DemoSiteEnabled:false}')"

  SETUP_RESPONSE="$(curl --silent --fail \
    --request POST \
    --header "Authorization: Bearer ${IDENTITY_TOKEN}" \
    --header 'Content-Type: application/json' \
    --data "$SETUP_PAYLOAD" \
    "${SERVICE_URL}/api/setup")"

  jq -e '.success == true' <<<"$SETUP_RESPONSE" >/dev/null
fi

echo "Enabling secure cookies for the final HTTPS origin..."
gcloud run services update "$SERVICE_NAME" \
  --project="$PROJECT_ID" \
  --region="$REGION" \
  --update-env-vars="SESSION_COOKIE_SECURE=true,SESSION_COOKIE_TRUSTED_URL=${SERVICE_URL}" \
  --quiet >/dev/null

echo "Opening the initialized service to public traffic..."
gcloud run services add-iam-policy-binding "$SERVICE_NAME" \
  --project="$PROJECT_ID" \
  --region="$REGION" \
  --member=allUsers \
  --role=roles/run.invoker \
  --quiet >/dev/null

PUBLIC_STATUS="$(curl --silent --fail "${SERVICE_URL}/api/status")"
jq -e '.success == true' <<<"$PUBLIC_STATUS" >/dev/null

echo
echo "Deployment complete."
echo "Service URL: ${SERVICE_URL}"
echo "Admin username: admin"
echo "Admin password is stored in Secret Manager: ${ADMIN_PASSWORD_SECRET}"
echo "Retrieve it with: gcloud secrets versions access latest --secret=${ADMIN_PASSWORD_SECRET} --project=${PROJECT_ID}"
