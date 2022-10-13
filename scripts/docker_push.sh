#!/usr/bin/env bash
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

set -eEuo pipefail

if [ -z "${CONTAINER_REGISTRY:-}" ]; then
  echo "Missing CONTAINER_REGISTRY!" >&2
  exit 1
fi

# Push all local JVS images to the given container registry.
docker image push --all-tags ${CONTAINER_REGISTRY}/jvs-cert-rotation
docker image push --all-tags ${CONTAINER_REGISTRY}/jvs-justification
docker image push --all-tags ${CONTAINER_REGISTRY}/jvs-public-key

# Add other images built by the goreleaser here.
