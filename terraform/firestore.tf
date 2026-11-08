# Firestore Native, us-east1 (ADR 0001). No backups (accepted risk), so the
# database is guarded twice: Terraform refuses to plan its destruction, and
# GCP refuses to delete it.
resource "google_firestore_database" "default" {
  name                    = "(default)"
  location_id             = local.region
  type                    = "FIRESTORE_NATIVE"
  delete_protection_state = "DELETE_PROTECTION_ENABLED"
  deletion_policy         = "ABANDON"

  lifecycle {
    prevent_destroy = true
  }

  depends_on = [google_project_service.apis]
}
