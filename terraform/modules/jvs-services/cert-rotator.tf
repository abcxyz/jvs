# Copyright 2023 The Authors (see AUTHORS file)
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

resource "google_project_service" "scheduler_api" {
  project            = var.project_id
  service            = "cloudscheduler.googleapis.com"
  disable_on_destroy = false
}

module "cert_rotator_cloud_run" {
  source = "git::https://github.com/abcxyz/terraform-modules.git//modules/cloud_run?ref=5445543e21491176528fb5cd7adcb505d9dec5dd"

  project_id = var.project_id
  region     = var.region
  name       = "jvs-cert-rotator"
  image      = var.jvs_cert_rotator_service_image

  # Cert rotator is not a user facing service. Ignore the ingress input.
  ingress = "all"

  service_account_email = var.jvs_cert_rotator_service_account
  envvars               = merge({ "KEY_NAMES" : google_kms_crypto_key.signing_key.id }, var.cert_rotator_envvars)
}

data "google_compute_default_service_account" "default" {
  project = var.project_id
}

resource "google_cloud_scheduler_job" "job" {
  # Don't create scheduler if cadence is zero.
  count       = var.kms_key_rotation_minutes > 0 ? 1 : 0
  name        = "cert-rotation-job"
  project     = var.project_id
  region      = var.region
  description = "Regularly executes the certificate rotator"
  schedule    = "*/${var.kms_key_rotation_minutes} * * * *"

  http_target {
    http_method = "POST"
    uri         = module.cert_rotator_cloud_run.url

    oidc_token {
      # TODO(yolocs): Shouldn't use the default service account.
      service_account_email = data.google_compute_default_service_account.default.email
    }
  }

  depends_on = [
    google_project_service.scheduler_api,
  ]
}
