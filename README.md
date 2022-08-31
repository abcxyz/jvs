# Justification Verification

**JVS is not an official Google product.**

This repository contains components related to a justification verification
service

## Use Case

Audit logs are special logs that record `when` and `who` called `which`
application and accessed `what` data. And `why` the access was necessary (aka.
the justification of the data access). JVS is a solution to produce verified
justifications and in combination with
[abcxyz/lumberjack](https://github.com/abcxyz/lumberjack) the justifications
could be audit logged.

## Components

JVS consists of the following components:

*   JVS APIs
    *   [Justification API](./cmd/justification): verify justifications and mint
        short-lived justification tokens.*
    *   [Cert Rotator API](./cmd/cert-rotation): rotate signing
        [keys](https://cloud.google.com/kms/docs/key-rotation).
    *   [Public Key API](./cmd/public-key):
        [JWKs](https://auth0.com/docs/secure/tokens/json-web-tokens/json-web-key-sets)
*   [CLI Tool](./cmd/jvsctl)

See manuals for [JVS APIs usage](./docs/jvs-apis.md) and
[JVS CLI Tool Usage](./docs/cli-tool.md)

**TODO(#115):** add a simple diagram to describe the user experience flow of
enabling JVS in Lumberjack.
