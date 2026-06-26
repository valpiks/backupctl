#!/bin/sh
set -eu

cat <<'MSG'
backupctl installed.

Default config:
  /etc/backupctl/config.yaml

Default env file:
  /etc/backupctl/backupctl.env

Next:
  1. Edit /etc/backupctl/config.yaml
  2. Edit /etc/backupctl/backupctl.env
  3. Run backupctl doctor --deep
  4. Run backupctl backup --dry-run

See:
  /usr/share/doc/backupctl/
MSG
