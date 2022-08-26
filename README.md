# Justification Verification

**JVS is not an official Google product.**

This repository contains components related to a justification verification
service

## Use Case

Audit logs are special logs that record `when` and `who` called `which` application and accessed `what` data. 
And `why` the access was necessary (aka. the justification of the data access).
JVS is a solution to produce verified justifications and
in combination with [abcxyz/lumberjack](https://github.com/abcxyz/lumberjack) the justifications could be audit logged.

## Components

JVS consists of the following components:

* JVS APIs
  * [Justification API](./cmd/justification) <br />
    run verifications and provide a justification token. Each token will be valid for a short time.
  * [Cert Rotator API](./cmd/cert-rotation) <br />
    rotates keys' key versions. <br />
    `key`: A grouping of key versions. This stays stable through key rotations, and is how we can refer to keys over time.<br />
    `key version`: This is an actual asymmetric key pair.
  * [Public Key API](./cmd/public-key) <br />
    provide the public keys for use in validating the token's authenticity.
* [CLI Tool](./cmd/jvsctl)

See manuals for [JVS APIs usage](./docs/jvs-apis.md) and
[JVS CLI Tool Usage](./docs/cli-tool.md)
