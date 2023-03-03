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

variable "jvs_api_service_name" {
  description = "Name for JVS API service."
  type        = string
  default     = "jvs-api"
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

variable "jvs_api_service_iam" {
  description = "IAM member bindings for JVS API service."
  type = object({
    admins     = list(string)
    developers = list(string)
    invokers   = list(string)
  })
  default = {
    admins     = []
    developers = []
    invokers   = []
  }
}

variable "jvs_ui_service_name" {
  description = "Name for JVS UI service."
  type        = string
  default     = "jvs-ui"
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

variable "jvs_ui_service_iam" {
  description = "IAM member bindings for JVS UI service."
  type = object({
    admins     = list(string)
    developers = list(string)
    invokers   = list(string)
  })
  default = {
    admins     = []
    developers = []
    invokers   = []
  }
}

variable "jvs_cert_rotator_service_name" {
  description = "Name for JVS cert rotator service."
  type        = string
  default     = "jvs-cert-rotator"
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

variable "jvs_cert_rotator_service_iam" {
  description = "IAM member bindings for JVS cert rotator service."
  type = object({
    admins     = list(string)
    developers = list(string)
    invokers   = list(string)
  })
  default = {
    admins     = []
    developers = []
    invokers   = []
  }
}

variable "jvs_public_key_service_name" {
  description = "Name for JVS public key service."
  type        = string
  default     = "jvs-public-key"
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

variable "jvs_public_key_service_iam" {
  description = "IAM member bindings for JVS public key service."
  type = object({
    admins     = list(string)
    developers = list(string)
    invokers   = list(string)
  })
  default = {
    admins     = []
    developers = []
    # Public key service is meant to be public.
    invokers = ["allUsers"]
  }
}

variable "keyring_id" {
  description = "KMS keyring to create signing keys."
  type        = string
}

variable "key_name" {
  description = "KMS key id for use with signing."
  type        = string
}

variable "key_ttl" {
  description = "the length of time that we expect a key to be valid for."
  default     = "10m"
  type        = string
}

variable "key_grace_period" {
  description = "length of time between when we rotate the key and when an old Key Version is no longer valid and available."
  type        = string
  default     = "5m"
}

variable "key_disabled_period" {
  description = "length of time between when the key is disabled and when we delete the key."
  type        = string
  default     = "5m"
}

variable "key_propagation_delay" {
  description = "length of time that it takes for a change in the key in KMS to be reflected in the client."
  type        = string
  default     = "2m"
}

variable "key_rotation_cadence" {
  type        = number
  default     = 5
  description = "Cadence (expressed in minutes) to run the certificate rotator on. If set to 0, key rotation won't be scheduled."
}

variable "key_cache_timeout" {
  type        = string
  default     = "10m"
  description = "Duration before cache entries are invalided."
}

variable "ui_dev_mode" {
  type        = string
  default     = "false"
  description = "Whether UI is in dev mode."
}

variable "ui_origin_allowlist" {
  type        = string
  default     = "*"
  description = "UI origin allowlist."
}
