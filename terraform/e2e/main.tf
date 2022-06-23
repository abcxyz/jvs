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
locals {
  github_slug = "abcxyz/jvs"
}

resource "google_project_service" "serviceusage" {
  project            = var.project_id
  service            = "serviceusage.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "services" {
  project = var.project_id
  for_each = toset([
    "cloudresourcemanager.googleapis.com",
    "iamcredentials.googleapis.com",
    "artifactregistry.googleapis.com",
  ])
  service            = each.value
  disable_on_destroy = false

  depends_on = [
    google_project_service.serviceusage,
  ]
}

resource "google_service_account" "gh-access-acc" {
  project      = var.project_id
  account_id   = "gh-access-sa"
  display_name = "GitHub Access Account"
}

// IAM roles needed to run tests.
resource "google_project_iam_member" "gh_access_acc_iam" {
  for_each = toset(var.e2e_iam_roles)
  project  = var.project_id
  role     = each.key
  member   = "serviceAccount:${google_service_account.gh-access-acc.email}"
}

module "abcxyz_pkg" {
  source      = "github.com/abcxyz/pkg//terraform/modules/workload-identity-federation"
  project_id  = var.project_id
  github_slug = local.github_slug
}

resource "google_service_account_iam_member" "external_provider_roles" {
  service_account_id = google_service_account.gh-access-acc.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${module.abcxyz_pkg.pool_name}/attribute.repository/${local.github_slug}"
}

resource "google_artifact_registry_repository" "image_registry" {
  provider = google-beta

  location      = var.artifact_registry_location
  project       = var.project_id
  repository_id = "docker-images"
  description   = "Container Registry for the images."
  format        = "DOCKER"
  depends_on = [
    google_project_service.services["artifactregistry.googleapis.com"],
  ]
}

module "jvs-e2e" {
  source     = "../modules/jvs-e2e"
  project_id = var.project_id
}
