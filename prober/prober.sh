set -eEuo pipefail

ID_TOKEN=$(curl -sF "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity?audience=$AUDIENCE" -H "Metadata-Flavor: Google")

./jvsctl token --auth-token $ID_TOKEN -e "jvs_prober" > jvs_token

cat jvs_token | ./jvsctl validate --token -
