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

output "workload_identity_provider_name" {
  value = module.github_action[0].workload_identity_provider_name
}

output "jvs_server_url" {
  value = module.e2e.jvs_server_url
}

output "public_key_server_url" {
  value = module.e2e.public_key_server_url
}

output "cert_rotator_server_url" {
  value = module.e2e.cert_rotator_server_url
}
