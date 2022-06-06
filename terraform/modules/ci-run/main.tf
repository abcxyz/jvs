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

resource "google_kms_key_ring_iam_member" "server_acc_roles" {
  for_each = toset([
    "roles/cloudkms.viewer",
    "roles/cloudkms.cryptoOperator"
  ])

  key_ring_id = var.key_ring
  role        = each.key
  member      = "serviceAccount:${var.jvs_service_account}"
}

resource "google_kms_key_ring_iam_member" "rotator_acc_roles" {
  for_each = toset([
    "roles/cloudkms.admin",
  ])

  key_ring_id = var.key_ring
  role        = each.key
  member      = "serviceAccount:${var.rotator_service_account}"
}

module "jvs-service" {
  source          = "../jvs-service"
  project_id      = var.project_id
  key_ring        = var.key_ring
  service_account = var.jvs_service_account
  tag             = local.tag
  depends_on      = [google_kms_key_ring_iam_member.server_acc_roles]
}

module "cert-rotator" {
  source                = "../cert-rotator"
  project_id            = var.project_id
  key_ring              = var.key_ring
  service_account       = var.rotator_service_account
  tag                   = local.tag
  key_disabled_period   = var.key_disabled_period
  key_grace_period      = var.key_grace_period
  key_propagation_delay = var.key_propagation_delay
  key_ttl               = var.key_ttl
  depends_on            = [google_kms_key_ring_iam_member.rotator_acc_roles]
}
