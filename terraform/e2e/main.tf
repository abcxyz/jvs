resource "google_project_service" "server_project_services" {
  project = var.project_id
  for_each = toset([
    "artifactregistry.googleapis.com",
    "cloudkms.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "cloudscheduler.googleapis.com",
    "compute.googleapis.com",
    "run.googleapis.com",
    "serviceusage.googleapis.com",
  ])
  service            = each.value
  disable_on_destroy = false
}

module "github_actions" {
  source     = "../modules/github-actions"
  project_id = var.project_id
}

resource "google_artifact_registry_repository" "image_registry" {
  provider = google-beta

  location      = var.artifact_registry_location
  project       = var.project_id
  repository_id = "docker-images"
  description   = "Container Registry for the images."
  format        = "DOCKER"
}

module "e2e" {
  source     = "../modules/e2e-setup"
  project_id = var.project_id
}




