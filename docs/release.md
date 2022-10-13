Reserve space for cshou.

# Build & Release

**JVS is not an official Google product.**

## Build

We leverage [goreleaser] to build JVS images. In repo root:

```sh
# The container registry for the images.
CONTAINER_REGISTRY=us-docker.pkg.dev/my-project/images

# Use goreleaser to build the images.
goreleaser release --snapshot --rm-dist
```

By default, the images will be tagged in the format of `{{ .Version
}}-SNAPSHOT-{{.ShortCommit}}` per
[goreleaser format](https://goreleaser.com/customization/snapshots/). To
override that, set the env var `TAG_OVERRIDE`.

```sh
TAG_OVERRIDE=my-tag goreleaser release --snapshot --rm-dist
```

goreleaser won't push the images in snapshot mode. Run the follow script to push
all the images:

```sh
./scripts/docker_push.sh
```
