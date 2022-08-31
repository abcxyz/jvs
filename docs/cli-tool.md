# CLI Tool

**JVS is not an official Google product.**

[jvsctl](../cmd/jvsctl) facilitates the justification verification flow provided
by abcxyz/jvs

## Install

1.  Install jvsctl

    ```shell
    go install github.com/abcxyz/jvs/cmd/jvsctl
    ```

## Usage

jvsctl [command]

Run `jvsctl -h` for details of available flags.

### Config

Minimally we need a JVS server address in the config to mint justification
tokens.

```yaml
server: example.com
```

By default, we will connect to the JVS server securely. When it's not applicable
(e.g. locally run JVS server for testing), use insecure connection by adding the
following block in the config:

```yaml
authentication:
  insecure: true
```

Alternatively, both of these values could be provided via CLI flags `--server`
and `--insecure`.

## Command

### jvsctl token [flags]

To generate a justification token

#### Flags

Run `jvsctl token -h` for details

#### Example

```shell
jvsctl token --explanation "issues/12345" --ttl 30m
```

The example above generates a signed justification token with 30m time-to-live
duration and such token provide reasons that data access is required for
"issues/12345"

```shell
jvsctl token --breakglass true --explanation "jvs is down" --ttl 30m
```

In certain cases, we might need to bypass JVS for minting signed JVS tokens.
E.g. JVS couldn't verify a justification because the ticket system is down. In
such cases, we can mint break-glass token instead.
