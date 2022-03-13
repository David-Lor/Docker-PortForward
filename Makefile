.DEFAULT_GOAL := help

IMAGE_NAME := "local/portforward:latest"

build: ## docker build the image
	docker build . --pull -t "${IMAGE_NAME}"

test-integration: ## run integration tests (running a built image)
	cd forwarder && PORTFORWARD_IMAGE="${IMAGE_NAME}" go test -v -run "Integration"

test-unit: ## run unit tests
	cd forwarder && go test -v -short

help: ## show this help.
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'
