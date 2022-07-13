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

// intended to be run once to set up the environment.

resource "google_project_service" "serviceusage" {
  project            = var.project_id
  service            = "serviceusage.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "services" {
  project = var.project_id
  for_each = toset([
    "cloudresourcemanager.googleapis.com",
    "cloudkms.googleapis.com",
    "iam.googleapis.com"
  ])
  service            = each.value
  disable_on_destroy = false

  depends_on = [
    google_project_service.serviceusage,
  ]
}

resource "google_kms_key_ring" "keyring" {
  project  = var.project_id
  name     = "ci-keyring"
  location = var.key_location
  depends_on = [
    google_project_service.services["cloudkms.googleapis.com"],
  ]
}

resource "google_kms_crypto_key" "asymmetric-sign-key" {
  name     = "jvs-key"
  key_ring = google_kms_key_ring.keyring.id
  purpose  = "ASYMMETRIC_SIGN"

  version_template {
    algorithm = "EC_SIGN_P256_SHA256"
  }

  lifecycle {
    prevent_destroy = false
  }

  depends_on = [
    google_project_service.services["cloudkms.googleapis.com"],
  ]
}

resource "google_service_account" "server-acc" {
  project      = var.project_id
  account_id   = "jvs-service-sa"
  display_name = "JVS Service Account"
}

resource "google_service_account" "rotator-acc" {
  project      = var.project_id
  account_id   = "rotator-sa"
  display_name = "Rotator Service Account"
}

resource "google_service_account" "public-key-acc" {
  project      = var.project_id
  account_id   = "pubkey-sa"
  display_name = "Public Key Hosting Service Account"
}

resource "google_kms_key_ring_iam_member" "server_acc_roles" {
  for_each = toset([
    "roles/cloudkms.viewer",
    "roles/cloudkms.cryptoOperator"
  ])

  key_ring_id = google_kms_key_ring.keyring.id
  role        = each.key
  member      = "serviceAccount:${google_service_account.server-acc.email}"
}

resource "google_kms_key_ring_iam_member" "rotator_acc_roles" {
  for_each = toset([
    "roles/cloudkms.admin",
  ])

  key_ring_id = google_kms_key_ring.keyring.id
  role        = each.key
  member      = "serviceAccount:${google_service_account.rotator-acc.email}"
}

resource "google_kms_key_ring_iam_member" "public_key_acc_roles" {
  for_each = toset([
    "roles/cloudkms.publicKeyViewer",
    "roles/cloudkms.viewer",
  ])

  key_ring_id = google_kms_key_ring.keyring.id
  role        = each.key
  member      = "serviceAccount:${google_service_account.public-key-acc.email}"
}

module "jvs-service" {
  source          = "../jvs-service"
  project_id      = var.project_id
  key_id          = google_kms_crypto_key.asymmetric-sign-key.id
  service_account = google_service_account.server-acc.email
  tag             = local.tag
  depends_on      = [google_kms_key_ring_iam_member.server_acc_roles]
}

module "cert-rotator" {
  source                = "../cert-rotator"
  project_id            = var.project_id
  key_id                = google_kms_crypto_key.asymmetric-sign-key.id
  service_account       = google_service_account.rotator-acc.email
  tag                   = local.tag
  key_disabled_period   = var.key_disabled_period
  key_grace_period      = var.key_grace_period
  key_propagation_delay = var.key_propagation_delay
  key_ttl               = var.key_ttl
  depends_on            = [google_kms_key_ring_iam_member.rotator_acc_roles]
}

module "cert-actions" {
  source                = "../cert-action-service"
  project_id            = var.project_id
  key_id                = google_kms_crypto_key.asymmetric-sign-key.id
  service_account       = google_service_account.rotator-acc.email
  tag                   = local.tag
  key_disabled_period   = var.key_disabled_period
  key_grace_period      = var.key_grace_period
  key_propagation_delay = var.key_propagation_delay
  key_ttl               = var.key_ttl
  depends_on            = [google_kms_key_ring_iam_member.rotator_acc_roles]
}

module "public-key" {
  source          = "../public-key"
  project_id      = var.project_id
  key_id          = google_kms_crypto_key.asymmetric-sign-key.id
  service_account = google_service_account.public-key-acc.email
  tag             = local.tag
  depends_on      = [google_kms_key_ring_iam_member.public_key_acc_roles]
}

module "monitoring" {
  source                     = "../monitoring"
  project_id                 = var.project_id
  jvs_service_name           = "jvs-${local.tag}"
  cert_rotation_service_name = "cert-rotator-${local.tag}"
  public_key_service_name    = "pubkey-${local.tag}"
}
