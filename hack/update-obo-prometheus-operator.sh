#!/usr/bin/env bash

set -euo pipefail

# Usage:
#   hack/update-obo-prometheus-operator.sh <old-version> <new-version>
#
# Example:
#   hack/update-obo-prometheus-operator.sh v0.87.0-rhobs1 v0.88.0-rhobs1
#
# This script replaces all occurrences of the given obo-prometheus-operator
# version string in source files, including:
#   - kustomization files (e.g. deploy/dependencies/kustomization.yaml)
#   - go.mod files
#   - any other YAML/YML files under the repo, excluding generated assets
#     under bundle/ and tmp/.
#
# After running this script you should typically run:
#   - go mod tidy
#   - make bundle

if [[ $# -ne 2 ]]; then
  echo "Usage: $0 <old-version> <new-version>" >&2
  exit 1
fi

OLD_VERSION="$1"
NEW_VERSION="$2"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

cd "${REPO_ROOT}"

echo "Updating obo-prometheus-operator from '${OLD_VERSION}' to '${NEW_VERSION}'"
echo "Repository root: ${REPO_ROOT}"

# Find candidate files:
# - All kustomization files
# - All go.mod files
# - All YAML/YML files
# Exclude:
# - Generated bundle content
# - Temporary / build output
mapfile -t FILES < <(
  grep -Rl --null "${OLD_VERSION}" . \
    --include='kustomization.yaml' \
    --include='go.mod' \
    --include='*.yaml' \
    --include='*.yml' \
    | tr '\0' '\n' \
    | grep -v -E '^./bundle/' \
    | grep -v -E '^./tmp/'
)

if [[ ${#FILES[@]} -eq 0 ]]; then
  echo "No files found containing '${OLD_VERSION}' (nothing to do)." >&2
  exit 1
fi

echo "Will update the following files:"
for f in "${FILES[@]}"; do
  echo "  - ${f}"
done

for f in "${FILES[@]}"; do
  sed -i "s/${OLD_VERSION}/${NEW_VERSION}/g" "${f}"
done

echo "Done."
echo "Next steps (recommended):"
echo "  - go mod tidy"
echo "  - make bundle"
