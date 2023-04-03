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
  description = "The GCP project to host the dashboards."
}

variable "jvs_service_name" {
  type        = string
  description = "The Justficaition Verification Cloud Run service to monitor."
}

variable "cert_rotator_service_name" {
  type        = string
  description = "The Cert Rotation Cloud Run service to monitor."
}

variable "public_key_service_name" {
  type        = string
  description = "The Public Key Cloud Run service to monitor."
}

variable "jvs_ui_service_name" {
  type        = string
  description = "The JVS-UI Cloud Run service to monitor."
}

variable "notification_channel_email" {
  type        = string
  description = "The Email address where alert notifications send to."
}

variable "is_prod_env" {
  type        = bool
  description = "If deploying to a prod environment, true if is prod env, otherwise false."
  default     = false
}

variable "cert_rotator_5xx_response_threshold" {
  type        = number
  description = "Send alert for Cert-Rotator when the number of response with 5xx code exceeds the threshold"
  default     = 5
}

variable "cert_rotator_latency_threshold" {
  type        = number
  description = "Send alert when UI-Service's latency (in ms) exceed the threshold"
  default     = 5000
}

variable "public_key_service_5xx_response_threshold" {
  type        = number
  description = "Send alert for Public-Key-Service when the number of response with 5xx code exceeds the threshold"
  default     = 5
}

variable "public_key_service_latency_threshold" {
  type        = number
  description = "Send alert for UI-Service's latency (in ms) exceed the threshold"
  default     = 5000
}

variable "jvs_service_5xx_response_threshold" {
  type        = number
  description = "Send alert for Justification-Service when the number of response with 5xx code exceeds the threshold"
  default     = 5
}

variable "jvs_service_latency_threshold" {
  type        = number
  description = "Send alert for UI-Service's latency (in ms) exceed the threshold"
  default     = 5000
}

variable "ui_service_5xx_response_threshold" {
  type        = number
  description = "Send alert for UI-Service when the number of response with 5xx code exceeds the threshold"
  default     = 5
}

variable "ui_service_latency_threshold" {
  type        = number
  description = "Send alert for UI-Service's latency (in ms) exceed the threshold"
  default     = 5000
}
