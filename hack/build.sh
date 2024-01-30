#! /usr/bin/env bash
#
# To push to your own registry, override the REGISTRY and NAMESPACE env vars,
# i.e:
#   $ REGISTRY=quay.io NAMESPACE=yourusername ./hack/build.sh
#
# REQUIREMENTS:
#  * a valid login session to a container registry.
#  * `docker`
#  * `jq`
#  * `yq`
#  * `opm`
#  * `skopeo`

set -eu -o pipefail

declare -r OPERATOR_NAME='observability-operator'
declare -r REGISTRY=${REGISTRY:-'quay.io'}
declare -r NAMESPACE=${NAMESPACE:-'rhobs'}
declare -r TAG=${TAG=$1}
declare -r CONTAINER_RUNTIME=$(shell command -v podman 2> /dev/null || echo docker)
declare -r CSV_PATH=${CSV_PATH:-'bundle/manifests/observability-operator.clusterserviceversion.yaml'}
declare -r ANNOTATIONS_PATH=${ANNOTATIONS_PATH:-'bundle/metadata/annotations.yaml'}

cleanup() {
	# shellcheck disable=SC2046
	if [ -x $(command -v git >/dev/null 2>&1) ]; then
		git checkout "${CSV_PATH}" >/dev/null 2>&1
		git checkout "${ANNOTATIONS_PATH}" >/dev/null 2>&1
	fi
}

trap cleanup EXIT

# prints pre-formatted info output.
info() {
	echo "INFO $(date '+%Y-%m-%dT%H:%M:%S') $*"
}

# prints pre-formatted error output.
error() {
	>&2 echo "ERROR $(date '+%Y-%m-%dT%H:%M:%S') $*"
}

digest() {
	local -n ret=$2
	IMAGE=$1
	${CONTAINER_RUNTIME} pull "${IMAGE}"
	# shellcheck disable=SC2034
	ret=$("${CONTAINER_RUNTIME}" inspect --format='{{index .RepoDigests 0}}' "${IMAGE}")
}

build_push_operator_image() {
	make operator-image OPERATOR_IMG=${REGISTRY}/${NAMESPACE}/${OPERATOR_NAME}:${TAG}
	make operator-push OPERATOR_IMG=${REGISTRY}/${NAMESPACE}/${OPERATOR_NAME}:${TAG}
	digest "${REGISTRY}/${NAMESPACE}/${OPERATOR_NAME}:${TAG}" OPERATOR_DIGEST
	# need exporting so that yq can see them
	declare -r OPERATOR_DIGEST
}

prepare_operator_files() {
	# prepare operator files, then build and push operator bundle and catalog
	# index images.

	/usr/local/bin/yq eval -i '
		.metadata.name = strenv(OPERATOR_NAME) |
		.metadata.annotations.version = strenv(TAG) |
		.metadata.annotations.containerImage = strenv(OPERATOR_DIGEST) |
		.metadata.labels += {"operatorframework.io/arch.amd64": "supported", "operatorframework.io/arch.ppc64le": "supported", "operatorframework.io/os.linux": "supported"} |
		del(.spec.replaces) |
		.spec.install.spec.deployments[0].name = strenv(OPERATOR_NAME) |
		.spec.install.spec.deployments[2].spec.template.spec.containers[0].image = strenv(OPERATOR_DIGEST)
		' "${CSV_PATH}"

	/usr/local/bin/yq eval -i '
		.annotations."operators.operatorframework.io.bundle.channel.default.v1" = "test" |
		.annotations."operators.operatorframework.io.bundle.channels.v1" = "test"
		' "${ANNOTATIONS_PATH}"	
}	

build_bundle_image() {
	make bundle-image BUNDLE_IMG=${REGISTRY}/${NAMESPACE}/${OPERATOR_NAME}-bundle:${TAG}
	make bundle-push BUNDLE_IMG=${REGISTRY}/${NAMESPACE}/${OPERATOR_NAME}-bundle:${TAG}
}

build_single_arch_index_image() {
	AMD64_DIGEST=$(skopeo inspect --raw  docker://${REGISTRY}/${NAMESPACE}/${OPERATOR_NAME}-bundle:${TAG} | \
               jq -r '.manifests[] | select(.platform.architecture == "amd64" and .platform.os == "linux").digest')
	POWER_DIGEST=$(skopeo inspect --raw  docker://${REGISTRY}/${NAMESPACE}/${OPERATOR_NAME}-bundle:${TAG} | \
               jq -r '.manifests[] | select(.platform.architecture == "ppc64le" and .platform.os == "linux").digest')
	/usr/local/bin/opm index add --build-tool ${CONTAINER_RUNTIME} --bundles "${REGISTRY}/${NAMESPACE}/${OPERATOR_NAME}-bundle@${AMD64_DIGEST}" --tag "${REGISTRY}/${NAMESPACE}/${OPERATOR_NAME}-catalog:${TAG}-amd64" --binary-image "quay.io/operator-framework/opm:v1.28.0-amd64"
	/usr/local/bin/opm index add --build-tool ${CONTAINER_RUNTIME} --bundles "${REGISTRY}/${NAMESPACE}/${OPERATOR_NAME}-bundle@${POWER_DIGEST}" --tag "${REGISTRY}/${NAMESPACE}/${OPERATOR_NAME}-catalog:${TAG}-ppc64le" --binary-image "quay.io/operator-framework/opm:v1.28.0-ppc64le"
}

push_single_arch_index_images() {
	${CONTAINER_RUNTIME} push "${REGISTRY}/${NAMESPACE}/observability-operator-catalog:${TAG}-amd64"
	${CONTAINER_RUNTIME} "${REGISTRY}/${NAMESPACE}/observability-operator-catalog:${TAG}-ppc64le"
}

build_catalog_manifest() {
	${CONTAINER_RUNTIME} manifest create "observability-operator-catalog:${TAG}" \
		"${REGISTRY}/${NAMESPACE}/observability-operator-catalog:${TAG}-amd64" \
		"${REGISTRY}/${NAMESPACE}/observability-operator-catalog:${TAG}-ppc64le"
}

push_catalog_manifest() {
	${CONTAINER_RUNTIME} manifest push "observability-operator-catalog:${TAG}" \
	       	"${REGISTRY}/${NAMESPACE}/observability-operator-catalog:${TAG}"
}

main() {
	build_push_operator_image
	prepare_operator_files
	build_bundle_image
	build_single_arch_index_image
	push_single_arch_index_images
	build_catalog_manifest
	push_catalog_manifest
	return $?
}

main "$@"
