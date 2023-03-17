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

resource "google_compute_global_address" "default" {
  project = var.project_id

  name       = "jvs-${random_id.default.hex}-address" # 63 character limit
  ip_version = "IPV4"

  depends_on = [
    google_project_service.services["compute.googleapis.com"],
  ]
}

resource "google_compute_global_forwarding_rule" "http" {
  project = var.project_id

  name                  = "jvs-${random_id.default.hex}-http" # 63 character limit
  target                = google_compute_target_http_proxy.default.self_link
  ip_address            = google_compute_global_address.default.address
  port_range            = "80"
  load_balancing_scheme = "EXTERNAL"
}

resource "google_compute_global_forwarding_rule" "https" {
  project = var.project_id

  name                  = "jvs-${random_id.default.hex}-https" # 63 character limit
  target                = google_compute_target_https_proxy.default.self_link
  ip_address            = google_compute_global_address.default.address
  port_range            = "443"
  load_balancing_scheme = "EXTERNAL"
}

resource "google_compute_managed_ssl_certificate" "default" {
  project = var.project_id

  name = "jvs-${random_id.default.hex}-cert" # 63 character limit

  managed {
    domains = toset([var.jvs_api_domain, var.jvs_ui_domain])
  }

  depends_on = [
    google_project_service.services["compute.googleapis.com"],
  ]
  lifecycle {
    create_before_destroy = true
  }
}

resource "google_compute_url_map" "default" {
  project = var.project_id

  name            = "jvs-${random_id.default.hex}-url-map" # 63 character limit
  default_service = google_compute_backend_service.jvs_api_backend.self_link

  host_rule {
    hosts        = [var.jvs_api_domain]
    path_matcher = "jvs-api"
  }

  host_rule {
    hosts        = [var.jvs_ui_domain]
    path_matcher = "jvs-ui"
  }

  path_matcher {
    name            = "jvs-api"
    default_service = google_compute_backend_service.jvs_api_backend.self_link

    path_rule {
      paths   = ["/.well-known/jwks"]
      service = google_compute_backend_service.jvs_public_key_backend.self_link
    }
  }

  path_matcher {
    name            = "jvs-ui"
    default_service = google_compute_backend_service.jvs_ui_backend.self_link
  }
}

resource "google_compute_url_map" "https_redirect" {
  project = var.project_id

  name = "jvs-${random_id.default.hex}-https-redirect" # 63 character limit
  default_url_redirect {
    https_redirect         = true
    redirect_response_code = "MOVED_PERMANENTLY_DEFAULT"
    strip_query            = false
  }

  depends_on = [
    google_project_service.services["compute.googleapis.com"],
  ]
}

resource "google_compute_target_http_proxy" "default" {
  project = var.project_id

  name = "jvs-${random_id.default.hex}-http-proxy" # 63 character limit

  url_map = google_compute_url_map.https_redirect.self_link
}

resource "google_compute_target_https_proxy" "default" {
  project = var.project_id

  name    = "jvs-${random_id.default.hex}-https-proxy" # 63 character limit
  url_map = google_compute_url_map.default.self_link

  ssl_certificates = [google_compute_managed_ssl_certificate.default.self_link]
}

resource "google_compute_region_network_endpoint_group" "jvs_api_neg" {
  project = var.project_id

  region                = var.region
  name                  = "jvs-api-${random_id.default.hex}-neg" # 63 character limit
  network_endpoint_type = "SERVERLESS"

  cloud_run {
    service = module.jvs_services.jvs_api_service_name
  }

  depends_on = [
    google_project_service.services["compute.googleapis.com"],
  ]
}

resource "google_compute_backend_service" "jvs_api_backend" {
  project = var.project_id

  name                  = "jvs-api-${random_id.default.hex}-backend" # 63 character limit
  load_balancing_scheme = "EXTERNAL"
  description           = "jvs-api backend"

  backend {
    description = "jvs-api serverless backend group"
    group       = google_compute_region_network_endpoint_group.jvs_api_neg.id
  }

  log_config {
    enable      = true
    sample_rate = "1.0"
  }
}

resource "google_compute_region_network_endpoint_group" "jvs_ui_neg" {
  project = var.project_id

  region                = var.region
  name                  = "jvs-ui-${random_id.default.hex}-neg" # 63 character limit
  network_endpoint_type = "SERVERLESS"

  cloud_run {
    service = module.jvs_services.jvs_ui_service_name
  }

  depends_on = [
    google_project_service.services["compute.googleapis.com"],
  ]
}

resource "google_compute_backend_service" "jvs_ui_backend" {
  project = var.project_id

  name                  = "jvs-ui-${random_id.default.hex}-backend" # 63 character limit
  load_balancing_scheme = "EXTERNAL"
  description           = "jvs-ui backend"

  backend {
    description = "jvs-ui serverless backend group"
    group       = google_compute_region_network_endpoint_group.jvs_ui_neg.id
  }

  log_config {
    enable      = true
    sample_rate = "1.0"
  }

  iap {
    oauth2_client_id     = google_iap_client.project_client.client_id
    oauth2_client_secret = google_iap_client.project_client.secret
  }
}

resource "google_compute_region_network_endpoint_group" "jvs_public_key_neg" {
  project = var.project_id

  region                = var.region
  name                  = "jvs-public-key-${random_id.default.hex}-neg" # 63 character limit
  network_endpoint_type = "SERVERLESS"

  cloud_run {
    service = module.jvs_services.jvs_public_key_service_name
  }

  depends_on = [
    google_project_service.services["compute.googleapis.com"],
  ]
}

resource "google_compute_backend_service" "jvs_public_key_backend" {
  project = var.project_id

  name                  = "jvs-public-key-${random_id.default.hex}-backend" # 63 character limit
  load_balancing_scheme = "EXTERNAL"
  description           = "jvs-public-key backend"

  backend {
    description = "jvs-public-key serverless backend group"
    group       = google_compute_region_network_endpoint_group.jvs_public_key_neg.id
  }

  log_config {
    enable      = true
    sample_rate = "1.0"
  }
}
