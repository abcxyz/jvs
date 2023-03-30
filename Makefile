# `make build` builds the service images. Docker and goreleaser must be
# installed to run this command.
build:
	chmod +x ./scripts/build.sh
	./scripts/build.sh

# `make integration` deploys the images to cloud resources and runs integration
# tests. Terraform and go must be installed to run this command
integration:
	chmod +x ./scripts/integration.sh
	./scripts/integration.sh

build-and-integration: build integration
