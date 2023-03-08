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

resource "google_iap_brand" "project_brand" {
  project           = var.project_id
  support_email     = var.iap_support_email
  application_title = "JVS UI"

  depends_on = [
    google_project_service.services["iap.googleapis.com"]
  ]
}

resource "google_iap_client" "project_client" {
  display_name = "JVS IAP Client"
  brand        = google_iap_brand.project_brand.name

  depends_on = [
    google_project_service.services["iap.googleapis.com"]
  ]
}

resource "google_iap_web_iam_member" "member" {
  for_each = toset(var.jvs_invoker_members)

  project = var.project_id
  member  = each.key
  role    = "roles/iap.httpsResourceAccessor"
}

# Allow allUsers to invoke the UI. This is safe because the service is behind
# GCLB + IAP and only allows internal + load balancer ingress.
#
# Per https://cloud.google.com/iap/docs/enabling-cloud-run#known_limitations,
# Cloud Run must have allUsers as the invoker to be fronted by IAP. Once IAP for
# Cloud Run is GA, we should change this to grant the IAP SA permission to
# invoke the Cloud Run service.
resource "google_cloud_run_service_iam_member" "iap_invoker" {
  location = var.region
  project  = var.project_id
  service  = module.jvs_services.jvs_ui_service_name
  role     = "roles/run.invoker"
  member   = "allUsers"
}
