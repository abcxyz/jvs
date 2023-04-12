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

variable "region" {
  type        = string
  default     = "us-central1"
  description = "The default Google Cloud region to deploy resources in (defaults to 'us-central1')."
}

variable "jvs_invoker_members" {
  description = "The list of members that can call JVS."
  type        = list(string)
  default     = []
}

variable "jvs_api_domain" {
  description = "The JVS API domain."
  type        = string
}

variable "jvs_ui_domain" {
  description = "The JVS UI domain."
  type        = string
}

variable "iap_support_email" {
  description = "The IAP support email."
  type        = string
}

variable "kms_key_location" {
  type        = string
  default     = "global"
  description = "The location where kms key will be created."
}

variable "jvs_container_image" {
  description = "Container image for the jvsctl CLI and server entrypoints."
  type        = string
}

variable "notification_channel_email" {
  type        = string
  description = "The Email address where alert notifications send to."
}

variable "jvs_prober_image" {
  type        = string
  description = "docker image for jvs-prober"
}

variable "prober_audience" {
  type        = string
  description = "The cloud run url for jvs api service or app address."
}
