# Release

**JVS is not an official Google product.**

We leverage [goreleaser](https://goreleaser.com/) for both container image
release and SCM (GitHub) release. Due to `goreleaser` limitation, we have to
split the two releases into two config files:

-   `.goreleaser.docker.yaml` for container image release
-   `.goreleaser.yaml` (default) for SCM (GitHub) release

## Release Workflow

TODO: Once we add the release workflow, document the most common steps to create
a new release. It should only include pushing a new tag to remote.

## Manually Release Images

Or if you want to build/push images for local development.

```sh
# By default we use JVS CI container registry for the images.
# To override, set the following env var.
# CONTAINER_REGISTRY=us-docker.pkg.dev/my-project/images

# goreleaser expects a "clean" repo to release so commit any local changes if
# needed.
git add . && git commit -m "local changes"

# goreleaser expects a tag.
# The tag must be a semantic version https://semver.org/
# DON'T push the tag if you're not releasing.
git tag -f -a v0.0.0-$(git rev-parse --short HEAD)

# Use goreleaser to build the images.
# It should in the end push all the images to the given container registry.
# All the images will be tagged with the git tag given earlier.
goreleaser release -f .goreleaser.docker.yaml --rm-dist
```
