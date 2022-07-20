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
set -u # -u makes bash error on undefined variables
ROOT="$(cd "$(dirname "$0")/.." &>/dev/null; pwd -P)"

printf "Argument project_id is %s\n" "${PROJECT_ID}"
printf "Argument keyring_id is %s\n" "${KEYRING_ID}"

export TEST_JVS_KMS_KEY_RING="projects/${PROJECT_ID}/locations/global/keyRings/${KEYRING_ID}"
export TEST_JVS_INTEGRATION=true
export TEST_JVS_FIRESTORE_PROJECT_ID=${PROJECT_ID}

cd ${ROOT}
go test ./test/integ/...
