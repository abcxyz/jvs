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

// intended to be run once to set up the environment.

resource "google_project" "jvs_project" {
  name            = var.project_id
  project_id      = var.project_id
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
    "cloudscheduler.googleapis.com",
  ])
  service            = each.value
  disable_on_destroy = false
}

resource "google_kms_key_ring" "keyring" {
  project  = var.project_id
  name     = "jvs-keyring"
  location = var.key_location
}

resource "google_service_account" "server-acc" {
  project      = var.project_id
  account_id   = "jvs-service-sa"
  display_name = "JVS Service Account"
}

resource "google_service_account" "rotator-acc" {
  project      = var.project_id
  account_id   = "rotator-sa"
  display_name = "Rotator Service Account"
}

resource "google_artifact_registry_repository" "image_registry" {
  provider = google-beta

  location      = var.artifact_registry_location
  project       = var.project_id
  repository_id = "docker-images"
  description   = "Container Registry for the images."
  format        = "DOCKER"
}
