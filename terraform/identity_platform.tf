# Identity Platform (ADR 0001). Creating this config is what turns Auth on in
# the project, so it must apply BEFORE anyone enables the Google provider in
# the console (wizard stage order); otherwise the create fails as "already
# exists" and needs an import block.
#
# The Google sign-in provider itself is NOT here: enabling it in the console
# creates the OAuth client, so its secret never reaches CI (ADR 0005).
resource "google_identity_platform_config" "default" {
  # localhost and <project>.web.app/.firebaseapp.com are the defaults; every
  # extra Hosting site or custom domain is added here (issue #14).
  authorized_domains = [
    "localhost",
    "${local.project}.firebaseapp.com",
    "${local.project}.web.app",
    "${google_firebase_hosting_site.hq.site_id}.web.app",
  ]

  depends_on = [google_project_service.apis]
}
