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

locals {
  tag = uuid()
}

resource "google_kms_key_ring" "keyring" {
  project  = var.project_id
  name     = "ci-keyring"
  location = var.key_location
}


resource "google_kms_key_ring_iam_member" "server_acc_roles" {
  for_each = toset([
    "roles/cloudkms.viewer",
    "roles/cloudkms.cryptoOperator"
  ])

  key_ring_id = google_kms_key_ring.keyring.id
  role        = each.key
  member      = "serviceAccount:${var.jvs_service_account}"
}

resource "google_kms_key_ring_iam_member" "rotator_acc_roles" {
  for_each = toset([
    "roles/cloudkms.admin",
  ])

  key_ring_id = google_kms_key_ring.keyring.id
  role        = each.key
  member      = "serviceAccount:${var.rotator_service_account}"
}
