SHELL=/usr/bin/env bash -o pipefail

include Makefile.tools

# IMAGE_BASE defines the registry/namespace and part of the image name
# This variable is used to construct full image tags for bundle and catalog images.
IMAGE_BASE ?= observability-operator

VERSION ?= $(shell cat VERSION)
OPERATOR_IMG = $(IMAGE_BASE):$(VERSION)
OPERATOR_BUNDLE=observability-operator.v$(VERSION)
CONTAINER_RUNTIME := $(shell command -v podman 2> /dev/null || echo docker)

# running `make` builds the operator (default target)
all: operator

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
			| $(GOJSONTOYAML) > deploy/monitoring/monitoring-$$component-rules.yaml ;\
		cd deploy/monitoring && \
		$(KUSTOMIZE) edit add resource "monitoring-$$component-rules.yaml" && cd - ;\
	done;

.PHONY: docs
docs: $(CRDOC)
	mkdir -p docs
	$(CRDOC) --resources deploy/crds/common --output docs/api.md

# This generates the prometheus-operator CRD manifests from the
# prometheus-operator dependency defined in go.mod. This ensures we carry the
# correct version of the CRD manifests.
.PHONY: generate-prom-op-crds
generate-prom-operator-crds: $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) crd \
		paths=github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/... \
		output:dir=. \
		output:crd:dir=./deploy/crds/kubernetes

.PHONY: generate-crds
generate-crds: $(CONTROLLER_GEN) generate-prom-op-crds
	$(CONTROLLER_GEN) crd \
		paths=./pkg/apis/... \
		paths=./pkg/controllers/... \
		rbac:roleName=observability-operator \
		output:dir=. \
		output:rbac:dir=./deploy/operator \
		output:crd:dir=./deploy/crds/common
	mv deploy/operator/role.yaml deploy/operator/observability-operator-cluster-role.yaml

.PHONY: generate-kustomize
generate-kustomize: $(KUSTOMIZE)
	cd deploy/operator && \
		$(KUSTOMIZE) edit set image observability-operator=$(OPERATOR_IMG)

.PHONY: generate-package-resources
generate-package-resources: $(KUSTOMIZE) generate-kustomize
	cd deploy/package-operator && \
		rm -rf package/crds package/dependencies package/operator ;\
		mkdir -p package/crds ;\
		$(KUSTOMIZE) build crds > package/crds/resources.yaml ;\
		mkdir -p package/dependencies ;\
		$(KUSTOMIZE) build dependencies > package/dependencies/resources.yaml ;\
		mkdir -p package/operator ;\
		$(KUSTOMIZE) build operator > package/operator/resources.yaml

.PHONY: generate-package-resources-kubeconfig
generate-package-resources-kubeconfig: $(KUSTOMIZE) generate-kustomize generate-crds
	cd deploy/package-operator && \
		rm -rf package/crds package/dependencies package/operator ;\
		mkdir -p package/crds ;\
		$(KUSTOMIZE) build crds > package/crds/resources.yaml ;\
		mkdir -p package/dependencies ;\
		$(KUSTOMIZE) build dependencies-kubeconfig > package/dependencies/resources.yaml ;\
		mkdir -p package/operator ;\
		$(KUSTOMIZE) build operator-kubeconfig > package/operator/resources.yaml

.PHONY: generate-deepcopy
generate-deepcopy: $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./pkg/apis/..."

.PHONY: generate
generate: generate-crds generate-deepcopy generate-kustomize generate-package-resources generate-prometheus-rules docs

.PHONY: operator
operator: generate build

.PHONY: build
build:
	go build -o ./tmp/operator ./cmd/operator/...

.PHONY: operator-image
operator-image: generate
	$(CONTAINER_RUNTIME) build -f build/Dockerfile . -t $(OPERATOR_IMG)

.PHONY: operator-push
operator-push:
	$(CONTAINER_RUNTIME) push $(PUSH_OPTIONS) ${OPERATOR_IMG}

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
	$(KUSTOMIZE) build deploy/olm | tee tmp/pre-bundle.yaml |  \
	 	$(OPERATOR_SDK) generate bundle \
			--overwrite \
		 	--version $(VERSION) \
			--kustomize-dir=deploy/olm \
			--package=observability-operator \
		 	$(BUNDLE_METADATA_OPTS)
	$(OPERATOR_SDK) bundle validate ./bundle

.PHONY: bundle-image
bundle-image: bundle ## Build the bundle image.
	$(CONTAINER_RUNTIME) build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Build the bundle image.
	$(CONTAINER_RUNTIME) push $(PUSH_OPTIONS) $(BUNDLE_IMG)

# The image tag given to the resulting catalog image
CATALOG_IMG_BASE ?= $(IMAGE_BASE)-catalog
CATALOG_IMG ?= $(CATALOG_IMG_BASE):$(VERSION)

# The tag is used as latest since it allows a CatalogSubscription to point to
# a single image which keeps updating there by allowing auto upgrades
CATALOG_IMG_LATEST ?= $(IMAGE_BASE)-catalog:latest


# NOTE: This is required to enable continuous deployment to
# staging/integration environments via app-interface (OSD-13603)
#
# The git short-hash of the most recent commit in the repository, which will
# be used for image tag association against the built catalog image
CATALOG_IMG_SHA = $(CATALOG_IMG_BASE):$(shell git rev-parse --short=8 HEAD)

# Build a catalog image by adding bundle images to an empty catalog using the
# operator package manager tool, 'opm'.
.PHONY: catalog-image
catalog-image: $(OPM)
	$(OPM) render $(BUNDLE_IMG) \
		--output=yaml  >> olm/observability-operator-index/index.yaml
	./olm/update-channels.sh $(CHANNELS) $(OPERATOR_BUNDLE)
	$(OPM) validate ./olm/observability-operator-index

	$(CONTAINER_RUNTIME) build \
		-f olm/observability-operator-index.Dockerfile \
		-t $(CATALOG_IMG)

	# tag the catalog img:version as latest so that continious release
	# is possible by refering to latest tag instead of a version
	$(CONTAINER_RUNTIME) tag $(CATALOG_IMG) $(CATALOG_IMG_LATEST)

# NOTE: This target is created to ensure that the catalog image points to the
# SHA/commit in olm-catalog branch (instead of the commmit in main) which adds
# the bundle and the olm catalog instead of the SHA of the checkout (code change)
.PHONY: catalog-tag-sha
catalog-tag-sha: ## Push a catalog image.
	$(CONTAINER_RUNTIME) tag $(CATALOG_IMG) $(CATALOG_IMG_SHA)

# Push the catalog image.
.PHONY: catalog-push
catalog-push: catalog-tag-sha ## Push a catalog image.
	$(CONTAINER_RUNTIME) push $(PUSH_OPTIONS) $(CATALOG_IMG)
	$(CONTAINER_RUNTIME) push $(PUSH_OPTIONS) $(CATALOG_IMG_LATEST)
	$(CONTAINER_RUNTIME) push $(PUSH_OPTIONS) $(CATALOG_IMG_SHA)

## package-operator package

# The image tag given to the resulting package image
PACKAGE_IMG_BASE ?= $(IMAGE_BASE)-package
PACKAGE_IMG ?= $(PACKAGE_IMG_BASE):$(VERSION)

.PHONY: package
package: generate
	cd deploy/package-operator && \
		$(CONTAINER_RUNTIME) build \
			-f package.Containerfile \
			-t $(PACKAGE_IMG) package/

.PHONY: package-kubeconfig
package-kubeconfig: generate-package-resources-kubeconfig
	cd deploy/package-operator && \
		$(CONTAINER_RUNTIME) build \
			-f package.Containerfile \
			-t $(PACKAGE_IMG)-kubeconfig package/

.PHONY: package-push
package-push:
	$(CONTAINER_RUNTIME) push $(PUSH_OPTIONS) $(PACKAGE_IMG)

.PHONY: package-push-kubeconfig
package-push-kubeconfig:
	$(CONTAINER_RUNTIME) push $(PUSH_OPTIONS) $(PACKAGE_IMG)-kubeconfig

## Release process

.PHONY: release
release: operator-image operator-push bundle-image bundle-push catalog-image catalog-push

STANDARD_VERSION=$(TOOLS_DIR)/standard-version
$(STANDARD_VERSION):
	npm install -g --prefix tmp standard-version

.PHONY: initiate-release
initiate-release: $(STANDARD_VERSION)
	git fetch https://github.com/rhobs/observability-operator.git --tags
	$(STANDARD_VERSION) -a --skip.tag # The tag will be created in the pipeline

.PHONY: initiate-release-as
initiate-release-as: $(STANDARD_VERSION)
	git fetch https://github.com/rhobs/observability-operator.git --tags
	$(STANDARD_VERSION) -a --skip.tag --release-as $(RELEASE_VERSION)

.PHONY: kind-cluster
kind-cluster: $(OPERATOR_SDK)
	kind create cluster --config hack/kind/config.yaml
	$(OPERATOR_SDK) olm install
	kubectl apply -f hack/kind/registry.yaml -n operators
	kubectl create -k deploy/crds/kubernetes/
	kubectl create -k deploy/dependencies

.PHONY: clean
clean: clean-tools
	rm -rf $(JSONNET_VENDOR) bundle/ bundle.Dockerfile
