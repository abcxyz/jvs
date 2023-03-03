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

variable "project_id" {
  description = "The GCP project to host the justification verification service."
  type        = string
}

variable "kms_keyring_name" {
  description = "Keyring name."
  type        = string
  default     = "jvs-keyring"
}

variable "kms_key_location" {
  description = "The location where kms key will be created."
  type        = string
  default     = "global"
}

variable "jvs_api_service_account_name" {
  description = "Name for JVS API service."
  type        = string
  default     = "jvs-api"
}

variable "jvs_ui_service_account_name" {
  description = "Name for JVS UI service."
  type        = string
  default     = "jvs-ui"
}

variable "jvs_cert_rotator_service_account_name" {
  description = "Name for JVS cert rotator service."
  type        = string
  default     = "jvs-cert-rotator"
}


variable "jvs_public_key_service_account_name" {
  description = "Name for JVS public key service."
  type        = string
  default     = "jvs-public-key"
}
