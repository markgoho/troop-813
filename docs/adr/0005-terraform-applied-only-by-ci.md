# Terraform applied only by CI; path-filtered single-job workflows

Status: accepted (2026-11-08). Decided in [Deploy, CI and environments](https://github.com/markgoho/troop-813/issues/10). Builds on [ADR 0001](0001-stack-and-hosting.md).

## Context

The infra is small and should rarely change once built. The owner doesn't want to run, learn or babysit Terraform. In doula-cloud, a human applier plus overlapping deployers led to drift checks, an `applied_sha` stamp and image conflicts. The repo is private, so GitHub Actions minutes are capped (2,000/mo), and each job rounds up to a whole minute.

## Decision

- **Nobody runs `terraform apply` by hand.** `terraform.yml` runs `init → plan → apply -auto-approve` on any push to `trunk` that touches `terraform/**`. A failed apply is fixed forward. State lives in a versioned GCS bucket.
- **Terraform owns only what nothing else touches:** enabled APIs, the Artifact Registry repo and its keep-newest-5 cleanup policy, the Cloud Run service shape (`ignore_changes` on the image), service accounts and IAM, WIF, the Firestore database (`prevent_destroy`), and Identity Platform config. The **Firebase CLI** owns Hosting, rules and indexes. **CI** owns the image. **No console edits** to anything Terraform owns.
- **Manual and one-time, outside Terraform:** the billing account and its $1 budget, enabling the Google sign-in provider in the console (so no OAuth client secret ever reaches CI), and `terraform/bootstrap.sh`, which creates the project, the state bucket, the WIF pool/provider and the Terraform SA. That's chicken-and-egg: CI can't authenticate until those exist.
- **CI auth is WIF only**, restricted to `markgoho/troop-813` on `refs/heads/trunk`. No keys, and no secrets store in the MVP. Terraform uses its own SA. Deploys use a narrower deploy SA.
- **Three path-filtered workflows, one job each** (checks then deploy): `api.yml`, `web.yml`, `terraform.yml`. Docs-only commits run nothing. Each workflow queues and never cancels. Development is single-threaded: one push, one CI run at a time.
- **One environment** (prod), plus the Firebase emulators for local dev. Failures are fixed forward, with no rollback path.

## Considered and rejected

- **Local apply, optionally with CI plan/drift check** (doula-cloud): needs a human who remembers to apply and understands the plan. The drift machinery costs more than this infra.
- **Click-ops plus gcloud scripts, no Terraform**: the infra would be undocumented, and the point is GCP experience.
- **One workflow with dependent jobs** (API deploys before Hosting): pays a minute of rounding per job, and needs a filter job that runs even on docs commits. A commit touching both `api/` and `apps/` now deploys both in parallel, with a seconds-long mismatch. We accept that.

## Consequences

- The Terraform SA can change IAM in the project, so a bad commit to `terraform/` auto-applies. The trunk-only WIF condition and a single-owner repo are the guard.
- Markdown formatting isn't checked in CI on docs-only commits. Run `bun fmt` locally.
- A staging project doesn't exist. Real Identity Platform and rules first meet real traffic in prod.
