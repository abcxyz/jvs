/**
 * Copyright 2023 Google LLC
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

resource "google_monitoring_notification_channel" "email_notification_channel" {
  project = var.project_id

  display_name = "jvs alerts email channel"

  type = "email"

  labels = {
    email_address = var.notification_channel_email
  }

  force_delete = false
}

# This alert will trigger if: in a rolling window of 60s,
# the number of response with 5xx code exceeds 5.
resource "google_monitoring_alert_policy" "public_key_service_exceed_500_response_threshold" {
  project = var.project_id

  display_name = "Public-Key Service Alert: Too many 500 responses"

  combiner = "OR"

  conditions {
    display_name = "Too many 500 responses"

    condition_threshold {
      filter          = "metric.type=\"run.googleapis.com/request_count\" resource.type=\"cloud_run_revision\" resource.label.\"service_name\"=\"${var.public_key_service_name}\" AND metric.label.\"response_code\"=monitoring.regex.full_match(\"^5[0-9][0-9]$\")"
      duration        = "0s"
      comparison      = "COMPARISON_GT"
      threshold_value = 5

      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_MEAN"
      }
    }
  }

  notification_channels = [
    resource.google_monitoring_notification_channel.email_notification_channel.name
  ]

  enabled = var.is_prod_env
}

# This alert will trigger if: in a rolling window of 60s,
# the number of response with 5xx code exceeds 5.
resource "google_monitoring_alert_policy" "jvs_service_exceed_500_response_threshold" {
  project = var.project_id

  display_name = "Justification Service Alert: Too many 500 response"

  combiner = "OR"

  conditions {
    display_name = "Too many 500 responses"
    condition_threshold {
      filter          = "metric.type=\"run.googleapis.com/request_count\" resource.type=\"cloud_run_revision\" resource.label.\"service_name\"=\"${var.jvs_service_name}\" AND metric.label.\"response_code\"=monitoring.regex.full_match(\"^5[0-9][0-9]$\")"
      duration        = "0s"
      comparison      = "COMPARISON_GT"
      threshold_value = 5
      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_MEAN"
      }
    }
  }
  notification_channels = [
    resource.google_monitoring_notification_channel.email_notification_channel.name
  ]

  enabled = var.is_prod_env
}

# This alert will trigger if: in a rolling window of 60s,
# the number of response with 5xx code exceeds 5.
resource "google_monitoring_alert_policy" "cert_rotator_service_exceed_500_response_threshold" {
  project = var.project_id

  display_name = "Cert-Rotator Alert: Too many 500 response"

  combiner = "OR"

  conditions {
    display_name = "Too many 500 response"
    condition_threshold {
      filter          = "metric.type=\"run.googleapis.com/request_count\" resource.type=\"cloud_run_revision\" resource.label.\"service_name\"=\"${var.cert_rotation_service_name}\" AND metric.label.\"response_code\"=monitoring.regex.full_match(\"^5[0-9][0-9]$\")"
      duration        = "0s"
      comparison      = "COMPARISON_GT"
      threshold_value = 5
      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_MEAN"
      }
    }
  }
  notification_channels = [
    resource.google_monitoring_notification_channel.email_notification_channel.name
  ]

  enabled = var.is_prod_env
}

# This alert will trigger if: in a rolling window of 60s,
# the 95% percentile of latency exceeds 5000ms.
resource "google_monitoring_alert_policy" "ui_service_exceed_latency_threshold" {
  project = var.project_id

  display_name = "UI Service Alert: Latency too high"

  combiner = "OR"

  conditions {
    display_name = "Latency too high"
    condition_threshold {
      filter          = "metric.type=\"run.googleapis.com/request_latencies\" resource.type=\"cloud_run_revision\" resource.label.\"service_name\"=\"${var.jvs_ui_service_name}\""
      duration        = "0s"
      comparison      = "COMPARISON_GT"
      threshold_value = 5000
      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_PERCENTILE_95"
      }
    }
  }

  notification_channels = [
    resource.google_monitoring_notification_channel.email_notification_channel.name
  ]

  enabled = var.is_prod_env
}
