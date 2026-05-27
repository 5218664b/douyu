#!/bin/sh
set -eu

mkdir -p /srv/samba

if ! id x >/dev/null 2>&1; then
  useradd -M -s /usr/sbin/nologin x
fi

(echo x; echo x) | smbpasswd -s -a x >/dev/null

exec smbd --foreground --no-process-group
