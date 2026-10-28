#!/bin/sh
set -eu

media_dir="${MEDIA_DIR:-/app/data/media}"
mkdir -p "$media_dir"

# A fresh named volume is owned by root. Fix ownership once, then drop
# privileges for the long-running process.
if [ "$(id -u)" = "0" ]; then
    chown -R blog:blog /app/data
    exec su-exec blog /usr/local/bin/arnav.blog "$@"
fi

exec /usr/local/bin/arnav.blog "$@"
