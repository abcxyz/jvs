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
  github_owner_id = 93787867  # abcxyz
  github_repo_id  = 479173136 # abcxyz/jvs
}

resource "random_id" "default" {
  byte_length = 2
}

resource "google_project_service" "services" {
  for_each = toset([
    "cloudkms.googleapis.com",
  ])

  project = var.project_id

  service            = each.value
  disable_on_destroy = false
}

// Images build from the CI workflow will be uploaded to this AR repo
resource "google_artifact_registry_repository" "artifact_repository" {
  project = var.project_id

  location      = var.region
  repository_id = var.registry_repository_id
  description   = "Container registry for docker images."
  format        = "DOCKER"

  depends_on = [
    google_project_service.services["artifactregistry.googleapis.com"],
  ]
}

// ci service account will need repoAdmin role to read, write and delete images
resource "google_artifact_registry_repository_iam_member" "ci_service_account_iam" {
  project = google_artifact_registry_repository.artifact_repository.project

  location   = google_artifact_registry_repository.artifact_repository.location
  repository = google_artifact_registry_repository.artifact_repository.name
  role       = "roles/artifactregistry.repoAdmin"
  member     = format("serviceAccount:%s", var.ci_service_account_email)
}


module "jvs_common" {
  source = "../modules/common"

  project_id = var.project_id

  kms_key_location = var.kms_key_location
}

module "jvs_services" {
  source = "../modules/jvs-services"

  project_id = var.project_id

  region          = var.region
  service_ingress = "all"

  jvs_api_service_account          = module.jvs_common.jvs_api_service_account_email
  jvs_ui_service_account           = module.jvs_common.jvs_ui_service_account_email
  jvs_cert_rotator_service_account = module.jvs_common.jvs_cert_rotator_service_account_email
  jvs_public_key_service_account   = module.jvs_common.jvs_public_key_service_account_email

  jvs_container_image = var.jvs_container_image

  kms_keyring_id = module.jvs_common.kms_keyring_id
  kms_key_name   = "signing-${random_id.default.hex}"
  plugin_envvars = var.plugin_envvars
}

