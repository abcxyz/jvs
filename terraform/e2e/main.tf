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
    "cloudresourcemanager.googleapis.com",
    "compute.googleapis.com",
    "serviceusage.googleapis.com",
    "iap.googleapis.com",
  ])

  project            = var.project_id
  service            = each.value
  disable_on_destroy = false
}

# TODO(yolocs): This is not ideal because it's on the project level.
resource "google_project_iam_member" "jvs_invoker_iam" {
  for_each = toset(var.jvs_invoker_members)
  project  = var.project_id
  role     = "roles/run.invoker"
  member   = each.key
}

resource "random_id" "default" {
  byte_length = 2
}

module "jvs_common" {
  source           = "../modules/common"
  project_id       = var.project_id
  kms_key_location = var.kms_key_location
}

module "jvs_services" {
  source          = "../modules/jvs-services"
  project_id      = var.project_id
  region          = var.region
  service_ingress = "internal-and-cloud-load-balancing"

  jvs_api_service_account          = module.jvs_common.jvs_api_service_account_email
  jvs_ui_service_account           = module.jvs_common.jvs_ui_service_account_email
  jvs_cert_rotator_service_account = module.jvs_common.jvs_cert_rotator_service_account_email
  jvs_public_key_service_account   = module.jvs_common.jvs_public_key_service_account_email

  jvs_api_service_image          = var.jvs_api_service_image
  jvs_ui_service_image           = var.jvs_ui_service_image
  jvs_cert_rotator_service_image = var.jvs_cert_rotator_service_image
  jvs_public_key_service_image   = var.jvs_public_key_service_image

  kms_keyring_id = module.jvs_common.kms_keyring_id
  kms_key_name   = "signing-${random_id.default.hex}"
}
