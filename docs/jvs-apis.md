# JVS APIs

**JVS is not an official Google product.**

## Try JVS APIs
Refer to [JVS Setup](./jvs-setup.md) for a quick JVS APIs setup.

### Justification API
1. Export the domain part of the `public_key_server_url` from Terraform outputs like `jvs-e2e-xxxx-uc.a.run.app`
```shell
export JVS_SERVER_DOMAIN=<jvs_server_domain> 
```
2. Create Justification Token via command:
```shell
grpcurl -import-path ../.. -proto protos/v0/jvs_service.proto \
-H "Authorization: Bearer $(gcloud auth print-identity-token )" \
-d '{"justifications": [{"category": "explanation", "value": "this is a test"}], "ttl": "3600s"}' \
-max-msg-sz 9999999999 \
${JVS_SERVER_DOMAIN}:443 \
abcxyz.jvs.JVSService/CreateJustification
```
You should see output similar to follows
```shell
{
  "token": "eyJhbGciOiJFUzI1NiIsImtpZCI6InByb2plY3RzL3hpeXVlLWp2cy1kb2MtdGVzdC0xL2xvY2F0aW9ucy9nbG
  9iYWwva2V5UmluZ3MvY2kta2V5cmluZy9jcnlwdG9LZXlzL2p2cy1rZXkvY3J5cHRvS2V5VmVyc2lvbnMvNyIsInR5cCI6Ik
  pXVCJ9.eyJhdWQiOiJUT0RPICMyMiIsImV4cCI6MTY2MDg2Mjg3OCwianRpIjoiNGJkODY1ZDItOWNkOS00M2NhLWJhMTQtY
  TA1Y2VlNzlmMmI0IiwiaWF0IjoxNjYwODU5Mjc4LCJpc3MiOiJqdnMuYWJjeHl6LmRldiIsIm5iZiI6MTY2MDg1OTI3OCwic3
  ViIjoieGl5dWVAZ29vZ2xlLmNvbSIsImp1c3RzIjpbeyJjYXRlZ29yeSI6ImV4cGxhbmF0aW9uIiwidmFsdWUiOiJ0aGlzIGl
  zIGEgdGVzdCJ9XX0.6BaM4HHM7lqAIuo-NW4oRt67mYD2jPojtrIK7Nxv2ARL6NIpcx5v1y86tGF1jETTV7nhfXxal0DOe4GFk
  _Xq5Q"
}
```
### Public Key API
1. Export the `public_key_server_url` from Terraform outputs
```shell
export PUBLIC_KEY_SERVER_URL=<public_key_server_url>
```
2. Fetch public keys via command:
```shell
curl -H "Authorization: Bearer $(gcloud auth print-identity-token )" \
"${PUBLIC_KEY_SERVER_URL}/.well-known/jwks"  
```
You should see output similar to follows
```shell
{"keys":[{"crv":"P-256","kid":"projects/test-project/locations/global/keyRings/ci-keyring/cryptoKeys/jvs-key/cryptoKeyVersions/1",
"kty":"EC","x":"u4SVWCYAZtD8J9r4bc150doTctTviIltS215qKkw8bF","y":"E3zbf_rvi7jTQykxcyUZqerXo_ssS6auvwR6mLchLll"},
{"crv":"P-256","kid":"projects/test-project/locations/global/keyRings/ci-keyring/cryptoKeys/jvs-key/cryptoKeyVersions/2",
"kty":"EC","x":"L4tcY2n2qKngEsLzatLXE_iTK39hUg18bE27H-r_p_M","y":"S0TrLBOPhyw7guoEIR2LSU6tLhelHLE3pZ4XaEJnzLN"}]}
```

### Cert Rotation API
1. Export the `cert_rotator_server_url` from Terraform outputs
```shell
export CERT_ROTATOR_SERVER_URL=<cert_rotator_server_url>
```
2. Rotate keys via command:
```shell
curl -H "Authorization: Bearer $(gcloud auth print-identity-token )" \
"${CERT_ROTATOR_SERVER_URL}"  
```
You should see output similar to follows
```shell
finished with all keys successfully.
```
