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

variable "alert_enabled" {
  type        = bool
  description = "True if alerts are enabled, otherwise false."
  default     = false
}

variable "cert_rotator_5xx_response_threshold" {
  type        = number
  description = "Send alert for Cert-Rotator when the number of response with 5xx code exceeds the threshold."
  default     = 5
}

variable "cert_rotator_latency_threshold_ms" {
  type        = number
  description = "Send alert when UI-Service's latency (in ms) exceed the threshold."
  default     = 5000
}

variable "public_key_service_5xx_response_threshold" {
  type        = number
  description = "Send alert for Public-Key-Service when the number of response with 5xx code exceeds the threshold."
  default     = 5
}

variable "public_key_service_latency_threshold_ms" {
  type        = number
  description = "Send alert for UI-Service's latency (in ms) exceed the threshold."
  default     = 5000
}

variable "jvs_service_5xx_response_threshold" {
  type        = number
  description = "Send alert for Justification-Service when the number of response with 5xx code exceeds the threshold."
  default     = 5
}

variable "jvs_service_latency_threshold_ms" {
  type        = number
  description = "Send alert for UI-Service's latency (in ms) exceed the threshold."
  default     = 5000
}

variable "ui_service_5xx_response_threshold" {
  type        = number
  description = "Send alert for UI-Service when the number of response with 5xx code exceeds the threshold."
  default     = 5
}

variable "ui_service_latency_threshold_ms" {
  type        = number
  description = "Send alert for UI-Service's latency (in ms) exceed the threshold."
  default     = 5000
}

variable "jvs_prober_image" {
  type        = string
  description = "docker image for jvs-prober"
}

variable "prober_jvs_api_address" {
  type        = string
  description = "jvs service address"
}

variable "prober_jvs_public_key_endpoint" {
  type        = string
  description = "jvs public key service address"
}

variable "prober_audience" {
  type        = string
  description = "The cloud run url for jvs api service or app address."
}

variable "prober_scheduler" {
  type        = string
  description = "How often the prober service should be triggered, default is every 10 minutes. Learn more at: https://cloud.google.com/scheduler/docs/configuring/cron-job-schedules?&_ga=2.26495481.-578386315.1680561063#defining_the_job_schedule."
  default     = "*/10 * * * *"
}

variable "prober_alert_align_window_size_in_seconds" {
  type        = string
  description = "The sliding window size for counting failed prober job runs. Format example: 600s."
  default     = "3600s"
}

variable "prober_alert_threshold" {
  type        = number
  description = "Send alert for Prober-Service when the number of failed prober runs exceeds the threshold."
  default     = 4
}
