# The Go API (ADR 0001). Terraform owns the shape; api.yml owns the image.
# Hosting rewrites /api/** here (firebase.json), so the service is public and
# the API does its own auth.
resource "google_cloud_run_v2_service" "api" {
  name     = "api"
  location = local.region
  ingress  = "INGRESS_TRAFFIC_ALL"

  deletion_protection = false

  template {
    service_account                  = google_service_account.api_runtime.email
    max_instance_request_concurrency = 80
    timeout                          = "60s" # the Hosting rewrite gives up at 60 s anyway

    scaling {
      min_instance_count = 0 # scale to zero: the first Monday request pays a cold start
      max_instance_count = 2
    }

    containers {
      # Placeholder for the first apply only; api.yml deploys the real image.
      image = "us-docker.pkg.dev/cloudrun/container/hello"

      env {
        name  = "GOOGLE_CLOUD_PROJECT" # main.go needs it; Cloud Run doesn't set it
        value = local.project
      }

      resources {
        cpu_idle          = true # request-based billing
        startup_cpu_boost = true
        limits = {
          cpu    = "1"
          memory = "256Mi"
        }
      }
    }
  }

  lifecycle {
    # deploy-cloudrun writes the image and its own labels; leave them to CI.
    ignore_changes = [
      client,
      client_version,
      labels,
      template[0].labels,
      template[0].containers[0].image,
    ]
  }

  depends_on = [google_project_service.apis]
}

resource "google_cloud_run_v2_service_iam_member" "api_public" {
  name     = google_cloud_run_v2_service.api.name
  location = google_cloud_run_v2_service.api.location
  role     = "roles/run.invoker"
  member   = "allUsers"
}
