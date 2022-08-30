.PHONY: build push gotest gobuild

IMAGE_NAME := quay.io/app-sre/git-sync-pull
IMAGE_TAG := $(shell git rev-parse --short=7 HEAD)
DOCKER_CONF := $(CURDIR)/.docker
GOOS := $(shell go env GOOS)
PWD := $(shell pwd)

gotest:
	CGO_ENABLED=0 GOOS=$(GOOS) go test ./...

gobuild: gotest
	CGO_ENABLED=0 GOOS=$(GOOS) go build -a -installsuffix cgo -o git-sync-pull ./cmd/git-sync-pull

build:
	@docker build --no-cache -t $(IMAGE_NAME):$(IMAGE_TAG) .

push:
	@docker tag $(IMAGE_NAME):$(IMAGE_TAG) $(IMAGE_NAME):latest
	@docker --config=$(DOCKER_CONF) push $(IMAGE_NAME):$(IMAGE_TAG)
	@docker --config=$(DOCKER_CONF) push $(IMAGE_NAME):latest