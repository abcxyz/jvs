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

// intended to be run on each ci run. uses an environment set up by env/ci

resource "google_kms_crypto_key" "asymmetric-sign-key" {
  name     = uuid()
  key_ring = var.key_ring
  purpose  = "ASYMMETRIC_SIGN"

  version_template {
    algorithm = "EC_SIGN_P256_SHA256"
  }
}

resource "google_project_iam_member" "server_acc_roles" {
  for_each = toset([
    "roles/cloudkms.viewer",
    "roles/cloudkms.cryptoOperator"
  ])

  project = var.project_id
  role    = each.key
  condition {
    title      = "Only on relevant key"
    expression = "resource.name.startsWith(\"${google_kms_crypto_key.asymmetric-sign-key.id}\")"
  }
  member = "serviceAccount:${var.jvs_service_account}"
}

module "jvs-service" {
  source          = "../jvs-service"
  project_id      = var.project_id
  service_name    = var.service_name
  key_id          = google_kms_crypto_key.asymmetric-sign-key.id
  jvs_service_acc = var.jvs_service_account
  depends_on = [google_project_iam_member.server_acc_roles]
}
