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

variable "artifact_registry_location" {
  type        = string
  default     = "us"
  description = "The artifact registry location."
}

variable "e2e_test_iam_roles" {
  type        = list(string)
  default     = []
  description = "List of IAM roles needed to run e2e tests included in CI/CD."
}

variable "jvs_image" {
  type        = string
  description = "The JVS service image."
}

variable "cert_rotation_image" {
  type        = string
  description = "The cert rotation service image."
}

variable "public_key_image" {
  type        = string
  description = "The public key service image."
}
