resource "google_monitoring_notification_channel" "basic" {
  project = var.project_id

  display_name = "owner notification channel"

  type = "email"

  labels = {
    email_address = var.notification_channel_email
  }

  force_delete = false
}

resource "google_monitoring_alert_policy" "public_key_service_alert" {
  project = var.project_id

  display_name = "Public-Key Service Alert: Too many none-200 response"

  combiner = "AND"

  conditions {
    display_name = "Too many None-200 response"

    condition_threshold {
      filter          = "metric.type=\"run.googleapis.com/request_count\" resource.type=\"cloud_run_revision\" resource.label.\"service_name\"=\"${var.public_key_service_name}\" AND metric.label.\"response_code\"!=\"200\""
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
    resource.google_monitoring_notification_channel.basic.name
  ]

  alert_strategy {
    auto_close = "604800s"
  }
}

resource "google_monitoring_alert_policy" "jvs_service_alert" {
  project = var.project_id

  display_name = "Justification Service Alert: Too many none-200 response"

  combiner = "AND"

  conditions {
    display_name = "too many None-200 response"
    condition_threshold {
      filter          = "metric.type=\"run.googleapis.com/request_count\" resource.type=\"cloud_run_revision\" resource.label.\"service_name\"=\"${var.jvs_service_name}\" AND metric.label.\"response_code\"!=\"200\""
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
    resource.google_monitoring_notification_channel.basic.name
  ]
  alert_strategy {
    auto_close = "604800s"
  }
}

resource "google_monitoring_alert_policy" "cert_rotator_alert" {
  project = var.project_id

  display_name = "Cert-Rotator Alert: Too many none-200 response"

  combiner = "AND"

  conditions {
    display_name = "too many None-200 response"
    condition_threshold {
      filter          = "metric.type=\"run.googleapis.com/request_count\" resource.type=\"cloud_run_revision\" resource.label.\"service_name\"=\"${var.cert_rotation_service_name}\" AND metric.label.\"response_code\"!=\"200\""
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
    resource.google_monitoring_notification_channel.basic.name
  ]
  alert_strategy {
    auto_close = "604800s"
  }
}


resource "google_monitoring_alert_policy" "ui_service_alert" {
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
    resource.google_monitoring_notification_channel.basic.name
  ]

  alert_strategy {
    auto_close = "604800s"
  }
}