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
  github_owner_id = 93787867  # abcxyz
  github_repo_id  = 479173136 # abcxyz/jvs
}

resource "google_project_service" "services" {
  for_each = toset([
    "cloudkms.googleapis.com",
  ])

  project = var.project_id

  service            = each.value
  disable_on_destroy = false
}

// IAM roles needed to run tests.
resource "google_project_iam_member" "gh_access_acc_iam" {
  for_each = toset(var.ci_iam_roles)

  project = var.project_id

  role   = each.key
  member = module.github_ci_infra.service_account_member
}

module "github_ci_infra" {
  source = "git::https://github.com/abcxyz/terraform-modules.git//modules/github_ci_infra?ref=46d3ffd82d7c3080bc5ec2cc788fe3e21176a8be"

  project_id = var.project_id

  name                 = "jvs"
  github_repository_id = local.github_repo_id
  github_owner_id      = local.github_owner_id
}

module "jvs_common" {
  source = "../modules/common"

  project_id = var.project_id

  kms_key_location = var.kms_key_location
}
