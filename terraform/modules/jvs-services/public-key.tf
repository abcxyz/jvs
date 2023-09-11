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

resource "google_project_service" "kms_services" {
  for_each = toset([
    "cloudkms.googleapis.com",
  ])

  project = var.project_id

  service            = each.value
  disable_on_destroy = false
}

module "public_key_cloud_run" {
  source = "git::https://github.com/abcxyz/terraform-modules.git//modules/cloud_run?ref=46d3ffd82d7c3080bc5ec2cc788fe3e21176a8be"

  project_id = var.project_id

  region = var.region
  name   = "jvs-public-key"
  image  = var.jvs_container_image
  args   = ["public-key", "server"]

  ingress = var.service_ingress

  service_account_email = var.jvs_public_key_service_account
  service_iam = {
    admins     = []
    developers = []
    # Public key service is meant to be public.
    invokers = ["allUsers"]
  }

  envvars = merge({
    "PROJECT_ID" : var.project_id
    "JVS_KEY_NAMES" : google_kms_crypto_key.signing_key.id,
  }, var.public_key_envvars)

  depends_on = [
    google_project_service.services["cloudkms.googleapis.com"],
  ]
}
