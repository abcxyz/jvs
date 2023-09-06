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

variable "kms_key_location" {
  type        = string
  default     = "global"
  description = "The location where kms key will be created."
}

variable "ci_iam_roles" {
  type = list(string)
  default = [
    # To deploy and invoke cloud run services.
    "roles/iam.serviceAccountUser",
    "roles/run.admin",

    # To operate KMS.
    "roles/cloudkms.admin",
    "roles/cloudkms.cryptoOperator",

    # To read and edit project service during CI.
    "roles/serviceusage.serviceUsageAdmin",

    # To set project IAM policies.
    "roles/resourcemanager.projectIamAdmin",
  ]
  description = "List of IAM roles needed to run integration tests included in CI/CD."
}

variable "region" {
  description = "The default Google Cloud region to deploy resources in (defaults to 'us-central1')."
  type        = string
  default     = "us-central1"
}

variable "jvs_container_image" {
  description = "Container image for the jvsctl CLI and server entrypoints."
  type        = string
}

variable "plugin_envvars" {
  description = "Env vars for plugin."
  type        = map(string)
  default     = {}
}

variable "registry_repository_id" {
  description = "name for artifact registry."
  type        = string
}

variable "ci_service_account_email" {
  description = "service account email."
  type        = string
}
