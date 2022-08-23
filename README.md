# Justification Verification

**JVS is not an official Google product.**

This repository contains components related to a justification verification
service

## Use Case

Enable companies to audit usage of user data by creating a mechanism for users
to provide justifications when accessing data.

## Components

JVS consists of the following components:

* JVS APIs
  * [Justification API](./cmd/justification)
  * [Cert Rotator API](./cmd/cert-rotation)
  * [Public Key API](./cmd/public-key)
* [CLI Tool](./cmd/jvsctl)

See manuals for [JVS APIs usage](./docs/jvs-apis.md) and
[JVS CLI Tool Usage](./docs/cli-tool.md)
