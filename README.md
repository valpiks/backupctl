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
* Environment checks with `doctor`
* Safe restore with confirmation prompt
* Restore into a specific target database

---

## Installation

Install from a released tag with Go:

```bash
go install github.com/valpiks/backupctl/cmd/backupctl@v0.6.0
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

Prebuilt binaries are published automatically in GitHub Releases for version tags like `v0.6.0`.

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
```

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
* required database tools for the active driver

Driver-specific tool checks:

* `postgres`: `pg_dump`, `psql`, `pg_restore`
* `mongo`: `mongosh`, `mongodump`, `mongorestore`

---

### Restore

```bash
./backupctl restore --file your_backup.sql.gz -c configs/config.example.yaml
```

You will be asked for confirmation:

```text
WARNING: you are about to restore database "testdb"
This may overwrite existing data. Continue? [y/N]:
```

Skip confirmation:

```bash
./backupctl restore --file your_backup.sql.gz --yes
./backupctl restore --file your_backup.sql.gz --target-db restoredb
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
./backupctl cleanup --keep-last 5
```

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

---

## Project structure

```text
cmd/backupctl          # entrypoint
internal/app           # CLI commands
internal/backup        # backup service
internal/database      # DB drivers
internal/storage       # storage layer (local, s3)
internal/compression   # compression logic
internal/config        # config loading
internal/logger        # logging
```

---

## Notes

* Backup is streamed (no large memory usage)
* S3 upload uses streaming via multipart upload for large files
* PostgreSQL uses native tools `pg_dump`, `psql`, and `pg_restore`
* MongoDB uses native tools `mongosh`, `mongodump`, and `mongorestore`
* Plain format is text-based and human-readable
* Custom format is optimized for restore and storage
* Restore auto-detects format from metadata or filename
* Metadata contains: format, schema-only, data-only, tables, compression
* All S3-compatible providers are supported (AWS, MinIO, R2, DigitalOcean, Yandex Cloud, etc.)

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
git tag v0.6.0
git push origin main
git push origin v0.6.0
```

The workflow publishes archives and checksums to the GitHub Release page for that tag.

---

## Security

* Do NOT commit real config files with passwords
* Use `config.example.yaml` for sharing config structure
* Set credentials via environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
* Use `.env` files locally (added to `.gitignore`)

---

## Roadmap

- [x] S3 / cloud storage support
- [x] PostgreSQL multiple backup modes and formats
- [ ] scheduler (cron-based backups)
- [ ] incremental backups
- [ ] MySQL support
- [ ] encryption for backups
- [ ] backup verification
- [ ] S3 lifecycle policies integration
