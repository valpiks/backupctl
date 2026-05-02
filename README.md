# backupctl

CLI utility for database backups and restore (PostgreSQL).

## Version

```bash
backupctl version
```

## Features

* Full database backups using `pg_dump`
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
* PostgreSQL client tools:

  * `pg_dump`
  * `psql`

Check installation:

```bash
pg_dump --version
psql --version
```

---

## Configuration

Create a config file (do not commit real credentials):

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
* `pg_dump` and `psql` presence

---

### Restore

```bash
./backupctl restore --file backups/backup.sql.gz
```

You will be asked for confirmation:

```text
WARNING: you are about to restore database "testdb"
This may overwrite existing data. Continue? [y/N]:
```

Skip confirmation:

```bash
./backupctl restore --file backups/backup.sql.gz --yes
./backupctl restore --file backups/backup.sql.gz --target-db restoredb
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
./backupctl backup

# list backups
./backupctl list --limit 10

# restore backup
./backupctl restore --file backups/your_backup.sql.gz --target-db restoredb

# cleanup old backups
./backupctl cleanup --keep-last 5 --dry-run
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
* Uses native PostgreSQL tools (`pg_dump`, `psql`)
* Requires PostgreSQL client tools installed locally
* Designed to be extended (MySQL, S3, scheduler, etc.)

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
