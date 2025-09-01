#!/usr/bin/env bash
set -e -u -o pipefail


#usage $0 channel1[,channel2,...] bundle

to_upper() {
  echo "$@" | tr '[:lower:]' '[:upper:]'
}

err() {
  echo "ERROR: $*"
}

declare -r CATALOG_TEMPLATE="olm/index-template.yaml"

update_channel() {
  local channel="$1"; shift
  local bundle="$1"; shift

  echo "updating channel: $channel | bundle: $bundle"

  old=$(CHANNEL=$channel :; yq '.entries[] | select(.name == strenv(CHANNEL) and .schema == "olm.channel").entries[-1].name' "$CATALOG_TEMPLATE")

  # dev releases are suffixed with `-$(date +%y%m%d%H%M%S)`, those are replaced.
  # We track RC and actual releases fully.
  if [ "$channel" == "development" ]; then
      yq -i 'del(.entries[] | select(.image | test(".*-\d{12}$") and .schema == "olm.bundle"))' "$CATALOG_TEMPLATE"
      CHANNEL=$channel yq -i 'del(.entries[] | select(.name == strenv(CHANNEL) and .schema == "olm.channel").entries[] | select(.name | test(".*-\d{12}$")))' "$CATALOG_TEMPLATE"
  fi

  operator=${bundle//"-bundle"/}
  BUNDLE="$bundle" yq -i '.entries += {"image": strenv(BUNDLE),"schema": "olm.bundle"}' "$CATALOG_TEMPLATE"
  (OLD=$old OP=$operator CHANNEL=$channel yq -i '(.entries[] | select(.name == strenv(CHANNEL) and .schema == "olm.channel").entries) += [{"name": strenv(OP), "replaces": strenv(OLD)}]' "$CATALOG_TEMPLATE")
}

main() {
  cd "$(git rev-parse --show-toplevel)"
  local channels="$1"; shift
  local bundle="$1"; shift

  echo "channels: $channels | bundle: $bundle"

  # convert comma seperated list to an array
  local -a channel_list
  readarray -td, channel_list <<< "$channels,"; unset 'channel_list[-1]'

  for ch in "${channel_list[@]}"; do
    update_channel "$ch" "$bundle"
  done

  return $?
}

main "$@"
