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

JVS_SERVICE_ACCOUNT="jvs-service-sa@jvs-ci.iam.gserviceaccount.com"
ROTATOR_SERVICE_ACCOUNT="rotator-sa@jvs-ci.iam.gserviceaccount.com"
PUB_KEY_SERVICE_ACCOUNT="pubkey-sa@jvs-ci.iam.gserviceaccount.com"
PROJECT_ID="jvs-ci"

CI_DIR=${ROOT}/terraform/modules/ci-run

cd $CI_DIR
terraform init
terraform apply -auto-approve \
  -var="project_id=${PROJECT_ID}" \
  -var="jvs_service_account=${JVS_SERVICE_ACCOUNT}" \
  -var="rotator_service_account=${ROTATOR_SERVICE_ACCOUNT}" \
  -var="public_key_service_account=${PUB_KEY_SERVICE_ACCOUNT}"

export TEST_JVS_KMS_KEY_RING=$(terraform output key_ring)
export TEST_JVS_INTEGRATION=true

cd ${ROOT}
go test ./test/integ/...
