# VPS install guide

This guide is the recommended production layout for a Linux VPS.

## Requirements

Install the native database tools for the driver you use.

PostgreSQL:

```bash
sudo apt update
sudo apt install -y postgresql-client
pg_dump --version
psql --version
pg_restore --version
```

MongoDB:

```bash
mongosh --version
mongodump --version
mongorestore --version
```

## Install

### Script install

```bash
curl -fsSL https://raw.githubusercontent.com/valpiks/backupctl/main/install.sh | sh
backupctl version
```

Install a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/valpiks/backupctl/main/install.sh | VERSION=v1.0.0 sh
```

Install into a custom directory:

```bash
curl -fsSL https://raw.githubusercontent.com/valpiks/backupctl/main/install.sh | INSTALL_DIR="$HOME/.local/bin" sh
```

### Debian / Ubuntu package

Download the `.deb` from GitHub Releases and install it:

```bash
wget https://github.com/valpiks/backupctl/releases/download/v1.0.0/backupctl_1.0.0_linux_amd64.deb
sudo apt install ./backupctl_1.0.0_linux_amd64.deb
backupctl version
```

### RPM package

For RHEL, Rocky, AlmaLinux, or Fedora:

```bash
curl -LO https://github.com/valpiks/backupctl/releases/download/v1.0.0/backupctl_1.0.0_linux_amd64.rpm
sudo rpm -Uvh ./backupctl_1.0.0_linux_amd64.rpm
backupctl version
```

### Alpine package

```bash
curl -LO https://github.com/valpiks/backupctl/releases/download/v1.0.0/backupctl_1.0.0_linux_amd64.apk
sudo apk add --allow-untrusted ./backupctl_1.0.0_linux_amd64.apk
backupctl version
```

Package installs the binary to `/usr/bin/backupctl`, creates runtime directories, and installs default config files:

```text
/etc/backupctl/config.yaml
/etc/backupctl/backupctl.env
/var/lib/backupctl/backups
/var/log/backupctl
```

Example configs are installed under:

```text
/usr/share/doc/backupctl/
```

## Server layout

Use this layout for manual installs too:

```text
/etc/backupctl/config.yaml
/etc/backupctl/backupctl.env
/var/lib/backupctl/backups
/var/log/backupctl/scheduler.log
```

For package installs, bootstrap is already done. For manual installs, create the layout:

```bash
sudo mkdir -p /etc/backupctl /var/lib/backupctl/backups /var/log/backupctl
sudo cp /usr/share/doc/backupctl/config.vps.example.yaml /etc/backupctl/config.yaml
sudo cp /usr/share/doc/backupctl/backupctl.env.example /etc/backupctl/backupctl.env
```

For script installs from the repository checkout:

```bash
sudo mkdir -p /etc/backupctl /var/lib/backupctl/backups /var/log/backupctl
sudo cp configs/config.vps.example.yaml /etc/backupctl/config.yaml
sudo cp configs/backupctl.env.example /etc/backupctl/backupctl.env
```

Edit:

```bash
sudo nano /etc/backupctl/config.yaml
sudo nano /etc/backupctl/backupctl.env
```

Minimum env file for PostgreSQL password-based configs:

```env
BACKUPCTL_POSTGRES_PASSWORD=change-me
```

Recommended config changes for local VPS storage:

The packaged default config already uses this layout:

```yaml
runtime:
  env_file: /etc/backupctl/backupctl.env

storage:
  type: local
  local:
    path: /var/lib/backupctl/backups

backup:
  scheduler:
    enabled: true
    interval: 24h
    log_file: /var/log/backupctl/scheduler.log
```

## Verify manual backup

```bash
backupctl config validate -c /etc/backupctl/config.yaml
backupctl doctor -c /etc/backupctl/config.yaml --deep
backupctl backup -c /etc/backupctl/config.yaml --dry-run
backupctl backup -c /etc/backupctl/config.yaml --verify
backupctl list -c /etc/backupctl/config.yaml
```

## Enable scheduler service

```bash
backupctl schedule -c /etc/backupctl/config.yaml --interval 24h
backupctl jobs
sudo backupctl service install --system --config /etc/backupctl/config.yaml --binary /usr/bin/backupctl
sudo backupctl service status --system
sudo backupctl service logs --system
```

For a user-level service:

```bash
backupctl service install --user --config /etc/backupctl/config.yaml --binary "$(command -v backupctl)"
backupctl service status --user
backupctl service logs --user
```

Native systemd checks:

```bash
systemctl status backupctl-scheduler
journalctl -u backupctl-scheduler -f
```

## Smoke checklist

Run this once on a clean VPS or VM before tagging a release:

```bash
backupctl version
backupctl config validate -c /etc/backupctl/config.yaml
backupctl doctor -c /etc/backupctl/config.yaml --deep
backupctl backup -c /etc/backupctl/config.yaml --verify
backupctl list -c /etc/backupctl/config.yaml
backupctl list -c /etc/backupctl/config.yaml --verified
backupctl restore -c /etc/backupctl/config.yaml --file <backup-file> --dry-run
backupctl cleanup -c /etc/backupctl/config.yaml --keep-last 5 --dry-run
backupctl jobs run <job-id>
sudo backupctl service restart --system
sudo backupctl service logs --system --tail 100
```
