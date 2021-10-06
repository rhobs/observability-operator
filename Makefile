SHELL=/usr/bin/env bash -o pipefail

IMAGE = monitoring-stack-operator
VERSION ?= $(shell cat VERSION)

# running make builds the operator (default target)
all: operator


## Tools

TOOLS_DIR = $(shell pwd)/tmp/bin
CONTROLLER_GEN=$(TOOLS_DIR)/controller-gen
GOLANGCI_LINT=$(TOOLS_DIR)/golangci-lint

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



# Install all required tools
tools: $(CONTROLLER_GEN) $(GOLANGCI_LINT) ##

## Development

.PHONY: lint
lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run ./... --fix

.PHONY: generate-crds
generate-crds: $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) crd \
		paths=./pkg/apis/... \
		rbac:roleName=monitoring \
		output:dir=./deploy \
		output:crd:dir=./deploy/crds

.PHONY: generate-deepcopy
generate-deepcopy: $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) object paths=./pkg/apis/...

.PHONY: generate
generate: generate-crds generate-deepcopy

operator: generate
	go build -o ./tmp/operator ./cmd/operator/...

.PHONY: image
image:
	docker build -f build/Dockerfile . -t $(IMAGE):$(VERSION)
