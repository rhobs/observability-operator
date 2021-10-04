SHELL=/usr/bin/env bash -o pipefail

.PHONY: tools
tools:
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.7.0

.PHONY: generate-crds
generate-crds:
	controller-gen \
		crd paths=./pkg/apis/... \
		rbac:roleName=monitoring \
		output:dir=./deploy \
		output:crd:dir=./deploy/crds

.PHONY: generate-deepcopy
generate-deepcopy:
	controller-gen object paths=./pkg/apis/...

.PHONY: generate
generate: generate-crds generate-deepcopy
