SHELL=/usr/bin/env bash -o pipefail

IMAGE = monitoring-stack-operator
VERSION?=$(shell cat VERSION)

TOOLS_BIN_DIR=$(shell pwd)/tmp/bin
export PATH := $(TOOLS_BIN_DIR):$(PATH)
export GOBIN := $(TOOLS_BIN_DIR)

TOOLS=\
	$(TOOLS_BIN_DIR)/controller-gen \
	$(TOOLS_BIN_DIR)/golangci-lint

$(TOOLS_BIN_DIR):
	mkdir -p $(TOOLS_BIN_DIR)

$(TOOLS): $(TOOLS_BIN_DIR)
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.7.0
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.42.1

.PHONY: lint
lint: $(TOOLS)
	golangci-lint run ./... --fix

.PHONY: generate-crds
generate-crds: $(TOOLS)
	controller-gen \
		crd paths=./pkg/apis/... \
		rbac:roleName=monitoring \
		output:dir=./deploy \
		output:crd:dir=./deploy/crds

.PHONY: generate-deepcopy
generate-deepcopy: $(TOOLS)
	controller-gen object paths=./pkg/apis/...

.PHONY: generate
generate: generate-crds generate-deepcopy

.PHONY: image
image:
	docker build -f build/Dockerfile . -t $(IMAGE):$(VERSION)
