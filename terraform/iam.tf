# Two service accounts besides bootstrap's Terraform SA:
#   api-runtime: what the API runs as
#   deploy:      what api.yml and web.yml impersonate through WIF

resource "google_service_account" "api_runtime" {
  account_id   = "api-runtime"
  display_name = "HQ API (Cloud Run runtime)"
  depends_on   = [google_project_service.apis]
}

# Firestore reads and writes (the API is the only writer, ADR 0001).
resource "google_project_iam_member" "api_runtime_firestore" {
  project = local.project
  role    = "roles/datastore.user"
  member  = google_service_account.api_runtime.member
}

# Read Auth users (internal/auth falls back to GetUser for a first sign-in's
# email/name).
resource "google_project_iam_member" "api_runtime_auth" {
  project = local.project
  role    = "roles/firebaseauth.viewer"
  member  = google_service_account.api_runtime.member
}

# signBlob on itself: the Admin SDK signs Kiosk custom tokens through IAM with
# ADC, so no key file exists (ADR 0003, issue #10).
resource "google_service_account_iam_member" "api_runtime_sign_self" {
  service_account_id = google_service_account.api_runtime.name
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = google_service_account.api_runtime.member
}

resource "google_service_account" "deploy" {
  account_id   = "deploy"
  display_name = "CI deploys (api.yml, web.yml)"
  depends_on   = [google_project_service.apis]
}

resource "google_service_account_iam_member" "deploy_wif" {
  service_account_id = google_service_account.deploy.name
  role               = "roles/iam.workloadIdentityUser"
  member             = local.github_principals
}

resource "google_project_iam_member" "deploy" {
  for_each = toset([
    "roles/run.developer",                     # deploy new revisions (api.yml); Hosting reads the service (web.yml)
    "roles/firebasehosting.admin",             # firebase deploy --only hosting
    "roles/firebaserules.admin",               # --only firestore:rules
    "roles/datastore.indexAdmin",              # --only firestore:indexes
    "roles/serviceusage.serviceUsageConsumer", # firebase-tools checks enabled APIs
  ])
  project = local.project
  role    = each.value
  member  = google_service_account.deploy.member
}

# Push images.
resource "google_artifact_registry_repository_iam_member" "deploy_push" {
  location   = google_artifact_registry_repository.api.location
  repository = google_artifact_registry_repository.api.name
  role       = "roles/artifactregistry.writer"
  member     = google_service_account.deploy.member
}

# Deploy revisions that run as api-runtime.
resource "google_service_account_iam_member" "deploy_acts_as_runtime" {
  service_account_id = google_service_account.api_runtime.name
  role               = "roles/iam.serviceAccountUser"
  member             = google_service_account.deploy.member
}
