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

output "jvs_api_service_url" {
  description = "JVS API service url."
  value       = module.api_cloud_run.url
}

output "jvs_ui_service_url" {
  description = "JVS UI service url."
  value       = module.ui_cloud_run.url
}

output "jvs_cert_rotator_service_url" {
  description = "JVS cert rotator service url."
  value       = module.cert_rotator_cloud_run.url
}

output "jvs_public_key_service_url" {
  description = "JVS public key service url."
  value       = module.public_key_cloud_run.url
}
