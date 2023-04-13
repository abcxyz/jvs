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

resource "google_cloud_run_v2_job" "jvs_prober" {

  project = var.project_id

  name = "jvs-prober"

  location = "us-central1"

  template {

    template {

      containers {
        image = var.jvs_prober_image

        env {
          name  = "JVSCTL_SERVER_ADDRESS"
          value = var.prober_jvs_api_address
        }

        env {
          name  = "JVSCTL_JWKS_ENDPOINT"
          value = var.prober_jvs_public_key_endpoint
        }

        env {
          name  = "AUDIENCE"
          value = var.prober_audience
        }
      }
      service_account = resource.google_service_account.prober_service_account.email
    }
  }

  lifecycle {
    ignore_changes = [
      launch_stage,
    ]
  }
}

# This is the service account which will be used by scheduler_job
# to trigger jvs-prober cloud run job.
resource "google_service_account" "prober_service_account" {
  project = var.project_id

  account_id   = "jvs-prober"
  display_name = "Prober Service Account"
}

# Grant jvs-prober cloud run invoker role.
resource "google_project_iam_member" "cloudrun_invoker" {
  project = var.project_id

  role   = "roles/run.invoker"
  member = resource.google_service_account.prober_service_account.member
}

# This is the scheduler for triggering jvs-prober cloud run job
# in a user defined frequency.
resource "google_cloud_scheduler_job" "job" {

  project = var.project_id

  schedule    = var.prober_scheduler
  name        = "jvs-prober-scheduler"
  description = "prober cloud run job scheduler"
  region      = "us-central1"

  retry_config {
    retry_count = 0
  }

  http_target {
    http_method = "POST"
    uri         = "https://${resource.google_cloud_run_v2_job.jvs_prober.location}-run.googleapis.com/apis/run.googleapis.com/v1/namespaces/${var.project_id}/jobs/${resource.google_cloud_run_v2_job.jvs_prober.name}:run"

    oauth_token {
      service_account_email = resource.google_service_account.prober_service_account.email
    }
  }

  depends_on = [resource.google_cloud_run_v2_job.jvs_prober]
}

# This alert will trigger if: in a user defined rolling window size, the number
# # of failed jvs-prober cloud job runs exceed the user defined threshold.
resource "google_monitoring_alert_policy" "prober_service_failed_number_exceed_threshold" {
  project = var.project_id

  display_name = "JVS-Prober Service Alert: Too many failed JVS probes"

  combiner = "OR"

  # Conditions are:
  # 1. The metric is completed_execution_count
  # 2. The metrics is applied to jvs-prober
  # 3. Only count on failed jobs.
  # 4. When the failed jobs exceed the threshold,
  #    alert will be triggered.
  conditions {
    display_name = "Too many failed JVS probes"
    condition_threshold {
      filter          = "metric.type=\"run.googleapis.com/job/completed_execution_count\" resource.type=\"cloud_run_job\" resource.label.\"job_name\"=\"${resource.google_cloud_run_v2_job.jvs_prober.name}\" AND metric.label.\"result\"=\"failed\""
      duration        = "0s"
      comparison      = "COMPARISON_GT"
      threshold_value = var.prober_alert_threshold
      aggregations {
        alignment_period   = var.prober_alert_align_window_size_in_seconds
        per_series_aligner = "ALIGN_COUNT"
      }
    }
  }
  notification_channels = [
    resource.google_monitoring_notification_channel.email_notification_channel.name
  ]

  enabled = var.prober_alert_enabled
}
