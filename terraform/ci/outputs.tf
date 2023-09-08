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


output "jvs_api_service_account_email" {
  description = "JVS API service account email."
  value       = module.jvs_common.jvs_api_service_account_email
}

output "jvs_api_service_account_name" {
  description = "JVS API service account name."
  value       = module.jvs_common.jvs_api_service_account_name
}

output "jvs_ui_service_account_email" {
  description = "JVS UI service account email."
  value       = module.jvs_common.jvs_ui_service_account_email
}

output "jvs_ui_service_account_name" {
  description = "JVS UI service account name."
  value       = module.jvs_common.jvs_ui_service_account_name
}

output "jvs_cert_rotator_service_account_email" {
  description = "JVS cert rotator service account email."
  value       = module.jvs_common.jvs_cert_rotator_service_account_email
}

output "jvs_cert_rotator_service_account_name" {
  description = "JVS cert rotator service account name."
  value       = module.jvs_common.jvs_cert_rotator_service_account_name
}

output "jvs_public_key_service_account_email" {
  description = "JVS public key service account email."
  value       = module.jvs_common.jvs_public_key_service_account_email
}

output "jvs_public_key_service_account_name" {
  description = "JVS public key service account name."
  value       = module.jvs_common.jvs_public_key_service_account_name
}

output "kms_keyring_id" {
  description = "KMS keyring for JVS."
  value       = module.jvs_common.kms_keyring_id
}
