# backupctl

CLI utility for database backups and restore with pluggable database and storage drivers.

## Version

```bash
backupctl version
```

Release builds inject the Git tag version automatically via GoReleaser.

## Features

* Full database backups for PostgreSQL and MongoDB
* Multiple backup modes and formats for PostgreSQL:
  - Full backup with all data
  - Schema-only backup (`--schema-only`)
  - Data-only backup (`--data-only`)
  - Select specific tables (`--tables`)
  - Plain format (`.sql.gz`) - text-based, human-readable
  - Custom format (`.dump`) - optimized for restore
* Streaming gzip compression (low memory usage)
* Local and S3-compatible storage
* Streaming upload to S3 (no large memory usage)
* Restore from backups (auto-detects format from metadata or filename)
* Backup metadata (JSON with format, mode, tables, compression info)
* List backups (table / JSON output / limit)
* Backup cleanup with retention and dry-run preview
* Scheduled backups with cron or interval jobs
* Scheduled job management (`jobs`, `jobs status`, `jobs delete`)
* Background scheduler service installation with systemd and launchd
* Console and file logging for scheduler runs
* Environment checks with `doctor`
* Safe restore with confirmation prompt
* Restore into a specific target database

---

## Installation

Install with the release script:

```bash
curl -fsSL https://raw.githubusercontent.com/valpiks/backupctl/main/install.sh | sh
backupctl version
```

Install from a released tag with Go:

```bash
go install github.com/valpiks/backupctl/cmd/backupctl@v1.0.0
```

Or install the latest released version:

```bash
go install github.com/valpiks/backupctl/cmd/backupctl@latest
```

Build from source:

```bash
git clone https://github.com/valpiks/backupctl.git
cd backupctl
go build -o backupctl ./cmd/backupctl
```

Prebuilt binaries and Linux packages (`.deb`, `.rpm`, `.apk`) are published automatically in GitHub Releases for version tags like `v1.0.0`.

For VPS installation and production service setup, see [`docs/vps-install.md`](docs/vps-install.md).

---

## Quickstart

```bash
backupctl init
backupctl doctor
backupctl backup
backupctl list
```

By default, `backupctl init` writes:

```text
~/.config/backupctl/config.yaml
~/.config/backupctl/backupctl.env
```

Use an explicit path when you want project-local config:

```bash
backupctl init --output ./backupctl.yaml
backupctl doctor -c ./backupctl.yaml
backupctl backup -c ./backupctl.yaml
```

Linux packages install a default VPS-oriented config at:

```text
/etc/backupctl/config.yaml
/etc/backupctl/backupctl.env
```

When that system config exists and no user config exists, commands can be run without `--config`.

---

## Config discovery

backupctl resolves config path in this order:

1. `--config <path>`
2. `BACKUPCTL_CONFIG`
3. `$XDG_CONFIG_HOME/backupctl/config.yaml`
4. `~/.config/backupctl/config.yaml`, when it exists
5. `/etc/backupctl/config.yaml`, when it exists
6. `~/.config/backupctl/config.yaml` as the default path for `backupctl init`

Commands that need config fail with a hint if no config path can be resolved or the config file does not exist.

---

## Shell completion

```bash
# zsh, current session
source <(backupctl completion zsh)

# zsh, persistent
backupctl completion zsh > "${fpath[1]}/_backupctl"

# bash, current session
source <(backupctl completion bash)

# fish
backupctl completion fish > ~/.config/fish/completions/backupctl.fish
```

---

## Output controls

`stdout` is reserved for command output. Diagnostic logs are written to `stderr`.

```bash
backupctl backup -c configs/config.yaml --quiet
backupctl backup -c configs/config.yaml --verbose
```

JSON output is available for script-friendly commands:

```bash
backupctl list -c configs/config.yaml --json
backupctl list -c configs/config.yaml --verified
backupctl backup -c configs/config.yaml --json
backupctl backup -c configs/config.yaml --dry-run
backupctl backup -c configs/config.yaml --verify
backupctl restore -c configs/config.yaml --file app.sql.gz --yes --json
backupctl cleanup -c configs/config.yaml --keep-last 10 --dry-run --json
backupctl jobs --json
backupctl jobs status job_20260607_120000 --json
backupctl doctor -c configs/config.yaml --json
backupctl config print -c configs/config.yaml --json
backupctl config validate -c configs/config.yaml --json
backupctl version --json
```

Doctor status colors can be controlled explicitly:

```bash
backupctl doctor --color auto
backupctl doctor --color always
backupctl doctor --color never
NO_COLOR=1 backupctl doctor
```

---

## Requirements

* Go 1.22+
* Database tools for the selected driver

PostgreSQL:

```bash
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

---

## Configuration

Create a config file and choose the active driver with `database.type` and storage with `storage.type`.

Examples in the repository:
* `configs/config.example.yaml` for PostgreSQL with local storage
* `configs/config.mongo.example.yaml` for MongoDB
* `configs/config.s3.example.yaml` for S3 storage
* `configs/config.encrypted.example.yaml` for encrypted PostgreSQL backups
* `configs/backupctl.env.example` for runtime environment variables
* `configs/config.scheduler.cron.example.yaml` for scheduled cron backups
* `configs/config.scheduler.interval.example.yaml` for scheduled interval backups

### Validate config

```bash
backupctl config validate
backupctl config validate -c ./backupctl.yaml
backupctl config validate --json
```

`config validate` checks config structure and required fields without running backup, restore, scheduler, database, or storage workflows.

### Print config summary

```bash
backupctl config print
backupctl config print -c ./backupctl.yaml
backupctl config print --json
backupctl config print --redacted
```

`backupctl config` is kept as shorthand for `backupctl config print`.

### Runtime env file

`runtime.env_file` loads environment variables before resolving `*_env` config fields.

```yaml
runtime:
  env_file: /etc/backupctl/backupctl.env
```

Example env file:

```env
BACKUPCTL_POSTGRES_PASSWORD=...
BACKUPCTL_ENCRYPTION_PASSWORD=...
AWS_ACCESS_KEY_ID=...
AWS_SECRET_ACCESS_KEY=...
```

This works for manual commands and background services because `backupctl` loads the env file itself before resolving secrets.

### Local storage (PostgreSQL)

```yaml
database:
  type: postgres
  postgres:
    host: localhost
    port: 5432
    user: postgres
    password: postgres
    name: testdb
    sslmode: disable

backup:
  type: full
  compression: gzip

storage:
  type: local
  local:
    path: ./backups

logging:
  level: info
```

### Scheduled backups

Cron schedule:

```yaml
backup:
  type: full
  compression: gzip
  scheduler:
    enabled: true
    cron: "0 3 * * *"
    log_file: ./logs/backupctl-scheduler.log
```

Interval schedule:

```yaml
backup:
  type: full
  compression: gzip
  scheduler:
    enabled: true
    interval: 24h
    log_file: ./logs/backupctl-scheduler.log
```

### Encrypted backups

Encrypted backups use AES-256-GCM. The encryption password is read from an environment variable and must be available for both backup and restore.

```yaml
runtime:
  env_file: ./backupctl.env

database:
  type: postgres
  postgres:
    password_env: BACKUPCTL_POSTGRES_PASSWORD

backup:
  type: full
  compression: gzip
  encryption:
    enabled: true
    password_env: BACKUPCTL_ENCRYPTION_PASSWORD
```

```bash
cp configs/backupctl.env.example ./backupctl.env
./backupctl backup -c configs/config.encrypted.example.yaml
```

For quick local tests you can also export the variables directly instead of using `runtime.env_file`.

Encrypted backup files use the `.enc` suffix, for example `.sql.gz.enc` or `.dump.enc`.

### Local storage (MongoDB)

```yaml
database:
  type: mongo
  mongo:
    uri: mongodb://localhost:27017
    database: app

backup:
  type: full
  compression: gzip

storage:
  type: local
  local:
    path: ./backups

logging:
  level: info
```

### S3 storage (MinIO local)

```yaml
database:
  type: postgres
  postgres:
    host: localhost
    port: 5432
    user: postgres
    password: postgres
    name: testdb
    sslmode: disable

backup:
  type: full
  compression: gzip

storage:
  type: s3
  s3:
    bucket: backupctl-test
    region: us-east-1
    prefix: backups/
    endpoint: http://localhost:9000
    force_path_style: true

logging:
  level: info
```

The provider examples below use S3-compatible API settings. Validate the final configuration against your actual provider with `backupctl doctor --deep`.

### S3 storage (AWS)

```yaml
storage:
  type: s3
  s3:
    bucket: my-production-backups
    region: eu-central-1
    prefix: backupctl/prod/
```

### S3 storage (Cloudflare R2)

```yaml
storage:
  type: s3
  s3:
    bucket: my-backups
    region: auto
    prefix: prod/
    endpoint: https://xxx.r2.cloudflarestorage.com
```

### S3 storage (DigitalOcean Spaces)

```yaml
storage:
  type: s3
  s3:
    bucket: my-backups
    region: nyc3
    prefix: prod/
    endpoint: https://nyc3.digitaloceanspaces.com
```

### S3 storage (Yandex Cloud)

```yaml
storage:
  type: s3
  s3:
    bucket: my-backups
    region: ru-central1
    prefix: prod/
    endpoint: https://storage.yandexcloud.net
```

Config notes:

* `database.type` selects the active driver (`postgres` or `mongo`)
* `database.postgres` is used only for `postgres`
* `database.mongo` is used only for `mongo`
* `backup.type` currently supports only `full`
* `backup.compression` currently supports `gzip`
* `backup.scheduler.cron` uses standard 5-field cron expressions
* `backup.scheduler.interval` uses Go durations such as `30m`, `6h`, or `24h`
* `backup.scheduler.cron` and `backup.scheduler.interval` cannot be used together
* `backup.scheduler.log_file` enables file logging for `scheduler run`
* `storage.type` supports `local` and `s3`
* `storage.local.path` is required for local storage
* `storage.s3.endpoint` is optional (defaults to AWS S3)
* `storage.s3.force_path_style` is needed for MinIO and self-hosted S3
* `storage.s3.prefix` organizes backups in virtual folders
* S3 credentials are read from `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` environment variables
* For local MinIO, default credentials `minioadmin/minioadmin` are used

---

## Usage

### Backup

```bash
./backupctl backup -c configs/config.example.yaml
```

Creates:

```text
backups/
  testdb_2026-05-14_12-00-00.sql.gz        # plain format backup
  testdb_2026-05-14_12-00-00.dump           # custom format backup
  testdb_2026-05-14_12-00-00.metadata.json  # backup metadata
```

#### Backup modes (PostgreSQL)

**Full backup (default):**

```bash
./backupctl backup -c configs/config.example.yaml
```

**Schema-only backup:**

```bash
./backupctl backup --schema-only -c configs/config.example.yaml
```

**Data-only backup:**

```bash
./backupctl backup --data-only -c configs/config.example.yaml
```

**Backup specific tables:**

```bash
./backupctl backup --tables users,orders -c configs/config.example.yaml
```

**Plain format backup (.sql.gz):**

```bash
./backupctl backup --format plain -c configs/config.example.yaml
```

**Custom format backup (.dump):**

```bash
./backupctl backup --format custom -c configs/config.example.yaml
```

Preview backup settings without connecting to the database or writing storage:

```bash
./backupctl backup --dry-run -c configs/config.example.yaml
```

Verify a backup immediately after upload:

```bash
./backupctl backup --verify -c configs/config.example.yaml
```

With S3 storage, backups are uploaded to the configured bucket and prefix.

---

### List backups

```bash
./backupctl list
```

Options:

```bash
./backupctl list --files   # show raw files
./backupctl list --json    # output as JSON
./backupctl list --limit 5 # show latest 5 backups
./backupctl list --verified # show only backups that pass metadata verification
```

---

### Verify

Verify a backup against metadata checksum and size:

```bash
./backupctl verify --file your_backup.sql.gz -c configs/config.example.yaml
./backupctl verify --file your_backup.sql.gz --json
```

Run deeper format checks:

```bash
./backupctl verify --file your_backup.sql.gz --deep
```

`--deep` validates gzip streams, decrypts encrypted backups when encryption is configured, and uses `pg_restore --list` for PostgreSQL custom-format dumps.

---

### Doctor

```bash
./backupctl doctor
```

Checks:

* config loading
* database driver init
* database connectivity
* storage init (including S3 bucket accessibility)
* optional storage write/read/delete check with `--deep`
* required database tools for the active driver

Driver-specific tool checks:

* `postgres`: `pg_dump`, `psql`, `pg_restore`
* `mongo`: `mongosh`, `mongodump`, `mongorestore`

---

### Restore

```bash
./backupctl restore --file your_backup.sql.gz -c configs/config.example.yaml
```

Validate restore without modifying the database:

```bash
./backupctl restore --file your_backup.sql.gz --dry-run
./backupctl restore --file your_backup.sql.gz --target-db restoredb --dry-run
./backupctl restore --file your_backup.sql.gz --dry-run --json
```

You will be asked for confirmation:

```text
You are about to restore:
  file:      your_backup.sql.gz
  source:    app
  target:    restoredb
  format:    plain
  encrypted: no
  metadata:  found

This may overwrite existing data. Continue? [y/N]:
```

Skip confirmation:

```bash
./backupctl restore --file your_backup.sql.gz --yes
./backupctl restore --file your_backup.sql.gz --target-db restoredb --yes
./backupctl restore --file your_backup.sql.gz --yes --json
```

The restore command auto-detects the backup format from:
1. Metadata file (preferred)
2. File extension (fallback)

Supported file extensions:
* `.sql.gz` or `.sql` → plain format
* `.dump` → custom format

---

### Cleanup

Preview deletions:

```bash
./backupctl cleanup --keep-last 5 --dry-run
```

Delete old backups and keep the latest 5:

```bash
./backupctl cleanup --keep-last 5 --yes
```

---

### Scheduled backups

Create a cron-based scheduled job:

```bash
export BACKUPCTL_POSTGRES_PASSWORD=your_password_here
./backupctl schedule --cron "0 3 * * *" -c configs/config.scheduler.cron.example.yaml
```

Create an interval-based scheduled job:

```bash
export BACKUPCTL_POSTGRES_PASSWORD=your_password_here
./backupctl schedule --interval 24h -c configs/config.scheduler.interval.example.yaml
```

List scheduled jobs:

```bash
./backupctl jobs
```

Show one job:

```bash
./backupctl jobs status job_20260522_030000
```

Delete one job:

```bash
./backupctl jobs delete job_20260522_030000
```

Run one job immediately:

```bash
./backupctl jobs run job_20260522_030000
```

Enable, disable, or inspect job logs:

```bash
./backupctl jobs disable job_20260522_030000
./backupctl jobs enable job_20260522_030000
./backupctl jobs logs job_20260522_030000 --tail 100
```

Run the scheduler process:

```bash
./backupctl scheduler run -c configs/config.scheduler.interval.example.yaml
```

`schedule` only stores the job in `.backupctl/jobs.json`. The job runs only while `scheduler run` is alive.

Install the scheduler as a background service:

```bash
./backupctl service install --config configs/config.scheduler.interval.example.yaml
```

`service install` uses the currently running `backupctl` executable as the default service binary path. If you installed with `go install`, this usually resolves to a path like `~/go/bin/backupctl`.

You can override the binary path explicitly when needed:

```bash
backupctl service install \
  --user \
  --config /path/to/config.yaml \
  --binary "$(command -v backupctl)"
```

When scheduled backups use `password_env` or encrypted backups, put `runtime.env_file` in the config. The installed service only needs the config path; `backupctl scheduler run` loads the env file itself.

```bash
sudo mkdir -p /etc/backupctl
sudo cp configs/config.encrypted.example.yaml /etc/backupctl/config.yaml
sudo cp configs/backupctl.env.example /etc/backupctl/backupctl.env
# Set runtime.env_file in /etc/backupctl/config.yaml to /etc/backupctl/backupctl.env.

backupctl service install --config /etc/backupctl/config.yaml
```

Do not put secrets directly into systemd units or launchd plists. Prefer `runtime.env_file`.

Check service status:

```bash
./backupctl service status
```

Restart or view service logs:

```bash
./backupctl service restart
./backupctl service logs --tail 100
```

Uninstall the service:

```bash
./backupctl service uninstall
```

Use `--dry-run` to preview the generated service file without installing it:

```bash
./backupctl service install --dry-run --config configs/config.scheduler.interval.example.yaml
```

On macOS, `backupctl` installs a user-level launchd service in `~/Library/LaunchAgents` by default. System-level launchd services are not supported yet, so use `--user` explicitly when you want to make that scope clear.

On Linux, `backupctl` installs a system-level systemd service in `/etc/systemd/system` by default. Use `--user` for a user-level systemd service in `~/.config/systemd/user`.

Generated service files use absolute binary/config paths and set the working directory to the directory where `service install` was run. On macOS, the launchd plist also includes a PATH that covers common Homebrew locations for PostgreSQL tools such as `pg_dump`.

Windows service installation is not supported yet. Manual commands and config/env loading work on Windows, but `backupctl service install` currently supports Linux systemd and macOS launchd only.

Useful flags:

* `--binary` sets the path to the `backupctl` binary used by the service
* `--force` overwrites an existing service file
* `--no-start` writes the service file without starting it
* `--name` changes the service name from `backupctl-scheduler`

Scheduler jobs have an in-process lock and a file lock under `.backupctl/locks` to avoid overlapping runs. Interval jobs use `run_once` missed-run behavior by default after scheduler downtime.

For production, prefer `service install` or another process supervisor instead of leaving `scheduler run` attached to a terminal.

---

## Example workflow

```bash
# check environment
./backupctl doctor

# create full backup (plain format)
./backupctl backup --format plain -c configs/config.example.yaml

# create schema-only backup for specific tables (custom format)
./backupctl backup --schema-only --tables users,orders --format custom -c configs/config.example.yaml

# list backups
./backupctl list -c configs/config.example.yaml --limit 10

# restore backup
./backupctl restore --file testdb_2026-05-14_12-00-00.sql.gz -c configs/config.example.yaml

# restore backup to different database
./backupctl restore --file testdb_2026-05-14_12-00-00.dump --target-db restoredb --yes -c configs/config.example.yaml

# cleanup old backups
./backupctl cleanup -c configs/config.example.yaml --keep-last 5 --dry-run

# create and run scheduled backups
./backupctl schedule --interval 24h -c configs/config.scheduler.interval.example.yaml
./backupctl jobs
./backupctl scheduler run -c configs/config.scheduler.interval.example.yaml
```

---

## S3 Storage Setup

### Local development with MinIO

```bash
# Start MinIO
docker run -d \
  --name minio \
  -p 9000:9000 \
  -p 9001:9001 \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin \
  minio/minio server /data --console-address ":9001"

# Create bucket
docker exec minio mc alias set local http://localhost:9000 minioadmin minioadmin
docker exec minio mc mb local/backupctl-test

# Web console at http://localhost:9001
```

### Cloud providers

For AWS, Cloudflare R2, DigitalOcean Spaces, or any S3-compatible storage, set credentials via environment variables:

```bash
export AWS_ACCESS_KEY_ID=your_access_key
export AWS_SECRET_ACCESS_KEY=your_secret_key
```

Optional S3 server-side encryption:

```yaml
storage:
  type: s3
  s3:
    bucket: backupctl-test
    region: us-east-1
    server_side_encryption: AES256
    # server_side_encryption: aws:kms
    # sse_kms_key_id: arn:aws:kms:...
```

---

## Project structure

```text
cmd/backupctl          # entrypoint
internal/app           # CLI commands
internal/backup        # backup service
internal/database      # DB drivers
internal/dbdriver      # shared DB driver interface
internal/storage       # storage layer (local, s3)
internal/compression   # compression logic
internal/config        # config loading
internal/encryption    # AES-GCM encryption logic
internal/envfile       # runtime env file parser
internal/logger        # logging
internal/scheduler     # scheduled jobs and scheduler execution
internal/secrets       # secret redaction helpers
internal/service       # systemd and launchd service installation
internal/integration   # opt-in integration tests
```

---

## Notes

* Backup is streamed (no large memory usage)
* S3 upload uses streaming via multipart upload for large files
* PostgreSQL uses native tools `pg_dump`, `psql`, and `pg_restore`
* MongoDB uses native tools `mongosh`, `mongodump`, and `mongorestore`
* Plain format is text-based and human-readable
* Custom format is optimized for restore and storage
* Encrypted backups use AES-256-GCM and the `.enc` suffix
* Restore auto-detects format and encryption from metadata or filename
* Metadata contains: format, schema-only, data-only, tables, compression, encryption
* S3-compatible providers can be used when their endpoint, region, credentials, and path-style settings work with the AWS S3 API
* Scheduled jobs are stored in `.backupctl/jobs.json`
* `scheduler run` must stay running for cron and interval jobs to execute
* `service install` keeps `scheduler run` alive in the background through systemd or launchd

---

## Integration Testing

Integration tests are opt-in and require real database tools and running database instances.

Enable integration tests:

```bash
export BACKUPCTL_RUN_INTEGRATION=1
```

### PostgreSQL smoke test

```bash
export BACKUPCTL_PG_HOST=localhost
export BACKUPCTL_PG_PORT=5432
export BACKUPCTL_PG_USER=postgres
export BACKUPCTL_PG_PASSWORD=postgres
export BACKUPCTL_PG_DB=backupctl_test

go test ./internal/integration -run TestPostgresBackupRestoreSmoke -v
```

### MongoDB smoke test

```bash
export BACKUPCTL_MONGO_URI='mongodb://localhost:27017'
export BACKUPCTL_MONGO_DB='backupctl_test'

go test ./internal/integration -run TestMongoBackupRestoreSmoke -v
```

### S3 storage test

Requires MinIO running on localhost:9000:

```bash
go test ./internal/storage/s3/ -v
```

Skip integration tests in CI:

```bash
go test -short ./...
```

---

## Releases

GitHub Releases are built automatically by GoReleaser when you push a version tag.

Local release dry-run:

```bash
goreleaser release --snapshot --clean
```

Release flow:

```bash
git tag v1.0.0
git push origin main
git push origin v1.0.0
```

The workflow publishes archives, Linux packages, and checksums to the GitHub Release page for that tag.

---

## Security

* Do NOT commit real config files with passwords
* Use `config.example.yaml` for sharing config structure
* Use `database.postgres.password_env` for PostgreSQL passwords
* Use `database.mongo.uri_env` for MongoDB connection strings
* Use `backup.encryption.password_env` for backup encryption passwords
* Set S3 credentials via environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
* Use `.env` files locally (added to `.gitignore`)

PostgreSQL password from an environment variable:

```yaml
database:
  type: postgres
  postgres:
    host: localhost
    port: 5432
    user: postgres
    password_env: BACKUPCTL_POSTGRES_PASSWORD
    name: app
    sslmode: disable
```

MongoDB URI from an environment variable:

```yaml
database:
  type: mongo
  mongo:
    uri_env: BACKUPCTL_MONGO_URI
    database: app
```

Backup encryption password from an environment variable:

```yaml
backup:
  encryption:
    enabled: true
    password_env: BACKUPCTL_ENCRYPTION_PASSWORD
```
