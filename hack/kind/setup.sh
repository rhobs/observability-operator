#!/usr/bin/env bash
set -e -u -o pipefail

# This script is now a wrapper around the unified setup script
# It maintains backward compatibility while using the new unified approach

SCRIPT_PATH=$(readlink -f "$0")
declare -r SCRIPT_PATH
SCRIPT_DIR=$(cd "$(dirname "$SCRIPT_PATH")" && pwd) 
declare -r SCRIPT_DIR
declare -r PROJECT_ROOT_DIR="$SCRIPT_DIR/../../"

# Print deprecation notice
echo "⚠️  DEPRECATION NOTICE: hack/kind/setup.sh is deprecated"
echo "    Please use the new unified setup script: hack/setup-e2e-env.sh"
echo "    This script will forward to the new one for backward compatibility"
echo ""

# Forward to the new unified script with appropriate options
exec "$PROJECT_ROOT_DIR/hack/setup-e2e-env.sh" "$@"
