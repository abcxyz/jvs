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
  type        = string
  description = "The GCP project to host the justification verification service."
}

variable "region" {
  type        = string
  default     = "us-central1"
  description = "The default Google Cloud region to deploy resources in (defaults to 'us-central1')."
}

variable "service_ingress" {
  type        = string
  default     = "all"
  description = "The ingress settings for user facing services, possible values: all, internal, internal-and-cloud-load-balancing (defaults to 'all')."
}

variable "jvs_api_service_account" {
  description = "Service account for JVS API service."
  type        = string
}

variable "jvs_api_service_image" {
  description = "Container image for JVS API service."
  type        = string
  default     = "gcr.io/cloudrun/hello:latest"
}

variable "jvs_ui_service_account" {
  description = "Service account for JVS UI service."
  type        = string
}

variable "jvs_ui_service_image" {
  description = "Container image for JVS UI service."
  type        = string
  default     = "gcr.io/cloudrun/hello:latest"
}

variable "jvs_cert_rotator_service_account" {
  description = "Service account for JVS cert rotator service."
  type        = string
}

variable "jvs_cert_rotator_service_image" {
  description = "Container image for JVS cert rotator service."
  type        = string
  default     = "gcr.io/cloudrun/hello:latest"
}

variable "jvs_public_key_service_account" {
  description = "Service account for JVS public key service."
  type        = string
}

variable "jvs_public_key_service_image" {
  description = "Container image for JVS public key service."
  type        = string
  default     = "gcr.io/cloudrun/hello:latest"
}

variable "kms_keyring_id" {
  description = "KMS keyring to create signing keys."
  type        = string
}

variable "kms_key_name" {
  description = "KMS key name for use with signing."
  type        = string
}

variable "kms_key_rotation_minutes" {
  type        = number
  default     = 5
  description = "Cadence (expressed in minutes) to run the certificate rotator on. If set to 0, key rotation won't be scheduled."
}

variable "cert_rotator_envvars" {
  description = "Env vars for JVS cert rotator service."
  type        = map(string)
  default = {
    "KEY_TTL" : "10m",
    "GRACE_PERIOD" : "5m",
    "DISABLED_PERIOD" : "5m",
    "PROPAGATION_DELAY" : "2m",
  }
}

variable "public_key_envvars" {
  description = "Env vars for JVS public key service."
  type        = map(string)
  default = {
    "CACHE_TIMEOUT" : "10m",
  }
}

variable "ui_envvars" {
  description = "Env vars for JVS UI service."
  type        = map(string)
  default = {
    "DEV_MODE"  = "false",
    "ALLOWLIST" = "*",
  }
}

variable "public_key_invokers" {
  description = "Public key service invokers. It is meant to be public, therefore it is allUsers by default."
  type        = list(string)
  default = ["allUsers"]
}
