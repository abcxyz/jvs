set -eEuo pipefail

ID_TOKEN=$(curl "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity?audience=$AUDIENCE" -H "Metadata-Flavor: Google")

./jvsctl token --auth-token $ID_TOKEN -e "prober" > jvs_token

cat jvs_token | ./jvsctl validate --token -

./jvsctl token --auth-token $ID_TOKEN -e "prober" > jvs_breakglass_token

cat jvs_breakglass_token | ./jvsctl validate --token -
