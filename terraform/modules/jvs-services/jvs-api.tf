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

module "api_cloud_run" {
  source = "git::https://github.com/abcxyz/terraform-modules.git//modules/cloud_run?ref=5445543e21491176528fb5cd7adcb505d9dec5dd"

  project_id = var.project_id

  region = var.region
  name   = "jvs-api"
  image  = var.jvs_container_image
  args   = ["api", "server"]

  ingress = var.service_ingress

  service_account_email = var.jvs_api_service_account

  envvars = merge({
    "PROJECT_ID" : var.project_id
    "KEY" : google_kms_crypto_key.signing_key.id,
  }, var.api_envvars)
}
