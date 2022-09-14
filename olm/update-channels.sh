#!/usr/bin/env bash
set -e -u -o pipefail


#usage $0 channel1[,channel2,...] bundle

to_upper() {
  echo "$@" | tr '[:lower:]' '[:upper:]'
}

err() {
  echo "ERROR: $*"
}

declare -r CATALOG_INDEX_FILE="olm/observability-operator-index/index.yaml"

update_channel() {
  local channel=$1; shift
  local bundle=$1; shift

  echo "updating channel: $channel | bundle: $bundle"
  local marker
  marker="### $(to_upper "$channel")_CHANNEL_MARKER ###"

  if ! grep -q "$marker" "$CATALOG_INDEX_FILE"; then
    err "No marker '$marker' found in $CATALOG_INDEX_FILE"
    return 1
  fi

  # find the entry before that
  local previous_entry
  previous_entry=$(grep "$marker" -B2 "$CATALOG_INDEX_FILE" | grep  'name:' | cut -f2 -d:)
  echo " -> found previous: $previous_entry"

 sed -e \
   "s|^\($marker\)|  - name: $bundle\n    replaces: $previous_entry\n\1|" \
   -i "$CATALOG_INDEX_FILE"

}

main() {
  cd "$(git rev-parse --show-toplevel)"
  local channels=$1; shift
  local bundle=$1; shift

  echo "channels: $channels | bundle: $bundle"

  # convert coma seperated list to an array
  local -a channel_list
  readarray -td, channel_list <<< "$channels"

  for ch in "${channel_list[@]}"; do
    update_channel "$ch" "$bundle"
  done

  return $?
}

main "$@"
