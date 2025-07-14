# Copyright (C) 2025 Crash Override, Inc.
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the FSF, either version 3 of the License, or (at your option) any later version.
# See the LICENSE file in the root of this repository for full license text or
# visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

GO_SOURCES=$(shell find . -name '*.go' -not -path './cmd/*')

UPLOADER_SOURCES=$(shell find cmd/default-uploaders -name '*.go')
CRAWLER_SOURCES=$(shell find cmd/default-crawlers -name '*.go')
DOWNLOADER_SOURCES=$(shell find cmd/default-downloaders -name '*.go')

DEP_SOURCES="go.sum go.mod"

OCULAR_ENV_FILE ?= .env

OCULAR_VERSION ?= $(shell git describe --exact-match --tags 2>/dev/null || echo "dev")
OCULAR_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
OCULAR_BUILD_TIME ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

# Only if .env file is present
ifneq (,$(wildcard ${OCULAR_ENV_FILE}))
	include ${OCULAR_ENV_FILE}
endif


OCULAR_ENVIRONMENT ?= development
BASE_LD_FLAGS := -X github.com/crashappsec/ocular-default-integrations/internal/config.Version=${OCULAR_VERSION} -X github.com/crashappsec/ocular-default-integrations/internal/config.Commit=${OCULAR_COMMIT} -X github.com/crashappsec/ocular-default-integrations/internal/config.BuildTime=${OCULAR_BUILD_TIME}
ifeq ($(OCULAR_ENVIRONMENT), production)
	LDFLAGS:= -w -s ${BASE_LD_FLAGS}
else
	LDFLAGS:= ${BASE_LD_FLAGS}
endif


define NETRC
  machine github.com
  login x-access-token
  password ${GITHUB_TOKEN}
endef

ifneq ($(GITHUB_TOKEN),)
	export NETRC
endif

# logging level debug when using make
OCULAR_LOGGING_LEVEL ?= debug

DOCKER_BUILDKIT ?= 1

ifneq ($(DOCKER_DEFAULT_PLATFORM),)
	export DOCKER_DEFAULT_PLATFORM
endif
OCULAR_IMAGE_REGISTRY ?= ghcr.io
OCULAR_IMAGE_TAG ?= local
OCULAR_DEFAULT_DOWNLOADER_IMAGE_REPOSITORY ?= crashappsec/ocular-default-downloaders
OCULAR_DEFAULT_CRAWLER_IMAGE_REPOSITORY ?= crashappsec/ocular-default-crawlers
OCULAR_DEFAULT_UPLOADER_IMAGE_REPOSITORY ?= crashappsec/ocular-default-uploaders

OCULAR_DEFAULT_DOWNLOADER_IMAGE ?= ${OCULAR_IMAGE_REGISTRY}/${OCULAR_DEFAULT_DOWNLOADER_IMAGE_REPOSITORY}:${OCULAR_IMAGE_TAG}
OCULAR_DEFAULT_CRAWLER_IMAGE ?= ${OCULAR_IMAGE_REGISTRY}/${OCULAR_DEFAULT_CRAWLER_IMAGE_REPOSITORY}:${OCULAR_IMAGE_TAG}
OCULAR_DEFAULT_UPLOADER_IMAGE ?= ${OCULAR_IMAGE_REGISTRY}/${OCULAR_DEFAULT_UPLOADER_IMAGE_REPOSITORY}:${OCULAR_IMAGE_TAG}

export

.PHONY: all clean
all: build-docker

clean:
	@echo "Cleaning up build artifacts ..."
	@rm -rf bin
	@rm -f coverage.out

#########
# Build #
#########

.PHONY: build build-uploaders build-downloader build-crawler
build: build-uploaders build-downloader build-crawler

build-uploaders: bin/default-uploaders
build-crawler: bin/default-crawlers
build-downloader: bin/default-downloaders

bin/default-downloaders: cmd/default-downloaders/main.go $(DOWNLOADER_SOURCES) $(GO_SOURCES)
	@go build -o $@ -ldflags='${LDFLAGS}' $<

bin/default-crawlers: cmd/default-crawlers/main.go $(CRAWLER_SOURCES) $(GO_SOURCES)
	@go build -o $@ -ldflags='${LDFLAGS}' $<

bin/default-uploaders: cmd/default-uploaders/main.go $(UPLOADER_SOURCES) $(GO_SOURCES)
	@go build -o $@ -ldflags='${LDFLAGS}' $<

################
# Docker Build #
################

.PHONY: docker-build docker-build-downloaders docker-build-crawlers docker-build-uploaders
build-docker: generate
	@docker compose build

build-docker-downloaders:
	@docker compose build default-downloaders

build-docker-crawlers:
	@docker compose build default-crawlers

build-docker-uploaders:
	@docker compose build default-uploaders

##############
# Publishing #
##############

.PHONY: push-docker
push-docker: build-docker
	@docker compose push

###############
# Development #
###############

.PHONY: generate lint fmt test view-test-coverage fmt-code fmt-license
generate:
	@echo "Generating code ..."
	@OCULAR_LOGGING_LEVEL=error go generate ./...
	@$(MAKE) fmt-license # generated source code files will not have license headers, so we need to run fmt-license after generate

lint:
	@echo "Running linters ..."
	@golangci-lint run ./... --timeout 10m

fmt: generate fmt-code

fmt-license:
	@echo "Formatting license headers ..."
	@license-eye header fix

fmt-code:
	@echo "Running code formatters ..."
	@golangci-lint fmt ./...

test:
	@echo "Running unit tests ..."
	@go test $$(go list ./... | grep -v /internal/unittest) -coverprofile=coverage.out -covermode=atomic

view-test-coverage: test
	@go tool cover -html=coverage.out

serve-docs:
	@command -v godoc > /dev/null 2>&1 || (echo "Please install godoc using 'go install golang.org/x/tools/cmd/godoc@latest'" && exit 1)
	@echo "Serving documentation at http://localhost:6060/pkg/github.com/crashappsec/ocular/"
	@godoc -http=localhost:6060