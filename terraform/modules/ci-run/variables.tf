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

variable "project_id" {
  type        = string
  description = "The GCP project to host the justification verification service."
}

variable "key_ring" {
  type        = string
  description = "The id of the keyring to be used by the JVS"
}

variable "key_ttl" {
  type        = string
  default     = "10m"
  description = "the length of time that we expect a key to be valid for"
}

variable "key_grace_period" {
  type        = string
  default     = "5m"
  description = "length of time between when we rotate the key and when an old Key Version is no longer valid and available"
}

variable "key_disabled_period" {
  type        = string
  default     = "5m"
  description = "length of time between when the key is disabled and when we delete the key"
}

variable "key_propagation_delay" {
  type        = string
  default     = "2m"
  description = "length of time that it takes for a change in the key in KMS to be reflected in the client"
}

variable "jvs_service_account" {
  type        = string
  description = "The service account email for the JVS to use"
}

variable "rotator_service_account" {
  type        = string
  description = "The service account email for the cert rotator to use"
}
