# Release Branch Workflow

This fork keeps the upstream core and the deployed product line separate.

## Branch roles

- `main` mirrors `QuantumNous/new-api` and must not contain fork-specific changes.
- `release` is the default branch and contains the deployed AIMPowered changes.
- `feature/*` branches merge into `release` through pull requests.
- Upstream updates are brought into `release` with a `main` to `release` pull request.

## Updating the upstream mirror

Use GitHub's **Sync fork** action for `main`, or update it locally:

```bash
git fetch upstream main
git switch main
git reset --hard upstream/main
git push origin main --force-with-lease
```

After `main` is current, open a pull request with:

- base: `release`
- compare: `main`

Resolve conflicts on a temporary branch rather than committing conflict fixes directly to either long-lived branch.

## GitHub Actions

- `Release CI` validates backend, relaykit, frontend, and container builds for every pull request into `release` and every push to `release`.
- `Publish release image` runs only after a successful `release` push CI and publishes `ghcr.io/boompieman/new-api:release` plus an immutable commit tag.
- `Deploy release to Cloud Run` mirrors the GHCR image into Artifact Registry and deploys it after image publication.

The Cloud Run deployment remains safely skipped until the `production` environment contains these secrets:

- `GCP_WORKLOAD_IDENTITY_PROVIDER`
- `GCP_DEPLOY_SERVICE_ACCOUNT`

Optional repository variables override the deployment defaults:

- `GCP_PROJECT_ID` (`aimpowered`)
- `GCP_REGION` (`asia-east1`)
- `GCP_CLOUD_RUN_SERVICE` (`aimpowered-newapi`)
- `GCP_ARTIFACT_REPOSITORY` (`cloud-run-images`)

