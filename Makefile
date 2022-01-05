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

## NOTE: each tool must have a version that will be recorded in .github/tools
# The .github/tools file's hash is used to compute the key for cache in github
# see: .github/tools-cache/action.yaml

CONTROLLER_GEN = $(TOOLS_DIR)/controller-gen
CONTROLLER_GEN_VERSION = v0.7.0

KUSTOMIZE = $(TOOLS_DIR)/kustomize
KUSTOMIZE_VERSION = v3.9.4

OPERATOR_SDK = $(TOOLS_DIR)/operator-sdk
OPERATOR_SDK_VERSION = v1.13.0

OPM = $(TOOLS_DIR)/opm
OPM_VERSION = v1.15.1

GOLANGCI_LINT = $(TOOLS_DIR)/golangci-lint
GOLANGCI_LINT_VERSION = v1.42.1

## NOTE: promq does not have any releases, so we use a fake version starting with v0.0.1
# thus to upgrade/invalidate the github cache, increment the value
PROMQ = $(TOOLS_DIR)/promq
PROMQ_VERSION = v0.0.1

# NOTE: oc is NOT downloadable using the OC_VERSION in its URL, so this has to be manually updated
OC = $(TOOLS_DIR)/oc
OC_VERSION = v4.8.11

# jsonnet related tools and dependencies
JSONNET = $(TOOLS_DIR)/jsonnet
JSONNET_VERSION = v0.17.0

JSONNETFMT = $(TOOLS_DIR)/jsonnetfmt
JSONNETFMT_VERSION = v0.17.0

JSONNET_LINT = $(TOOLS_DIR)/jsonnet-lint
JSONNET_LINT_VERSION = v0.17.0

JB = $(TOOLS_DIR)/jb
JB_VERSION = v0.4.0

## NOTE: gojsontoyaml does not have any releases, so we use a fake version starting with v0.0.1
# thus to upgrade/invalidate the github cache, increment the value
GOJSONTOYAML = $(TOOLS_DIR)/gojsontoyaml
GOJSONTOYAML_VERSRION = 0.0.1

JSONNET_VENDOR = jsonnet/vendor
JSONNETFMT_ARGS = -n 2 --max-blank-lines 2 --string-style s --comment-style s

$(TOOLS_DIR):
	@mkdir -p $(TOOLS_DIR)

.PHONY: controller-gen
$(CONTROLLER_GEN) controller-gen: $(TOOLS_DIR)
	@{ \
		set -ex ;\
		GOBIN=$(TOOLS_DIR) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_GEN_VERSION) ;\
	}

.PHONY: golangci-lint
$(GOLANGCI_LINT) golangci-lint: $(TOOLS_DIR)
	@{ \
		set -ex ;\
		GOBIN=$(TOOLS_DIR) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) ;\
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
		GOBIN=$(TOOLS_DIR) go get sigs.k8s.io/kustomize/kustomize/v3@$(KUSTOMIZE_VERSION) ;\
		rm -rf $$TMP_DIR ;\
	}

.PHONY: operator-sdk
$(OPERATOR_SDK) operator-sdk: $(TOOLS_DIR)
	@{ \
		set -ex ;\
		[[ -f $(OPERATOR_SDK) ]] && exit 0 ;\
		OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
		curl -sSLo $(OPERATOR_SDK) https://github.com/operator-framework/operator-sdk/releases/download/$(OPERATOR_SDK_VERSION)/operator-sdk_$${OS}_$${ARCH} ;\
		chmod +x $(OPERATOR_SDK) ;\
	}

.PHONY: opm
$(OPM) opm: $(TOOLS_DIR)
	@{ \
		set -ex ;\
		[[ -f $(OPM) ]] && exit 0 ;\
		OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
		curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/$(OPM_VERSION)/$${OS}-$${ARCH}-opm ;\
		chmod +x $(OPM) ;\
	}

.PHONY: promq
$(PROMQ) promq: $(TOOLS_DIR)
	@{ \
		set -ex ;\
		[[ -f $(PROMQ) ]] && exit 0 ;\
		TMP_DIR=$$(mktemp -d) ;\
		cd $$TMP_DIR ;\
		echo "Downloading promq" ;\
		git clone --depth=1 https://github.com/kubernetes-sigs/instrumentation-tools ;\
		cd instrumentation-tools ;\
		go build -o $(PROMQ) . ;\
		rm -rf $$TMP_DIR ;\
	}

.PHONY: oc
$(OC) oc: $(TOOLS_DIR)
	@{ \
		set -ex ;\
		[[ -f $(OC) ]] && exit 0 ;\
		OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
		curl -sSLo $(OC) https://mirror.openshift.com/pub/openshift-v4/$${ARCH}/clients/oc/latest/$${OS}/oc.tar.gz ;\
		tar -xf $(TOOLS_DIR)/oc -C $(TOOLS_DIR) ;\
		rm -f $(TOOLS_DIR)/README.md ;\
		$(OC) version ;\
		version=$(OC_VERSION) ;\
		$(OC) version | grep -q $${version##v} ;\
	}

.PHONY: jsonnet
$(JSONNET) jsonnet: $(TOOLS_DIR)
		GOBIN=$(TOOLS_DIR) go install github.com/google/go-jsonnet/cmd/jsonnet@latest

.PHONY: jsonnetfmt
$(JSONNETFMT) jsonnetfmt: $(TOOLS_DIR)
		GOBIN=$(TOOLS_DIR)  go install github.com/google/go-jsonnet/cmd/jsonnetfmt@latest

.PHONY: jsonnet-lint
$(JSONNET_LINT) jsonnet-lint: $(TOOLS_DIR)
	GOBIN=$(TOOLS_DIR)  go install github.com/google/go-jsonnet/cmd/jsonnet-lint@latest

.PHONY: jb
$(JB) jb: $(TOOLS_DIR)
	GOBIN=$(TOOLS_DIR) go install github.com/jsonnet-bundler/jsonnet-bundler/cmd/jb@latest

.PHONY: gojsontoyaml
$(GOJSONTOYAML) gojsontoyaml: $(TOOLS_DIR)
	GOBIN=$(TOOLS_DIR) go install github.com/brancz/gojsontoyaml@latest

.PHONY: jsonnet-tools
jsonnet-tools: jsonnet jsonnetfmt jsonnet-lint jb gojsontoyaml

# Install all required tools
.PHONY: tools
tools: $(CONTROLLER_GEN) \
 		$(KUSTOMIZE) \
		$(OC) \
		$(OPERATOR_SDK) \
		$(OPM) \
		$(PROMQ) \
		jsonnet-tools
	@{ \
		set -ex ;\
		tools_file=.github/tools ;\
		echo '# DO NOT EDIT! Autogenerated by make tools' >$$tools_file ;\
		echo >> $$tools_file ;\
		echo  $$(basename $(CONTROLLER_GEN)) $(CONTROLLER_GEN_VERSION) >> $$tools_file ;\
		echo  $$(basename $(KUSTOMIZE)) $(KUSTOMIZE_VERSION) >> $$tools_file ;\
		echo  $$(basename $(OC)) $(OC_VERSION) >> $$tools_file ;\
		echo  $$(basename $(OPERATOR_SDK)) $(OPERATOR_SDK_VERSION) >> $$tools_file ;\
		echo  $$(basename $(OPM)) $(OPM_VERSION) >> $$tools_file ;\
		echo  $$(basename $(PROMQ)) $(PROMQ_VERSION) >> $$tools_file ;\
		echo  $$(basename $(JSONNET)) $(JSONNET_VERSION) >> $$tools_file ;\
		echo  $$(basename $(JSONNETFMT)) $(JSONNETFMT_VERSION) >> $$tools_file ;\
		echo  $$(basename $(JSONNET_LINT)) $(JSONNET_LINT_VERSION) >> $$tools_file ;\
		echo  $$(basename $(JB)) $(JB_VERSION) >> $$tools_file ;\
		echo  $$(basename $(GOJSONTOYAML)) $(GOJSONTOYAML_VERSRION) >> $$tools_file ;\
	}

## Development

.PHONY: lint
lint: lint-golang lint-jsonnet

.PHONY: lint-golang
lint-golang: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run ./... --fix

.PHONY: lint-jsonnet
lint-jsonnet: $(JSONNET_LINT) jsonnet-vendor
	find jsonnet/ -name 'vendor' -prune \
		-o -name '*.libsonnet' -print \
		-o -name '*.jsonnet' -print \
	| xargs -n 1 -- $(JSONNET_LINT) -J $(JSONNET_VENDOR)

.PHONY: fmt-jsonnet
fmt-jsonnet: $(JSONNETFMT) jsonnet-vendor
	find jsonnet/ -name 'vendor' -prune \
		-o -name '*.libsonnet' -print \
		-o -name '*.jsonnet' -print \
	| xargs -n 1 -- $(JSONNETFMT) $(JSONNETFMT_ARGS) -i


.PHONY: check-jq
check-jq:
	jq --version > /dev/null

.PHONY: jsonnet-vendor
jsonnet-vendor: $(JB)
	cd jsonnet && $(JB) install

.PHONY: generate-prometheus-rules
generate-prometheus-rules: jsonnet-tools check-jq kustomize jsonnet-vendor
	for dir in jsonnet/components/*/; do \
		component=$$(basename $$dir) ;\
		echo "Generating prometheusrule file for $$component" ;\
		$(JSONNET) -J $(JSONNET_VENDOR) $$dir/main.jsonnet \
			| jq .rule \
			| $(GOJSONTOYAML) > deploy/operator/monitoring-stack-operator-$$component-rules.yaml ;\
		cd deploy/operator && \
		$(KUSTOMIZE) edit add resource "monitoring-stack-operator-$$component-rules.yaml" && cd - ;\
	done;

.PHONY: generate-crds
generate-crds: $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) crd \
		paths=./pkg/apis/... \
		paths=./pkg/controllers/... \
		rbac:roleName=monitoring-stack-operator \
		output:dir=. \
		output:rbac:dir=./deploy/operator \
		output:crd:dir=./deploy/crds/common
	mv deploy/operator/role.yaml deploy/operator/monitoring-stack-operator-cluster-role.yaml

.PHONY: generate-kustomize
generate-kustomize: $(KUSTOMIZE)
	cd deploy/operator && \
		$(KUSTOMIZE) edit set image monitoring-stack-operator=*:$(VERSION)

.PHONY: generate-deepcopy
generate-deepcopy: $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./pkg/apis/..."

.PHONY: generate
generate: generate-crds generate-deepcopy generate-kustomize generate-prometheus-rules

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
	$(STANDARD_VERSION) -a --skip.tag # The tag will be created in the pipeline

.PHONY: initiate-release-as
initiate-release-as: $(STANDARD_VERSION)
	git fetch git@github.com:rhobs/monitoring-stack-operator.git --tags
	$(STANDARD_VERSION) -a --skip.tag --release-as $(RELEASE_VERSION)

.PHONY: kind-cluster
kind-cluster: $(OPERATOR_SDK)
	kind create cluster --config hack/kind/config.yaml
	$(OPERATOR_SDK) olm install
	kubectl apply -f hack/kind/registry.yaml -n operators
	kubectl create -k deploy/crds/kubernetes/
	kubectl create -k deploy/dependencies

.PHONY: clean
clean:
	rm -rf $(JSONNET_VENDOR) bundle/ bundle.Dockerfile
