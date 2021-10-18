SHELL=/usr/bin/env bash -o pipefail

# IMAGE_BASE defines the registry/namespace and part of the image name
# This variable is used to construct full image tags for bundle and catalog images.
IMAGE_BASE ?= monitoring-stack-operator


VERSION ?= $(shell cat VERSION)
OPERATOR_IMG = $(IMAGE_BASE):$(VERSION)

# running `make` builds the operator (default target)
all: operator

## Tools
TOOLS_DIR = $(shell pwd)/tmp/bin
CONTROLLER_GEN=$(TOOLS_DIR)/controller-gen
GOLANGCI_LINT=$(TOOLS_DIR)/golangci-lint
KUSTOMIZE=$(TOOLS_DIR)/kustomize
OPERATOR_SDK = $(TOOLS_DIR)/operator-sdk
OPM = $(TOOLS_DIR)/opm

$(TOOLS_DIR):
	@mkdir -p $(TOOLS_DIR)

.PHONY: controller-gen
$(CONTROLLER_GEN) controller-gen: $(TOOLS_DIR)
	@{ \
		set -ex ;\
		GOBIN=$(TOOLS_DIR) go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.7.0 ;\
	}

.PHONY: golangci-lint
$(GOLANGCI_LINT) golangci-lint: $(TOOLS_DIR)
	@{ \
		set -ex ;\
		GOBIN=$(TOOLS_DIR) go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.42.1 ;\
	}

# NOTE: kustomize does not support `go install` hence this workaround to install
# it by creating an tmp module and using go get to download the precise version
# needed for the project
# see: https://github.com/kubernetes-sigs/kustomize/issues/3618
.PHONY: kustomize
$(KUSTOMIZE) kustomize: $(TOOLS_DIR)
	@{ \
		set -ex ;\
		[[ -f $(KUSTOMIZE) ]] && exit 0 ;\
		TMP_DIR=$$(mktemp -d) ;\
		cd $$TMP_DIR ;\
		go mod init tmp ;\
		echo "Downloading kustomize" ;\
		GOBIN=$(TOOLS_DIR) go get sigs.k8s.io/kustomize/kustomize/v3@v3.9.4 ;\
		rm -rf $$TMP_DIR ;\
	}

.PHONY: operator-sdk
$(OPERATOR_SDK) operator-sdk: $(TOOLS_DIR)
	@{ \
		set -ex ;\
		[[ -f $(OPERATOR_SDK) ]] && exit 0 ;\
		OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
		curl -sSLo $(OPERATOR_SDK) https://github.com/operator-framework/operator-sdk/releases/download/v1.13.0/operator-sdk_$${OS}_$${ARCH} ;\
		chmod +x $(OPERATOR_SDK) ;\
	}

.PHONY: opm
$(OPM) opm: $(TOOLS_DIR)
	@{ \
		set -ex ;\
		OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
		curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/v1.15.1/$${OS}-$${ARCH}-opm ;\
		chmod +x $(OPM) ;\
	}

# Install all required tools
.PHONY: tools
tools: $(CONTROLLER_GEN) $(KUSTOMIZE) $(OPERATOR_SDK) $(OPM)

## Development

.PHONY: lint
lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run ./... --fix

.PHONY: generate-crds
generate-crds: $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) crd \
		paths=./pkg/apis/... \
		rbac:roleName=monitoring-stack-operator \
		output:dir=. \
		output:rbac:dir=./deploy/operator \
		output:crd:dir=./deploy/crds

.PHONY: generate-kustomize
generate-kustomize: $(KUSTOMIZE)
	cd deploy/operator && \
		$(KUSTOMIZE) edit set image monitoring-stack-operator=*:$(VERSION)

.PHONY: generate-deepcopy
generate-deepcopy: $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./pkg/apis/..."

.PHONY: generate
generate: generate-crds generate-deepcopy generate-kustomize

operator: generate
	go build -o ./tmp/operator ./cmd/operator/...


.PHONY: operator-image
operator-image: generate
	docker build -f build/Dockerfile . -t $(OPERATOR_IMG)

.PHONY: operator-push
operator-push:
	docker push ${OPERATOR_IMG}

.PHONY: test-e2e
test-e2e:
	go test ./test/e2e/...

## OLM - Bundle

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= $(IMAGE_BASE)-bundle:$(VERSION)

# CHANNELS define the bundle channels used in the bundle.
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=candidate,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="candidate,fast,stable")
CHANNELS ?= development
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle.
# To re-generate a bundle for any other default channel without changing the default setup, use:
# - DEFAULT_CHANNEL as arg of the bundle target (e.g make bundle DEFAULT_CHANNEL=stable)
# - environment variables to overwrite this value (e.g export DEFAULT_CHANNEL="stable")
DEFAULT_CHANNEL ?= development

ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)


.PHONY: bundle
bundle: $(KUSTOMIZE) $(OPERATOR_SDK) generate
	cd deploy/olm && \
		$(KUSTOMIZE) edit set image monitoring-stack-operator=$(OPERATOR_IMG)

	$(KUSTOMIZE) build deploy/olm | tee tmp/pre-bundle.yaml |  \
	 	$(OPERATOR_SDK) generate bundle \
			--overwrite \
		 	--version $(VERSION) \
			--kustomize-dir=deploy/olm \
			--package=monitoring-stack-operator \
		 	$(BUNDLE_METADATA_OPTS)
	$(OPERATOR_SDK) bundle validate ./bundle

.PHONY: bundle-image
bundle-image: bundle ## Build the bundle image.
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Build the bundle image.
	docker push $(BUNDLE_IMG)

# A comma-separated list of bundle images e.g.
# make catalog-build BUNDLE_IMGS=example.com/operator-bundle:v0.1.0,example.com/operator-bundle:v0.2.0).
#
# NOTE: These images MUST exist in a registry and be pull-able.
BUNDLE_IMGS ?= $(BUNDLE_IMG)

# The image tag given to the resulting catalog image
# The tag is used as latest since it allows a CatalogSubscription to point to
# a single image which keeps updating
CATALOG_IMG ?= $(IMAGE_BASE)-catalog:latest
# enable continuous release by referring to the same catalog image for `--from-index`
CATALOG_BASE_IMG ?= $(CATALOG_IMG)

# mark release as first by default for easier/quicker development
FIRST_OLM_RELEASE ?= true

# Set CATALOG_BASE_IMG to an existing catalog image tag to add $BUNDLE_IMGS to
# that image except for FIRST_OLM_RELEASE
ifeq ($(FIRST_OLM_RELEASE), false)
FROM_INDEX_OPT := --from-index $(CATALOG_BASE_IMG)
endif

# Build a catalog image by adding bundle images to an empty catalog using the
# operator package manager tool, 'opm'.
#
# NOTE: This recipe invokes 'opm' in 'semver' bundle add mode. For more information
# on add modes, see:
# https://github.com/operator-framework/community-operators/blob/7f1438c/docs/packaging-operator.md#updating-your-existing-operator
.PHONY: catalog-image
catalog-image: $(OPM)
	$(OPM) index add \
	 	--container-tool docker \
		--mode semver \
		--tag $(CATALOG_IMG) \
		--bundles $(BUNDLE_IMGS) $(FROM_INDEX_OPT)

# Push the catalog image.
.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	docker push $(CATALOG_IMG)

.PHONY: release
release: operator-image operator-push bundle-image bundle-push catalog-image catalog-push

STANDARD_VERSION=$(TOOLS_DIR)/standard-version
$(STANDARD_VERSION):
	npm install -g --prefix tmp standard-version

.PHONY: initiate-release
initiate-release: $(STANDARD_VERSION)
	git fetch git@github.com:rhobs/monitoring-stack-operator.git --tags
	$(STANDARD_VERSION) --skip.tag # The tag will be created in the pipeline

.PHONY: initiate-release-as
initiate-release-as: $(STANDARD_VERSION)
	git fetch git@github.com:rhobs/monitoring-stack-operator.git --tags
	$(STANDARD_VERSION) --skip.tag --release-as $(RELEASE_VERSION)
