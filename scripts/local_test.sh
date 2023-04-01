#!/bin/bash
# Copyright 2023 The Authors (see AUTHORS file)
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

set -eEuo pipefail

# Set the project where the cloud resources are located.
export PROJECT_ID=

# Set REGISTRY to where you want to upload the docker images such as
# "us-docker.pkg.dev/jvs-ci-test/ci-images"
export REGISTRY=

# Set BUILD_COMMON to true if you need to create cloud run service accounts and
# KMS keyring.
export BUILD_COMMON=true

# Set below variables only if BUILD_COMMON is set to false.
export API_SA=
export UI_SA=
export CERT_ROTATOR_SA=
export PUBLIC_KEY_SA=
export KMS_KEYRING_ID=

chmod +x ./scripts/build.sh
chmod +x ./scripts/integration.sh

./scripts/build.sh && ./scripts/integration.sh
