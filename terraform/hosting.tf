# One Hosting site per apps/* (issue #14). firebase.json + .firebaserc map the
# `hq` deploy target to it; the Firebase CLI deploys its content (web.yml).
resource "google_firebase_hosting_site" "hq" {
  provider = google-beta
  site_id  = "troop-813-hq"

  depends_on = [google_project_service.apis]
}
