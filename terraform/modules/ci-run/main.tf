/**
 * Copyright 2022 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

resource "google_folder" "top_folder" {
  display_name = var.top_folder_id
  parent       = var.folder_parent
}

resource "google_project" "jvs_project" {
  name            = var.project_id
  project_id      = var.project_id
  folder_id       = google_folder.top_folder.name
  billing_account = var.billing_account
}

resource "google_project_service" "server_project_services" {
  project = google_project.jvs_project.project_id
  for_each = toset([
    "serviceusage.googleapis.com",
    "artifactregistry.googleapis.com",
    "cloudkms.googleapis.com",
    "run.googleapis.com",
    "cloudresourcemanager.googleapis.com",
  ])
  service            = each.value
  disable_on_destroy = false
}

resource "google_kms_key_ring" "keyring" {
  project  = google_project.jvs_project.project_id
  name     = "jvs-keyring"
  location = var.key_location
}

resource "google_kms_crypto_key" "asymmetric-sign-key" {
  name     = "jvs-key"
  key_ring = google_kms_key_ring.keyring.id
  purpose  = "ASYMMETRIC_SIGN"

  version_template {
    algorithm = "EC_SIGN_P256_SHA256"
  }

  lifecycle {
    prevent_destroy = true
  }
}

resource "google_artifact_registry_repository" "image_registry" {
  provider = google-beta

  location      = var.artifact_registry_location
  project       = google_project.jvs_project.project_id
  repository_id = "images"
  description   = "Container Registry for the images."
  format        = "DOCKER"

  depends_on = [
    google_project_service.server_project_services,
  ]
}

module "jvs-service" {
  source                     = "../jvs-service"
  project_id                 = var.project_id
  service_name               = var.service_name
  artifact_registry_location = google_artifact_registry_repository.image_registry.location
  key_id                     = google_kms_crypto_key.asymmetric-sign-key.id

  depends_on = [
    google_project_service.server_project_services["run.googleapis.com"],
    google_project_service.server_project_services,
  ]
}
