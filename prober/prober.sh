#!/usr/bin/env bash

set -eEuo pipefail

ID_TOKEN=$(curl -sf "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity?audience=$AUDIENCE" -H "Metadata-Flavor: Google")

./jvsctl token create --auth-token $ID_TOKEN -e "jvs_prober" > jvs_token

cat jvs_token | ./jvsctl token validate --token -
