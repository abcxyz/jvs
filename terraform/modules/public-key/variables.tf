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

variable "region" {
  type        = string
  default     = "us-central1"
  description = "The default region for resources in the project; individual resources could have more specific variables defined to specify their region/location"
}

variable "project_id" {
  type        = string
  description = "The GCP project to host the justification verification service."
}

variable "artifact_registry_location" {
  type        = string
  default     = "us"
  description = "The artifact registry location."
}

variable "key_id" {
  type        = string
  description = "kms key id for use with signing"
}

variable "service_account" {
  type        = string
  description = "The service account email address to be used by the JVS"
}

variable "service_image" {
  type        = string
  description = "The public key service image."
}

variable "service_name" {
  type        = string
  default     = "jvs"
  description = "The name for the service."
}

variable "cache-timeout" {
  type        = string
  default     = "10m"
  description = "Duration before cache entries are invalided"
}
