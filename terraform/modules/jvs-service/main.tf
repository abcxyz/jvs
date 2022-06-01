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

resource "google_project_service" "resourcemanager" {
  project            = var.project_id
  service            = "cloudresourcemanager.googleapis.com"
  disable_on_destroy = false
}

resource "google_service_account" "server-acc" {
  project      = var.project_id
  account_id   = "jvs-service-sa"
  display_name = "JWT Service Account"
}

resource "google_project_iam_member" "server_roles" {
  for_each = toset([
    "roles/cloudkms.viewer",
    "roles/cloudkms.cryptoOperator"
  ])

  project = var.project_id
  role    = each.key
  condition {
    title      = "Only on relevant key"
    expression = "resource.name.startsWith(\"${var.key_id}\")"
  }
  member = "serviceAccount:${google_service_account.server-acc.email}"
}

resource "google_cloud_run_service" "server" {
  name     = var.service_name
  location = var.region
  project  = var.project_id

  template {
    spec {
      service_account_name = google_service_account.server-acc.email

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
