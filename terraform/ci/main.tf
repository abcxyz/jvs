resource "google_project_service" "server_project_services" {
  project = var.project_id
  for_each = toset([
    "cloudresourcemanager.googleapis.com",
    "serviceusage.googleapis.com",
    "iamcredentials.googleapis.com",
    "artifactregistry.googleapis.com",
    "cloudkms.googleapis.com",
  ])
  service            = each.value
  disable_on_destroy = false
}

module "github_action" {
  count      = var.is_local_env ? 0 : 1
  source     = "../modules/github-action"
  project_id = var.project_id
}

resource "google_kms_key_ring" "keyring" {
  project  = var.project_id
  name     = "ci-keyring"
  location = var.key_location
}

resource "google_artifact_registry_repository" "image_registry" {
  provider = google-beta

  location      = var.artifact_registry_location
  project       = var.project_id
  repository_id = "docker-images"
  description   = "Container Registry for the images."
  format        = "DOCKER"
}




