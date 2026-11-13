#!/usr/bin/env bash
set -euo pipefail

namespace=$1

# Find the operator-sdk Subscription by listing subscriptions for this package
# and matching each name against its installed CSV with dots replaced by hyphens
# and "-sub" appended. The image's CSV may differ from the local bundle's CSV.
# Return the name only when exactly one Subscription matches.
kubectl get subscriptions -n "$namespace" -o json |
  jq -er '
    [
      .items[]
      | select(.spec.name == "observability-operator")
      | .status.installedCSV as $csv
      | select($csv != null)
      | ($csv | gsub("\\."; "-") + "-sub") as $sdk_name
      | select(.metadata.name == $sdk_name)
      | .metadata.name
    ]
    | if length == 1 then .[0] else error("expected one operator-sdk Subscription") end
  '
