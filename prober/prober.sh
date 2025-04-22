#!/usr/bin/env bash

set -eEuo pipefail

# Expected to be set externally.
declare -r AUDIENCE

ID_TOKEN=$(curl -sf "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity?audience=${AUDIENCE}" -H "Metadata-Flavor: Google")

./jvsctl token create --auth-token "${ID_TOKEN}" -e "jvs_prober" > jvs_token

# shellcheck disable=SC2002
cat jvs_token | ./jvsctl token validate --token -
