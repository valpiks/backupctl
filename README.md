# backupctl

CLI utility for database backups and restore with pluggable database drivers.

## Version

```bash
backupctl version
```

## Features

* Full database backups for PostgreSQL and MongoDB
* Streaming gzip compression (low memory usage)
* Local file storage
* Restore from backups
* Backup metadata (JSON)
* List backups (table / JSON output / limit)
* Backup cleanup with retention and dry-run preview
* Environment checks with `doctor`
* Safe restore with confirmation prompt
* Restore into a specific target database

---

## Installation

```bash
git clone https://github.com/valpiks/backupctl.git
cd backupctl
go build -o backupctl ./cmd/backupctl
```

---

## Requirements

* Go 1.22+
* Database tools for the selected driver

PostgreSQL:

```bash
pg_dump --version
psql --version
```

MongoDB:

```bash
mongosh --version
mongodump --version
mongorestore --version
```

---

## Configuration

Create a config file and choose the active driver with `database.type`.

Examples in the repository:

* `configs/config.example.yaml` for PostgreSQL
* `configs/config.mongo.example.yaml` for MongoDB

PostgreSQL config:

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
  path: ./backups

logging:
  level: info
```

MongoDB config:

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
  path: ./backups

logging:
  level: info
```

Config notes:

* `database.type` selects the active driver
* `database.postgres` is used only for `postgres`
* `database.mongo` is used only for `mongo`
* `backup.type` currently supports only `full`
* `storage.type` currently supports only `local`
* `backup.compression` is currently `gzip`

---

## Usage

### Backup

```bash
./backupctl backup -c configs/config.local.yaml
```

Creates:

```text
backups/
  testdb_2026-04-27_16-02-07.sql.gz
  testdb_2026-04-27_16-02-07.metadata.json
```

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
* storage init
* required database tools for the active driver

Driver-specific tool checks:

* `postgres`: `pg_dump`, `psql`
* `mongo`: `mongosh`, `mongodump`, `mongorestore`

---

### Restore

```bash
./backupctl restore --file your_backup.sql.gz
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

# create backup
./backupctl backup -c configs/config.local.yaml

# list backups
./backupctl list -c configs/config.local.yaml --limit 10

# restore backup
./backupctl restore -c configs/config.local.yaml --file your_backup.sql.gz --target-db restoredb

# cleanup old backups
./backupctl cleanup -c configs/config.local.yaml --keep-last 5 --dry-run
```

---

## Project structure

```text
cmd/backupctl        # entrypoint
internal/app         # CLI commands
internal/backup      # backup service
internal/database    # DB drivers
internal/storage     # storage layer
internal/compression # compression logic
internal/config      # config loading
internal/logger      # logging
```

---

## Notes

* Backup is streamed (no large memory usage)
* PostgreSQL uses native tools `pg_dump` and `psql`
* MongoDB uses native tools `mongosh`, `mongodump`, and `mongorestore`
* Restore `--file` expects a backup file name from the configured storage path
* Designed to be extended with more database and storage drivers

---

## Security

* Do NOT commit real config files with passwords
* Use `config.example.yaml` for sharing config structure
* Use environment variables for production setups

---

## 6. Roadmap

```md
## Roadmap

- [ ] S3 / cloud storage support
- [ ] scheduler (cron-based backups)
- [ ] incremental backups
- [ ] MySQL support
- [ ] encryption for backups
```
