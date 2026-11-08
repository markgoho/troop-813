# One Hosting site per apps/* (issue #14). firebase.json + .firebaserc map the
# `hq` deploy target to it; the Firebase CLI deploys its content (web.yml).
resource "google_firebase_hosting_site" "hq" {
  provider = google-beta
  site_id  = "troop-813-hq"
  app_id   = google_firebase_web_app.hq.app_id

  depends_on = [google_project_service.apis]
}

# HQ's Firebase web app: its config (API key, app ID; public values) is what
# the client SDK initializes with. web.yml looks it up by display name with
# `firebase apps:sdkconfig` and passes it to `vite build` as VITE_FIREBASE_*.
resource "google_firebase_web_app" "hq" {
  provider        = google-beta
  display_name    = "HQ"
  deletion_policy = "DELETE"

  depends_on = [google_project_service.apis]
}
