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

module "ui_cloud_run" {
  source = "git::https://github.com/abcxyz/terraform-modules.git//modules/cloud_run?ref=5445543e21491176528fb5cd7adcb505d9dec5dd"

  project_id = var.project_id

  region                = var.region
  name                  = "jvs-ui"
  image                 = var.jvs_ui_service_image
  ingress               = var.service_ingress
  service_account_email = var.jvs_ui_service_account
  envvars               = merge({ "KEY" : google_kms_crypto_key.signing_key.id }, var.ui_envvars)
}
