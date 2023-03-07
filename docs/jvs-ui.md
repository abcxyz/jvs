# JVS UI

**JVS is not an official Google product.**

[JVS UI](../cmd/ui) facilitates the justification verification flow using a UI

## Environment Variables

The UI has the following environment variables: `PORT`, `ALLOWLIST`, and `DEV_MODE`

```shell
## default is 9091
PORT="1010" 
```

```shell
## default is false
DEV_MODE="true"
```

```shell
# A semi-colon separated string denoting the allowed domains and/or subdomains. This field is required.
ALLOWLIST="example.com;foo.bar.com"

# To allow all domain do the following
ALLOWLIST="*"
```

Setting `DEV_MODE` to `true` will allow any local IP to be accepted as a valid origin. 

TODO add steps for using a client library to interact with the UI
