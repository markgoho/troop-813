#!/usr/bin/env bash
# One-time, by a human, before CI can run Terraform (ADR 0005). Run it through
# the wizard (scripts/prod-setup-wizard.sh), which does billing first.
#
# Uses the EXISTING troop-813 project (it never creates or deletes one) and
# creates what CI needs to authenticate and keep state, which Terraform can't
# create for itself:
#   - gs://troop-813-tfstate: versioned Terraform state bucket, us-east1
#   - WIF pool `github` + OIDC provider `github`: trusts GitHub Actions tokens
#     only from markgoho/troop-813 (pinned by repo ID too, so a re-created repo
#     of the same name isn't trusted) on refs/heads/trunk
#   - terraform@troop-813.iam.gserviceaccount.com: roles/owner, impersonable
#     only through that pool. Owner because Terraform manages IAM and enables
#     APIs; ADR 0005 accepts that a bad commit to terraform/ auto-applies.
#
# Idempotent: every resource is described first and created only if missing;
# the provider's trust condition and the IAM bindings are re-applied each run.
# Needs: gcloud, logged in as a project owner (`gcloud auth login`).
set -euo pipefail

PROJECT_ID=troop-813
REGION=us-east1
STATE_BUCKET=gs://troop-813-tfstate
POOL=github
PROVIDER=github
REPO=markgoho/troop-813
REPO_ID=1448981854 # gh api repos/markgoho/troop-813 --jq .id
TF_SA_NAME=terraform
TF_SA="${TF_SA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"

ATTRIBUTE_MAPPING="google.subject=assertion.sub,attribute.repository=assertion.repository,attribute.repository_id=assertion.repository_id,attribute.ref=assertion.ref"
ATTRIBUTE_CONDITION="assertion.repository=='${REPO}' && assertion.repository_id=='${REPO_ID}' && assertion.ref=='refs/heads/trunk'"

log() { printf '==> %s\n' "$*"; }
die() { printf 'bootstrap: %s\n' "$*" >&2; exit 1; }

# ── Preconditions: change nothing until both hold ────────────────────────────
PROJECT_NUMBER=$(gcloud projects describe "$PROJECT_ID" --format='value(projectNumber)' 2>/dev/null) ||
  die "project $PROJECT_ID not found (or no access). It must already exist; this script never creates one."
[[ -n "$PROJECT_NUMBER" ]] || die "project $PROJECT_ID returned no project number"

billing=$(gcloud billing projects describe "$PROJECT_ID" --format='value(billingEnabled)')
[[ "$billing" == "True" ]] ||
  die "billing is not linked to $PROJECT_ID. Link the dedicated billing account first (wizard stage 2)."

WIF_PRINCIPALS="principalSet://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${POOL}/attribute.repository/${REPO}"

# ── APIs bootstrap itself needs (Terraform enables the rest) ─────────────────
log "enabling bootstrap APIs"
gcloud services enable --project="$PROJECT_ID" \
  iam.googleapis.com iamcredentials.googleapis.com sts.googleapis.com \
  cloudresourcemanager.googleapis.com serviceusage.googleapis.com storage.googleapis.com

# ── State bucket ─────────────────────────────────────────────────────────────
if gcloud storage buckets describe "$STATE_BUCKET" --project="$PROJECT_ID" >/dev/null 2>&1; then
  log "state bucket $STATE_BUCKET exists"
else
  log "creating state bucket $STATE_BUCKET"
  gcloud storage buckets create "$STATE_BUCKET" --project="$PROJECT_ID" \
    --location="$REGION" --uniform-bucket-level-access --public-access-prevention
fi
gcloud storage buckets update "$STATE_BUCKET" --project="$PROJECT_ID" --versioning

# ── WIF pool + provider ──────────────────────────────────────────────────────
if gcloud iam workload-identity-pools describe "$POOL" --project="$PROJECT_ID" --location=global >/dev/null 2>&1; then
  log "WIF pool $POOL exists"
else
  log "creating WIF pool $POOL"
  gcloud iam workload-identity-pools create "$POOL" --project="$PROJECT_ID" --location=global \
    --display-name="GitHub Actions"
fi

provider_flags=(
  --project="$PROJECT_ID" --location=global --workload-identity-pool="$POOL"
  --issuer-uri=https://token.actions.githubusercontent.com
  --attribute-mapping="$ATTRIBUTE_MAPPING"
  --attribute-condition="$ATTRIBUTE_CONDITION"
)
if gcloud iam workload-identity-pools providers describe "$PROVIDER" --project="$PROJECT_ID" \
  --location=global --workload-identity-pool="$POOL" >/dev/null 2>&1; then
  log "WIF provider $PROVIDER exists; re-applying its trust condition"
  gcloud iam workload-identity-pools providers update-oidc "$PROVIDER" "${provider_flags[@]}"
else
  log "creating WIF provider $PROVIDER (trunk on $REPO only)"
  gcloud iam workload-identity-pools providers create-oidc "$PROVIDER" "${provider_flags[@]}" \
    --display-name="GitHub trunk"
fi

# ── Terraform SA ─────────────────────────────────────────────────────────────
if gcloud iam service-accounts describe "$TF_SA" --project="$PROJECT_ID" >/dev/null 2>&1; then
  log "Terraform SA $TF_SA exists"
else
  log "creating Terraform SA $TF_SA"
  gcloud iam service-accounts create "$TF_SA_NAME" --project="$PROJECT_ID" \
    --display-name="Terraform (CI only)"
fi

log "granting roles"
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:${TF_SA}" --role=roles/owner --condition=None >/dev/null
gcloud storage buckets add-iam-policy-binding "$STATE_BUCKET" --project="$PROJECT_ID" \
  --member="serviceAccount:${TF_SA}" --role=roles/storage.objectAdmin >/dev/null
gcloud iam service-accounts add-iam-policy-binding "$TF_SA" --project="$PROJECT_ID" \
  --member="$WIF_PRINCIPALS" --role=roles/iam.workloadIdentityUser >/dev/null

log "done"
cat <<EOF

  workload_identity_provider: projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${POOL}/providers/${PROVIDER}
  terraform service_account:  ${TF_SA}
  state bucket:               ${STATE_BUCKET}
EOF
