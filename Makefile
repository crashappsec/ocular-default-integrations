# Copyright (C) 2025-2026 Crash Override, Inc.
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the FSF, either version 3 of the License, or (at your option) any later version.
# See the LICENSE file in the root of this repository for full license text or
# visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

OCULAR_ENV_FILE ?= .env

# Only if .env file is present
ifneq (,$(wildcard ${OCULAR_ENV_FILE}))
	include ${OCULAR_ENV_FILE}
endif


ifneq ($(DOCKER_DEFAULT_PLATFORM),)
	export DOCKER_DEFAULT_PLATFORM
endif

OCULAR_DEFAULTS_VERSION ?= latest
export OCULAR_DEFAULTS_VERSION
OCULAR_UPLOADERS_IMG ?= ghcr.io/crashappsec/ocular-default-uploaders:$(OCULAR_DEFAULTS_VERSION)
OCULAR_DOWNLOADERS_IMG ?= ghcr.io/crashappsec/ocular-default-downloaders:$(OCULAR_DEFAULTS_VERSION)
OCULAR_CRAWLERS_IMG ?= ghcr.io/crashappsec/ocular-default-crawlers:$(OCULAR_DEFAULTS_VERSION)
export OCULAR_UPLOADERS_IMG
export OCULAR_DOWNLOADERS_IMG
export OCULAR_CRAWLERS_IMG

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= docker

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# ld flags passed to go build
LDFLAGS ?= -X main.version=$(OCULAR_VERSION) -X main.buildTime=$(shell date -Iseconds) -X main.gitCommit=$(shell git rev-parse --short HEAD)


.PHONY: all clean
all: docker-build-all

clean:
	@echo "Cleaning up build artifacts ..."
	@rm -rf bin
	@rm -f coverage.out


##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)


##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | $(KUBECTL) apply -f -

deploy-%: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config
	$(KUSTOMIZE) build config/$(@:deploy-%=%) | $(KUBECTL) apply -f -

.PHONY: undeploy
undeploy: kustomize ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -


undeploy-%: kustomize ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/$(@:undeploy-%=%)  | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -


manifests: ## Generate manifests e.g. CRD, RBAC etc.
	@$(MAKE) generate
	@# empty command, since we are not using controller-gen to generate manifests
	@# but in order to keep the Makefile structure we leave this target here

##@ Development

.PHONY: generate lint fmt test view-test-coverage fmt-code fmt-license
generate:
	@echo "Generating code ..."
	@go generate ./...
	@$(MAKE) lint-fix


.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet setup-envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test $$(go list ./... | grep -v /e2e) -coverprofile cover.out

# The default setup assumes Kind is pre-installed and builds/loads the Manager Docker image locally.
# CertManager is installed by default; skip with:
# - CERT_MANAGER_INSTALL_SKIP=true
KIND_CLUSTER ?= ocular-test-e2e

.PHONY: setup-test-e2e
setup-test-e2e: ## Set up a Kind cluster for e2e tests if it does not exist
	@command -v $(KIND) >/dev/null 2>&1 || { \
		echo "Kind is not installed. Please install Kind manually."; \
		exit 1; \
	}
	@case "$$($(KIND) get clusters)" in \
		*"$(KIND_CLUSTER)"*) \
			echo "Kind cluster '$(KIND_CLUSTER)' already exists. Skipping creation." ;; \
		*) \
			echo "Creating Kind cluster '$(KIND_CLUSTER)'..."; \
			$(KIND) create cluster --name $(KIND_CLUSTER) ;; \
	esac

.PHONY: test-e2e
test-e2e: setup-test-e2e manifests generate fmt vet ## Run the e2e tests. Expected an isolated environment using Kind.
	KIND_CLUSTER=$(KIND_CLUSTER) go test ./test/e2e/ -v -ginkgo.v
	$(MAKE) cleanup-test-e2e

.PHONY: cleanup-test-e2e
cleanup-test-e2e: ## Tear down the Kind cluster used for e2e tests
	@$(KIND) delete cluster --name $(KIND_CLUSTER)

.PHONY: lint
lint: golangci-lint license-eye ## Run golangci-lint linter
	$(GOLANGCI_LINT) run
	$(LICENSE_EYE) header check

.PHONY: lint-fix
lint-fix: golangci-lint license-eye ## Run golangci-lint linter and perform fixes
	$(GOLANGCI_LINT) run --fix
	$(LICENSE_EYE) header fix

.PHONY: lint-config
lint-config: golangci-lint ## Verify golangci-lint linter configuration
	$(GOLANGCI_LINT) config verify

##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	go build -o bin/manager cmd/manager/main.go


# PLATFORMS is a list of platforms to
# build for. Production Ocular images are built
# with: 'linux/arm64,linux/amd64,linux/s390x,linux/ppc64le'
PLATFORMS=linux/arm64,linux/amd64

# Additionally, docker args can be set,
# adding --push will push the image
DOCKER_ARGS ?= --platform=$(PLATFORMS)

LDFLAGS ?= -X main.version=$(OCULAR_DEFAULTS_VERSION) -X main.buildTime=$(shell date -Iseconds) -X main.gitCommit=$(shell git rev-parse --short HEAD)
# If you wish to build the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build-all
docker-build-all: docker-build-img-downloaders docker-build-img-crawlers docker-build-img-uploaders ## Build all integration images

docker-build-img-%: ## Builds the docker image
	@$(CONTAINER_TOOL) build \
		--build-arg LDFLAGS="$(LDFLAGS)" \
		--build-arg INTEGRATION=$* \
		--tag $(OCULAR_$(shell echo '$*' | tr '[:lower:]' '[:upper:]')_IMG) \
		$(DOCKER_ARGS) \
		-f Dockerfile .

.PHONY: docker-push-all
docker-push-all: docker-push-img-downloaders docker-push-img-uploaders docker-push-img-crawlers ## Push docker image with the manager.

.PHONY: docker-push-downloaders
docker-push-downloaders: ## Push docker image with the manager.
	$(CONTAINER_TOOL) push ${OCULAR_DOWNLOADERS_IMG}

.PHONY: docker-push-crawlers
docker-push-crawlers: ## Push docker image with the manager.
	$(CONTAINER_TOOL) push ${OCULAR_CRAWLERS_IMG}

.PHONY: docker-push-uploaders
docker-push-uploaders: ## Push docker image with the manager.
	$(CONTAINER_TOOL) push ${OCULAR_UPLOADERS_IMG}


docker-push-img-%: ## Push docker image with the manager.
	$(CONTAINER_TOOL) push $(OCULAR_$(shell echo '$*' | tr '[:lower:]' '[:upper:]')_IMG)

.PHONY: build-installer
build-installer: manifests generate kustomize ## Generate a consolidated YAML with CRDs and deployment.
	@mkdir -p dist
	@$(KUSTOMIZE) build config/default > dist/install.yaml

build-installer-%: manifests generate kustomize ## Generate a consolidated YAML with CRDs and deployment.
	@mkdir -p dist
	@$(KUSTOMIZE) build config/$(@:build-installer-%=%) > dist/install-$(@:build-installer-%=%).yaml

.PHONY: build-helm
build-helm: manifests generate kustomize ## Generate a helm chart at dist/chart
	@./hack/scripts/generate-helm-chart.sh

.PHONY: clean-helm
clean-helm: ## Clean up the helm chart generated files
	@rm -rf dist/chart

##@ Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUBECTL ?= kubectl
KIND ?= kind
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint
YQ ?= $(LOCALBIN)/yq
CLIENT_GEN ?= $(LOCALBIN)/client-gen
LICENSE_EYE ?= $(LOCALBIN)/license-eye


## Tool Versions
KUSTOMIZE_VERSION ?= v5.6.0
CONTROLLER_TOOLS_VERSION ?= v0.18.0
#ENVTEST_VERSION is the version of controller-runtime release branch to fetch the envtest setup script (i.e. release-0.20)
ENVTEST_VERSION ?= $(shell go list -m -f "{{ .Version }}" sigs.k8s.io/controller-runtime | awk -F'[v.]' '{printf "release-%d.%d", $$2, $$3}')
#ENVTEST_K8S_VERSION is the version of Kubernetes to use for setting up ENVTEST binaries (i.e. 1.31)
ENVTEST_K8S_VERSION ?= $(shell go list -m -f "{{ .Version }}" k8s.io/api | awk -F'[v.]' '{printf "1.%d", $$3}')
GOLANGCI_LINT_VERSION ?= v2.8.0
YQ_VERSION ?= v4.47.1
CODE_GENERATOR_VERSION ?= v0.34.0
LICENSE_EYE_VERSION ?= v0.8.0

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

.PHONY: setup-envtest
setup-envtest: envtest ## Download the binaries required for ENVTEST in the local bin directory.
	@echo "Setting up envtest binaries for Kubernetes version $(ENVTEST_K8S_VERSION)..."
	@$(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path || { \
		echo "Error: Failed to set up envtest binaries for version $(ENVTEST_K8S_VERSION)."; \
		exit 1; \
	}

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

yq: $(YQ) ## Download yq locally if necessary.
$(YQ): $(LOCALBIN)
	$(call go-install-tool,$(YQ),github.com/mikefarah/yq/v4,$(YQ_VERSION))

.PHONY: client-gen
client-gen: $(CLIENT_GEN) ## Download code-generator locally if necessary.
$(CLIENT_GEN): $(LOCALBIN)
	$(call go-install-tool,$(CLIENT_GEN),k8s.io/code-generator/cmd/client-gen,$(CODE_GENERATOR_VERSION))

.PHONY: license-eye
license-eye: $(LICENSE_EYE) ## Download skywalking-eyes locally if necessary.
$(LICENSE_EYE): $(LOCALBIN)
	$(call go-install-tool,$(LICENSE_EYE),github.com/apache/skywalking-eyes/cmd/license-eye,$(LICENSE_EYE_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef
