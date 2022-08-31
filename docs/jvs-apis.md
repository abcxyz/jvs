# JVS APIs

**JVS is not an official Google product.**

## Justification API

### API Spec

Justification API is a [gRPC](https://grpc.io/) service. It accepts
[CreateJustificationRequests](https://github.com/abcxyz/jvs/blob/main/protos/v0/jvs_request.proto#L23-L28)
and responses with signed justification tokens as JWTs. See
[JVSService](https://github.com/abcxyz/jvs/blame/e718d4664467b880991b8e2a400070c2aa93a0b9/blob/main/protos/v0/jvs_service.proto)
for details.

### Setup Knobs

Justification API loads configs from environment variables. See
[JustificationConfig](https://github.com/abcxyz/jvs/blob/main/pkg/config/justification_config.go#L32-L49)
for details of supported config env variables.

## Public Key API

### API Spec

Public Key API exposes a JWKS endpoint which is found at
`${PUBLIC_KEY_SERVER_URL}/.well-known/jwks`. This endpoint will contain the JWK
used to verify all Auth0-issued JWTs. Refer to
[JWKs](https://auth0.com/docs/secure/tokens/json-web-tokens/json-web-key-sets).

### Setup Knobs

Currently, Public Key API loads in the config with the env variables specified
on the host. See
[PublicKeyConfig](https://github.com/abcxyz/jvs/blob/main/pkg/config/public_key_config.go#L26-L35)
for details of supported config env variables.

## Cert Rotation API

### API Spec

Cert Rotation API will do the following based on multiple conditions, 
see [RotateKey](https://github.com/abcxyz/jvs/blob/main/pkg/jvscrypto/rotation_handler.go#L42-L80) for details:
* Create new key versions
* Set the new primary key version
* Disable or delete old key versions

The service is meant to be triggered by [Cloud Scheduler](https://cloud.google.com/scheduler) job.

### Setup Knobs

Currently, Cert Rotation API loads in the config with the env variables
specified on the host. See
[CryptoConfig](https://github.com/abcxyz/jvs/blob/main/pkg/config/crypto_config.go#L31-L51)
for details of supported config env variables.
