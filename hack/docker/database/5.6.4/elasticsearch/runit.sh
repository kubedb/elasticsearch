#!/bin/bash

set -x
set -o errexit
set -o pipefail

sync

echo "Starting runit..."
exec /sbin/runsvdir -P /etc/service
