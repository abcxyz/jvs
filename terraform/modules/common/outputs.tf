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

output "jvs_api_service_account_email" {
  description = "JVS API service account email."
  value       = google_service_account.api_acc.email
}

output "jvs_api_service_account_member" {
  description = "JVS API service account member."
  value       = google_service_account.api_acc.member
}

output "jvs_ui_service_account_email" {
  description = "JVS UI service account email."
  value       = google_service_account.ui_acc.email
}

output "jvs_ui_service_account_member" {
  description = "JVS UI service account member."
  value       = google_service_account.ui_acc.member
}

output "jvs_cert_rotator_service_account_email" {
  description = "JVS cert rotator service account email."
  value       = google_service_account.rotator_acc.email
}

output "jvs_cert_rotator_service_account_member" {
  description = "JVS cert rotator service account member."
  value       = google_service_account.rotator_acc.member
}

output "jvs_public_key_service_account_email" {
  description = "JVS public key service account email."
  value       = google_service_account.public_key_acc.email
}

output "jvs_public_key_service_account_member" {
  description = "JVS public key service account member."
  value       = google_service_account.public_key_acc.member
}

output "kms_keyring_id" {
  value = google_kms_key_ring.keyring.id
}
