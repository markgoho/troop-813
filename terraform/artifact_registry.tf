# API images. api.yml pushes us-east1-docker.pkg.dev/troop-813/api/api:<sha>.
resource "google_artifact_registry_repository" "api" {
  location      = local.region
  repository_id = "api"
  format        = "DOCKER"

  # Keep the newest 5 versions, delete the rest (KEEP wins over DELETE),
  # so storage stays inside the free 0.5 GB.
  cleanup_policy_dry_run = false
  cleanup_policies {
    id     = "keep-newest-5"
    action = "KEEP"
    most_recent_versions {
      keep_count = 5
    }
  }
  cleanup_policies {
    id     = "delete-older"
    action = "DELETE"
    condition {
      tag_state = "ANY"
    }
  }

  depends_on = [google_project_service.apis]
}
