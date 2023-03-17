# CLI Tool

**JVS is not an official Google product.**

[jvsctl](../cmd/jvsctl) facilitates the justification verification flow provided
by abcxyz/jvs

## Installation

```sh
go install github.com/abcxyz/jvs/cmd/jvsctl
```

Or download from a
[release](https://github.com/abcxyz/lumberjack/releases/tag/v0.0.4).

## Usage

jvsctl [command]

Run `jvsctl -h` for details of available flags.

## Config

By default, `jvsctl` expects the config file at `~/.jvsctl/config.yaml`.
Minimally we need a JVS server address in the config file to mint justification
tokens.

```yaml
server: example.com:4567
```

By default, we will connect to the JVS server securely. When it's not applicable
(e.g. locally run JVS server for testing), use insecure connection by adding the
following block in the config:

```yaml
insecure: true
```

JWKS endpoint is also required if you want to validate justification tokens, it
will default to the server domain + `/.well-known/jwks` if it is not specified.

```yaml
jwks_endpoint: https://example.com/.well-known/jwks
```

Alternatively, all of these values could be provided via CLI flags `--server`,
`--insecure`, and `--jwks_endpoint`.

## Authentication

If you installed JVS using the provided Terraform module as described in the
[README](../README.md#installation), you can use
[`gcloud`](https://cloud.google.com/sdk/gcloud) generated ID token for
authentication. E.g.

```sh
jvsctl token --auth-token $(gcloud auth print-identity-token) -e "just testing"
```
