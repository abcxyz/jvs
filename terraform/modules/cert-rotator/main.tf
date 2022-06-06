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

resource "null_resource" "build" {
  triggers = {
    "tag" = var.tag
  }

  provisioner "local-exec" {
    environment = {
      PROJECT_ID = var.project_id
      TAG        = var.tag
      REPO       = "${var.artifact_registry_location}-docker.pkg.dev/${var.project_id}/docker-images/jvs"
      APP_NAME   = "rotator-service"
    }

    command = "${path.module}/../../../scripts/build.sh cert-rotation"
  }
}

resource "google_cloud_run_service" "cert-rotator" {
  name     = "cert-rotator-${var.tag}"
  location = var.region
  project  = var.project_id

  template {
    spec {
      service_account_name = var.service_account

      containers {
        image = "${var.artifact_registry_location}-docker.pkg.dev/${var.project_id}/docker-images/jvs/rotator-service:${var.tag}"

        resources {
          limits = {
            cpu    = "1000m"
            memory = "1G"
          }
        }
        env {
          name  = "JVS_KEY_NAMES"
          value = var.key_id
        }
        env {
          name  = "JVS_KEY_TTL"
          value = var.key_ttl
        }
        env {
          name  = "JVS_GRACE_PERIOD"
          value = var.key_grace_period
        }
        env {
          name  = "JVS_DISABLED_PERIOD"
          value = var.key_disabled_period
        }
        env {
          name  = "JVS_PROPAGATION_DELAY"
          value = var.key_propagation_delay
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

data "google_compute_default_service_account" "default" {
  project = var.project_id
}

resource "google_cloud_scheduler_job" "job" {
  name        = "cert-rotation-job-${var.tag}"
  project     = var.project_id
  region      = var.region
  description = "Regularly executes the certificate rotator"
  schedule    = "*/${var.cadence} * * * *"

  http_target {
    http_method = "POST"
    uri         = google_cloud_run_service.cert-rotator.status.0.url

    oidc_token {
      service_account_email = data.google_compute_default_service_account.default.email
    }
  }
}
