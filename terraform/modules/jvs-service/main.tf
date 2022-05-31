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
  tag = uuid()
}

resource "google_folder" "top_folder" {
  display_name = var.top_folder_id
  parent       = var.folder_parent
}

resource "google_project" "jvs_project" {
  name            = "jvs-dev-app-0"
  project_id      = "jvs-dev-app-0"
  folder_id       = google_folder.top_folder.name
  billing_account = var.billing_account
}

resource "google_project_service" "app_project_serviceusage" {
  count              = 1
  project            = google_project.jvs_project.project_id
  service            = "serviceusage.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "app_project_resourcemanager" {
  count              = 1
  project            = google_project.jvs_project.project_id
  service            = "cloudresourcemanager.googleapis.com"
  disable_on_destroy = false

  depends_on = [
    google_project_service.app_project_serviceusage,
  ]
}

resource "google_project_service" "server_project_services" {
  project = google_project.jvs_project.project_id
  for_each = toset([
    "serviceusage.googleapis.com",
    "artifactregistry.googleapis.com",
    "cloudkms.googleapis.com",
  ])
  service            = each.value
  disable_on_destroy = false

  depends_on = [
    google_project_service.app_project_resourcemanager,
  ]
}

resource "google_kms_key_ring" "keyring" {
  project  = google_project.jvs_project.project_id
  name     = "jvs-keyring"
  location = "global"
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

resource "null_resource" "build" {
  triggers = {
    "tag" = local.tag
  }

  provisioner "local-exec" {
    environment = {
      PROJECT_ID = google_project.jvs_project.project_id
      TAG        = local.tag
      REPO       = "${var.artifact_registry_location}-docker.pkg.dev/${google_project.jvs_project.project_id}/images/lumberjack"
      APP_NAME   = "go-jvs-service"
    }

    command = "${path.module}/../../../scripts/build.sh justification"
  }

  depends_on = [
    google_artifact_registry_repository.image_registry,
  ]
}

resource "google_project_service" "resourcemanager" {
  project            = google_project.jvs_project.project_id
  service            = "cloudresourcemanager.googleapis.com"
  disable_on_destroy = false
}

resource "google_service_account" "server" {
  count        = 1
  project      = google_project.jvs_project.project_id
  account_id   = "${var.service_name}-sa"
  display_name = "JWT Service Account"
}

resource "google_kms_crypto_key_iam_binding" "view_role" {
  crypto_key_id = google_kms_crypto_key.asymmetric-sign-key.id
  role = "roles/cloudkms.viewer"
  members = [
    "serviceAccount:${google_service_account.server[0].email}"
  ]
}

resource "google_kms_crypto_key_iam_binding" "operator_role" {
  crypto_key_id = google_kms_crypto_key.asymmetric-sign-key.id
  role = "roles/cloudkms.cryptoOperator"
  members = [
    "serviceAccount:${google_service_account.server[0].email}"
  ]
}


resource "google_cloud_run_service" "server" {
  name     = var.service_name
  location = var.region
  project  = google_project.jvs_project.project_id

  template {
    spec {
      service_account_name = google_service_account.server[0].email

      containers {
        image = "${var.artifact_registry_location}-docker.pkg.dev/${google_project.jvs_project.project_id}/images/lumberjack/go-jvs-service:${local.tag}"

        resources {
          limits = {
            cpu    = "1000m"
            memory = "1G"
          }
        }
        env {
          name  = "JVS_KEY"
          value = google_kms_crypto_key.asymmetric-sign-key.id
        }
      }
    }

  }

  autogenerate_revision_name = true

  depends_on = [
    google_project_service.server_project_services["run.googleapis.com"],
    google_project_service.server_project_services,
    null_resource.build,
  ]

  lifecycle {
    ignore_changes = [
      metadata[0].annotations["client.knative.dev/user-image"],
      metadata[0].annotations["run.googleapis.com/client-name"],
      metadata[0].annotations["run.googleapis.com/client-version"],
      metadata[0].annotations["run.googleapis.com/ingress-status"],
      metadata[0].annotations["serving.knative.dev/creator"],
      metadata[0].annotations["serving.knative.dev/lastModifier"],
      metadata[0].labels["cloud.googleapis.com/location"],
      template[0].metadata[0].annotations["client.knative.dev/user-image"],
      template[0].metadata[0].annotations["run.googleapis.com/client-name"],
      template[0].metadata[0].annotations["run.googleapis.com/client-version"],
      template[0].metadata[0].annotations["serving.knative.dev/creator"],
      template[0].metadata[0].annotations["serving.knative.dev/lastModifier"],
    ]
  }
}
