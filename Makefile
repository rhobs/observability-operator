SHELL=/usr/bin/env bash -o pipefail

# IMAGE_TAG_BASE defines the registry/namespace and part of the image name
# This variable is used to construct full image tags for bundle and catalog images.
#
IMAGE_TAG_BASE ?= quay.io/sthaha/monitoring-stack-operator


VERSION ?= $(shell cat VERSION)
OPERATOR_IMAGE = $(IMAGE_TAG_BASE):$(VERSION)

# running `make` builds the operator (default target)
all: operator

## Tools
TOOLS_DIR = $(shell pwd)/tmp/bin
CONTROLLER_GEN=$(TOOLS_DIR)/controller-gen
GOLANGCI_LINT=$(TOOLS_DIR)/golangci-lint
KUSTOMIZE=$(TOOLS_DIR)/kustomize
OPERATOR_SDK = $(TOOLS_DIR)/operator-sdk

$(TOOLS_DIR):
	@mkdir -p $(TOOLS_DIR)

$(CONTROLLER_GEN): $(TOOLS_DIR)
	@{ \
		set -e ;\
		GOBIN=$(TOOLS_DIR) go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.7.0 ;\
	}

$(GOLANGCI_LINT): $(TOOLS_DIR)
	@{ \
		set -e ;\
		GOBIN=$(TOOLS_DIR) go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.42.1 ;\
	}

$(KUSTOMIZE): $(TOOLS_DIR)
	@{ \
		set -e ;\
		GOBIN=$(TOOLS_DIR) go install sigs.k8s.io/kustomize/kustomize/v3 ;\
	}

$(OPERATOR_SDK): $(TOOLS_DIR)
	@{ \
		set -ex ;\
		OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
		curl -sSLo $(OPERATOR_SDK) https://github.com/operator-framework/operator-sdk/releases/download/v1.13.0/operator-sdk_$${OS}_$${ARCH} ;\
		chmod +x $(OPERATOR_SDK) ;\
	}

.PHONY: opm
OPM = $(TOOLS_DIR)/opm
$(OPM): $(TOOLS_DIR)
	@{ \
		set -e ;\
		OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
		curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/v1.15.1/$${OS}-$${ARCH}-opm ;\
		chmod +x $(OPM) ;\
	}

# Install all required tools
tools: $(CONTROLLER_GEN) $(GOLANGCI_LINT) $(KUSTOMIZE) \
			 $(OPERATOR_SDK) $(OPM)

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

.PHONY: generate-deepcopy
generate-deepcopy: $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./pkg/apis/..."

.PHONY: generate
generate: generate-crds generate-deepcopy

operator: generate
	go build -o ./tmp/operator ./cmd/operator/...


.PHONY: operator-image
operator-image: generate
	docker build -f build/Dockerfile . -t $(OPERATOR_IMAGE)

.PHONY: operator-push
operator-push:
	docker push ${OPERATOR_IMAGE}

## OLM - Bundle

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:$(VERSION)

# CHANNELS define the bundle channels used in the bundle.
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=candidate,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="candidate,fast,stable")
CHANNELS ?= candidate
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle.
# To re-generate a bundle for any other default channel without changing the default setup, use:
# - DEFAULT_CHANNEL as arg of the bundle target (e.g make bundle DEFAULT_CHANNEL=stable)
# - environment variables to overwrite this value (e.g export DEFAULT_CHANNEL="stable")
DEFAULT_CHANNEL ?= candidate

ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)


.PHONY: bundle
bundle: generate $(KUSTOMIZE) $(OPERATOR_SDK)
	cd deploy/operator && \
		$(KUSTOMIZE) edit set image monitoring-stack-operator=$(OPERATOR_IMAGE) && \
		$(KUSTOMIZE) edit set label app.kubernetes.io/version:$(VERSION)
	$(KUSTOMIZE) build deploy/olm | tee tmp/pre-bundle.yaml |  \
	 	$(OPERATOR_SDK) generate bundle \
			--overwrite \
		 	--version $(VERSION) \
			--kustomize-dir=deploy/olm \
			--package=monitoring-stack-operator \
		 	$(BUNDLE_METADATA_OPTS)
	$(OPERATOR_SDK) bundle validate ./bundle

.PHONY: bundle-build
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
CATALOG_IMG ?= $(IMAGE_TAG_BASE)-catalog:latest

# Set CATALOG_BASE_IMG to an existing catalog image tag to add $BUNDLE_IMGS to that image.
ifneq ($(origin CATALOG_BASE_IMG), undefined)
FROM_INDEX_OPT := --from-index $(CATALOG_BASE_IMG)
endif

# Build a catalog image by adding bundle images to an empty catalog using the
# operator package manager tool, 'opm'.
#
# NOTE: This recipe invokes 'opm' in 'semver' bundle add mode. For more information
# on add modes, see:
# https://github.com/operator-framework/community-operators/blob/7f1438c/docs/packaging-operator.md#updating-your-existing-operator
.PHONY: catalog-image
catalog-image: $(OPM) ## Build a catalog image.
	$(OPM) index add \
	 	--container-tool docker \
		--mode semver \
		--tag $(CATALOG_IMG) \
		--bundles $(BUNDLE_IMGS) $(FROM_INDEX_OPT)

# Push the catalog image.
.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	docker push $(CATALOG_IMG)
