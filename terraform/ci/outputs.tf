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

output "artifact_repository_name" {
  description = "The Artifact Registry name."
  value       = module.github_ci_infra.artifact_repository_name
}

output "wif_pool_name" {
  description = "The Workload Identity Federation pool name."
  value       = module.github_ci_infra.wif_pool_name
}

output "wif_provider_name" {
  description = "The Workload Identity Federation provider name."
  value       = module.github_ci_infra.wif_provider_name
}

output "ci_service_account_email" {
  description = "CI service account identity email address."
  value       = module.github_ci_infra.service_account_email
}

output "ci_service_account_member" {
  description = "CI service account identity in the form serviceAccount:{email}."
  value       = module.github_ci_infra.service_account_member
}

output "jvs_api_service_account_email" {
  description = "JVS API service account email."
  value       = module.jvs_common.jvs_api_service_account_email
}

output "jvs_ui_service_account_email" {
  description = "JVS UI service account email."
  value       = module.jvs_common.jvs_ui_service_account_email
}

output "jvs_cert_rotator_service_account_email" {
  description = "JVS cert rotator service account email."
  value       = module.jvs_common.jvs_cert_rotator_service_account_email
}

output "jvs_public_key_service_account_email" {
  description = "JVS public key service account email."
  value       = module.jvs_common.jvs_public_key_service_account_email
}

output "kms_keyring_id" {
  description = "KMS keyring for JVS."
  value       = module.jvs_common.kms_keyring_id
}
