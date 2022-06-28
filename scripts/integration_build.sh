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

# TODO: change it to jvs-test later
while getopts ":p:k:" opt; do
  case $opt in
    p) project_id="$OPTARG"
    ;;
    k) keyring_id="$OPTARG"
    ;;
    \?) echo "Invalid option -$OPTARG" >&2
    exit 1
    ;;
  esac

  case $OPTARG in
    -*) echo "Option $opt needs a valid argument"
    exit 1
    ;;
  esac
done

printf "Argument project_id is %s\n" "$project_id"
printf "Argument keyring_id is %s\n" "$keyring_id"
export TEST_JVS_KMS_KEY_RING="projects/${project_id}/locations/global/keyRings/${keyring_id}"
export TEST_JVS_INTEGRATION=true

cd ${ROOT}
go test ./test/integ/...
