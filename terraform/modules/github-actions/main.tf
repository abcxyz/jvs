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
locals {
  repo = "abcxyz/jvs"
}

resource "google_service_account" "gh-access-acc" {
  project      = var.project_id
  account_id   = "gh-access-sa"
  display_name = "GitHub Access Account"
}

resource "google_iam_workload_identity_pool" "pool" {
  provider                  = google-beta
  project                   = var.project_id
  workload_identity_pool_id = "github-pool"
  description               = "Github pool"
}

resource "google_iam_workload_identity_pool_provider" "provider" {
  provider                           = google-beta
  project                            = var.project_id
  workload_identity_pool_id          = google_iam_workload_identity_pool.pool.workload_identity_pool_id
  workload_identity_pool_provider_id = "github-provider"
  display_name                       = "Github provider"
  attribute_mapping                  = {
    "attribute.aud"        = "assertion.aud"
    "attribute.actor"      = "assertion.actor"
    "google.subject"       = "assertion.sub"
    "attribute.repository" = "assertion.repository"
  }
  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }
}

resource "google_service_account_iam_member" "external_provider_roles" {
  service_account_id = google_service_account.gh-access-acc.name
  role               = "roles/iam.workloadIdentityUser",
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.pool.id}/attribute.repository/${local.repo}"
}

resource "google_project_iam_member" "gh_access_acc_iam" {
  project = var.project_id
  role    = "roles/owner"
  member  = "serviceAccount:${google_service_account.gh-access-acc.email}"
}