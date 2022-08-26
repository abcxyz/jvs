# CLI Tool

**JVS is not an official Google product.**

[jvsctl](../cmd/jvsctl) facilitates the justification verification flow provided
by abcxyz/jvs

## Install
1. Change directory to where jvsctl code lives
   ```shell
   cd cmd/jvsctl
   ```

2. Install jvsctl
   ```shell
   go install
   ```

## Usage

jvsctl [command]

### Global Flags:

Run `jvsctl -h` for details

### Config
`jvsctl` reads config from a yaml file (default path: `$HOME/.jvsctl/config.yaml`). 
Basic config fields can be provided (or overwritten) by global flags above like `--server` and ` --insecure`.

1. Version - the version of the config
```yaml
version: 1
```

2. Server - the JVS server address
```yaml
server: example.com
```

3. Authentication - the authentication config

```yaml
authentication:
   # insecure indiates whether to use insecured connection to the JVS server.
   insecure: true
```

## Command
### jvsctl token [flags]
To generate a justification token

#### Flags

Run `jvsctl token -h` for details

#### Example

```shell
jvsctl token --explanation "issues/12345" --ttl 30m
```
The example above generates a signed justification token with 30m time-to-live duration
and such token provide reasons that data access is required for "issues/12345"

```shell
jvsctl token --breakglass true --explanation "jvs is down" --ttl 30m
```
The example above generates an unsigned justification token with 30m time-to-live duration
and such token provide reasons that break-glass data access is required because "jvs is down"