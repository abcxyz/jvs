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
  description = "The GCP project to host the audit logging service."
}

variable "service_name" {
  type        = string
  default     = "audit-logging"
  description = "The name for the audit logging server service."
}

variable "artifact_registry_location" {
  type        = string
  default     = "us"
  description = "The artifact registry location."
}

variable "folder_parent" {
  type        = string
  description = "The parent to hold the environment. E.g. organizations/102291006291 or folders/300968597098"
}

variable "top_folder_id" {
  type        = string
  description = "The top folder name to hold all the e2e resources."
}

variable "billing_account" {
  type        = string
  description = "The billing account to be linked to projects."
}
