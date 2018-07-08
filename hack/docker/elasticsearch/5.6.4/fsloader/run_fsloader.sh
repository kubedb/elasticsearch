#!/bin/bash

set -x
set -o errexit
set -o pipefail

searchguard="/elasticsearch/plugins/search-guard-5"

sync

case "$MODE" in
  client)
    # Run sgadmin in client node (with ordinal 0 only)
    ordinal="${NODE_NAME##*-}"
    if [ "$ordinal" == "0" ]; then
      /fsloader/run_sgadmin.sh
      /fsloader/fsloader run --mount-file "$searchguard"/sgconfig/sg_internal_users.yml \
        --boot-cmd /fsloader/run_sgadmin.sh
    fi
    ;;

  *) ;;
esac

echo "Ignore running sgadmin..."
tail -f /fsloader/run_sgadmin.sh
