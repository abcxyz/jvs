# CLI Tool

[jvsctl](../cmd/jvsctl) facilitates the justification verification flow provided
by abcxyz/jvs

## Usage

jvsctl [command]

### Global Flags:

*   `--config` : string <br />
    config file (default is $HOME/.jvsctl/config.yaml) <br />
*   `-h`, `--help` <br />
    help for jvsctl <br />
*   `--insecure` : bool <br />
    use insecure connection to JVS server <br />
*   `--server` : string <br />
    overwrite the JVS server address

### jvsctl token [flags]

To generate a justification token

#### Flags

`--breakglass` : bool <br />
Whether it will be a breakglass action <br />
`-e`, `--explanation` : string <br />
The explanation for the action <br />
`--ttl` : duration <br />
The token time-to-live duration (default 1h0m0s) <br />

#### Example

```shell
jvsctl token --explanation "issues/12345" -ttl 30m
```
