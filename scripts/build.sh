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

# .goreleaser.docker.yaml and integration.sh read DOCKER_TAG
export DOCKER_TAG=$(git rev-parse --short HEAD)

REGISTRY_HOST=`echo $REGISTRY | awk -F '/' '{ print $1}'`

# goreleaser requires a tag to publish images to container registry.
# We create a local tag to make it happy.
git tag -f `date "+%Y%m%d%H%M%S"`

# Configures Docker to authenticate to Artifact Registry hosts.
gcloud auth configure-docker $REGISTRY_HOST

# Build docker images.
goreleaser release -f .goreleaser.docker.yaml --rm-dist
