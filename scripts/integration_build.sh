#!/bin/bash
# Copyright 2022 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

ROOT="$(cd "$(dirname "$0")/.." &>/dev/null; pwd -P)"

JVS_SERVICE_ACCOUNT="jvs-service-sa@jvs-service-ci.iam.gserviceaccount.com"
KEY_ID="projects/jvs-service-ci/locations/global/keyRings/jvs-keyring/cryptoKeys/jvs-key"
PROJECT_ID="jvs-service-ci"
SERVICE_NAME="jvs-${RANDOM}"

JVS_DIR=${ROOT}/terraform/modules/ci-run

cd $JVS_DIR
terraform init
terraform apply -auto-approve \
  -var="project_id=${PROJECT_ID}" \
  -var="service_name=${SERVICE_NAME}" \
  -var="jvs_service_account=${JVS_SERVICE_ACCOUNT}" \
  -var="key_id=${KEY_ID}"
