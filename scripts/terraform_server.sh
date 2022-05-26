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


SERVICE_NAME=jvs-${RANDOM}
GO_BUILD_COMMAND=${ROOT}/pkg/scripts/build.sh

GCLOUD_ACCOUNT=$(gcloud config get-value account)
ID_TOKEN=$(gcloud auth print-identity-token)

JVS_DIR=${ROOT}/terraform/modules/jvs-service
JVS_PROJECT_ID=jvs-service

terraform -chdir=${JVS_DIR} init
terraform -chdir=${JVS_DIR} apply -auto-approve \
  -var="project_id=${JVS_PROJECT_ID}" \
  -var="service_name=${SERVICE_NAME}"
