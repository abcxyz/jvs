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

resource "google_artifact_registry_repository" "image_registry" {
  provider = google-beta

  location      = var.artifact_registry_location
  project       = var.project_id
  repository_id = "images"
  description   = "Container Registry for the images."
  format        = "DOCKER"
}

resource "null_resource" "build" {
  triggers = {
    "tag" = local.tag
  }

  provisioner "local-exec" {
    environment = {
      PROJECT_ID = var.project_id
      TAG        = local.tag
      REPO       = "${var.artifact_registry_location}-docker.pkg.dev/${var.project_id}/images/jvs"
      APP_NAME   = "jvs-service"
    }

    command = "${path.module}/../../../scripts/build.sh justification"
  }
}

resource "google_cloud_run_service" "server" {
  name     = var.service_name
  location = var.region
  project  = var.project_id

  template {
    spec {
      service_account_name = var.jvs_service_acc

      containers {
        image = "${var.artifact_registry_location}-docker.pkg.dev/${var.project_id}/images/jvs/jvs-service:${local.tag}"

        resources {
          limits = {
            cpu    = "1000m"
            memory = "1G"
          }
        }
        env {
          name  = "JVS_KEY"
          value = var.key_id
        }
      }
    }

  }

  autogenerate_revision_name = true

  depends_on = [
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