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

resource "google_project_service" "services" {
  for_each = toset([
    "cloudkms.googleapis.com",
  ])

  project = var.project_id

  service            = each.value
  disable_on_destroy = false
}

resource "random_id" "default" {
  byte_length = 2
}

resource "google_kms_key_ring" "keyring" {
  project = var.project_id

  name     = "${var.kms_keyring_name}-${random_id.default.hex}"
  location = var.kms_key_location
  depends_on = [
    google_project_service.services["cloudkms.googleapis.com"],
  ]
}

resource "google_service_account" "api_acc" {
  project = var.project_id

  account_id   = var.jvs_api_service_account_name
  display_name = "JVS API Service Account"
}

resource "google_kms_key_ring_iam_member" "api_acc_roles" {
  for_each = toset([
    "roles/cloudkms.viewer",
    "roles/cloudkms.cryptoOperator"
  ])

  key_ring_id = google_kms_key_ring.keyring.id
  role        = each.key
  member      = google_service_account.api_acc.member
}

resource "google_service_account" "ui_acc" {
  project = var.project_id

  account_id   = var.jvs_ui_service_account_name
  display_name = "JVS UI Service Account"
}

resource "google_kms_key_ring_iam_member" "ui_acc_roles" {
  for_each = toset([
    "roles/cloudkms.viewer",
    "roles/cloudkms.cryptoOperator"
  ])

  key_ring_id = google_kms_key_ring.keyring.id
  role        = each.key
  member      = google_service_account.ui_acc.member
}

resource "google_service_account" "rotator_acc" {
  project = var.project_id

  account_id   = var.jvs_cert_rotator_service_account_name
  display_name = "Rotator Service Account"
}

resource "google_kms_key_ring_iam_member" "rotator_acc_roles" {
  for_each = toset([
    "roles/cloudkms.admin",
  ])

  key_ring_id = google_kms_key_ring.keyring.id
  role        = each.key
  member      = google_service_account.rotator_acc.member
}

resource "google_service_account" "public_key_acc" {
  project = var.project_id

  account_id   = var.jvs_public_key_service_account_name
  display_name = "Public Key Hosting Service Account"
}

resource "google_kms_key_ring_iam_member" "public_key_acc_roles" {
  for_each = toset([
    "roles/cloudkms.publicKeyViewer",
    "roles/cloudkms.viewer",
  ])

  key_ring_id = google_kms_key_ring.keyring.id
  role        = each.key
  member      = google_service_account.public_key_acc.member
}
