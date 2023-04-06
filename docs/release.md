# Release

**JVS is not an official Google product.**

We leverage [goreleaser](https://goreleaser.com/) for both container image
release and SCM (GitHub) release. Due to `goreleaser` limitation, we have to
split the two releases into two config files:

-   `.goreleaser.docker.yaml` for container image release
-   `.goreleaser.yaml` (default) for SCM (GitHub) release

## New Release

```sh
# The version you're going to release.
REL_VER=v0.0.x

# Tag
git tag -f -a $REL_VER -m $REL_VER

# Push tag. This will trigger the release workflow.
git push origin $REL_VER
```

## Manually Release Images

Or if you want to build/push images for local development.

```sh
# Set the container registry for the images, for example:
CONTAINER_REGISTRY=us-docker.pkg.dev/my-project/images

# goreleaser expects a "clean" repo to release so commit any local changes if
# needed.
git add . && git commit -m "local changes"

# goreleaser expects a tag.
# The tag must be a semantic version https://semver.org/
# DON'T push the tag if you're not releasing.
git tag -f -a v0.0.0-$(git rev-parse --short HEAD)

# goreleaser will tag the image with the git tag, optionally, override it by:
DOCKER_TAG=mytag

# Use goreleaser to build the images.
# It should in the end push all the images to the given container registry.
# All the images will be tagged with the git tag given earlier.
goreleaser release -f .goreleaser.docker.yaml --clean
```
