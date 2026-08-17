# AIMPowered New API on Google Cloud

This deployment runs the upstream `QuantumNous/new-api` release on Cloud Run
with persistent PostgreSQL storage in Cloud SQL.

## Current sizing

- Region: `asia-east1` (Taiwan)
- Cloud Run: 1 vCPU, 512 MiB RAM, 0-2 instances, concurrency 20
- Cloud SQL: PostgreSQL 15, `db-f1-micro`, zonal, 10 GB SSD
- Secrets: Google Secret Manager
- Container: mirrored to Artifact Registry and deployed by immutable digest
- Public access: enabled only after the initial administrator is created
- Health checks: HTTP startup and liveness probes on `/api/status`
- Reverse proxy trust: restricted to Cloud Run's link-local proxy network

The database has automated backups, storage auto-growth, and deletion
protection. It is intentionally not highly available at this development-sized
stage.

## Deploy

```bash
./deploy/gcp/deploy.sh
```

The script is scoped to project `aimpowered` by default. All names and the
region can be overridden with environment variables documented at the top of
the script.

## Administrator password

The bootstrap password is never written to the repository or printed during
deployment. Retrieve it only when needed:

```bash
gcloud secrets versions access latest \
  --secret=aimpowered-newapi-admin-password \
  --project=aimpowered
```

Change the password after the first login and enable two-factor authentication.

## Before production scale-up

1. Upgrade Cloud SQL to a dedicated-core tier and regional HA.
2. Add Redis/Memorystore before increasing horizontal scale substantially.
3. Add a custom domain and include its exact HTTPS origin in
   `SESSION_COOKIE_TRUSTED_URL`.
4. Configure monitoring alerts, uptime checks, database recovery tests, and a
   documented incident response path.
5. Review the upstream AGPLv3 obligations and the regulatory requirements for
   operating a public generative-AI gateway.
