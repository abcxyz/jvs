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

locals {
  project_id = "REPLACE_PROJECT_ID"
}

module "jvs_e2e" {
  source = "git::https://github.com/abcxyz/jvs.git//terraform/e2e?ref=REPLACE_JVS_SHA"

  project_id = local.project_id

  region = "REPLACE_REGION"

  kms_key_location    = "REPLACE_KMS_KEY_LOCATION"
  jvs_invoker_members = REPLACE_JVS_INVOKE_MEMBERS
  jvs_container_image = "REPLACE_JVS_CONTAINER_IMAGE"
  jvs_api_domain      = "REPLACE_JVS_API_DOMAIN"
  jvs_ui_domain       = "REPLACE_JVS_UI_DOMAIN"
  iap_support_email   = "REPLACE_IAP_SUPPORT_EMAIL"

  notification_channel_email = "REPLACE_NOTIFICATION_CHANNEL_EMAIL"

  // Use gcloud app id because Cloud Run accepts it.
  prober_audience  = "REPLACE_PROBER_AUDIENCE"
  jvs_prober_image = "REPLACE_JVS_PROBER_IMAGE"
  alert_enabled    = REPLACE_ALERT_ENABLED

  plugin_envvars = REPLACE_PLUGIN_ENVVARS
}
