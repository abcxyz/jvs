# CLI Tool

**JVS is not an official Google product.**

[jvsctl](../cmd/jvsctl) facilitates the justification verification flow provided
by abcxyz/jvs

## Installation

```sh
go install github.com/abcxyz/jvs/cmd/jvsctl
```

Or download from a
[release](https://github.com/abcxyz/jvs/releases).

## Usage

jvsctl [command]

Run `jvsctl -h` for details of available flags.

## Config

The `jvsctl` CLI accepts flags for all its configuration. To avoid repetition,
some flags can be defined as environment variables. For example, to always use
the same justification server for minting tokens, add the following to your
`.bashrc` or `.zshrc` file:

```shell
export JVSCTL_SERVER_ADDRESS="https://jvs.corp.internal:8080"
```

Similarly, you can set the endpoint for getting the JWKS for verification:

```shell
export JVSCTL_JWKS_ENDPOINT="https://keys.corp.internal:8080/.well-known/jwks"
```

For the full list of options that correspond to your release, check the help
output. Append `-h` to any command or subcommand to see detailed usage
instructions:

```shell
jvsctl -h
jvsctl token -h
```

## Authentication

If you installed JVS using the provided Terraform module as described in the
[README](../README.md#installation), you can use
[`gcloud`](https://cloud.google.com/sdk/gcloud) generated ID token for
authentication. E.g.

```sh
jvsctl token create --auth-token $(gcloud auth print-identity-token) -e "just testing"
```
