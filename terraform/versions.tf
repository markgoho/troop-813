# Applied only by CI (.github/workflows/terraform.yml, ADR 0005). Never run
# `terraform apply` by hand; fix a failed apply forward with a commit.
# The state bucket, WIF pool/provider and the Terraform SA come from bootstrap.sh.
terraform {
  required_version = "~> 1.17"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 7.0"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = "~> 7.0"
    }
  }

  backend "gcs" {
    bucket = "troop-813-tfstate"
    prefix = "state"
  }
}

locals {
  project        = "troop-813"
  project_number = "1003933813667"
  region         = "us-east1"

  # Every GitHub Actions token the bootstrap provider accepts (already limited
  # to trunk on markgoho/troop-813 by its attribute condition).
  github_principals = "principalSet://iam.googleapis.com/projects/${local.project_number}/locations/global/workloadIdentityPools/github/attribute.repository/markgoho/troop-813"
}

# billing_project + user_project_override: Firebase and Identity Platform APIs
# want a quota project on every call.
provider "google" {
  project               = local.project
  region                = local.region
  billing_project       = local.project
  user_project_override = true
}

provider "google-beta" {
  project               = local.project
  region                = local.region
  billing_project       = local.project
  user_project_override = true
}
