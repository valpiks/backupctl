package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadNestedPostgresConfig(t *testing.T) {
	t.Parallel()

	configPath := writeTempConfig(t, `
database:
  type: postgres
  postgres:
    host: localhost
    port: 5432
    user: postgres
    password: secret
    name: app
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
`)

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Database.Type != "postgres" {
		t.Fatalf("Database.Type = %q", cfg.Database.Type)
	}

	if cfg.Database.Postgres.Host != "localhost" {
		t.Fatalf("Postgres.Host = %q", cfg.Database.Postgres.Host)
	}

	if cfg.Database.Postgres.Name != "app" {
		t.Fatalf("Postgres.Name = %q", cfg.Database.Postgres.Name)
	}

	if cfg.Database.Postgres.SSLMode != "disable" {
		t.Fatalf("Postgres.SSLMode = %q", cfg.Database.Postgres.SSLMode)
	}
}

func TestLoadNestedMongoConfig(t *testing.T) {
	t.Parallel()

	configPath := writeTempConfig(t, `
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
`)

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Database.Type != "mongo" {
		t.Fatalf("Database.Type = %q", cfg.Database.Type)
	}

	if cfg.Database.Mongo.URI != "mongodb://localhost:27017" {
		t.Fatalf("Mongo.URI = %q", cfg.Database.Mongo.URI)
	}

	if cfg.Database.Mongo.Database != "app" {
		t.Fatalf("Mongo.Database = %q", cfg.Database.Mongo.Database)
	}
}

func TestLoadInvalidPostgresConfig(t *testing.T) {
	t.Parallel()

	configPath := writeTempConfig(t, `
database:
  type: postgres
  postgres:
    host: localhost
    port: 5432
    user: postgres
    password: secret
backup:
  type: full
  compression: gzip
storage:
  type: local
  local:
    path: ./backups
logging:
  level: info
`)

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Load() error = nil, want validation error")
	}

	if !strings.Contains(err.Error(), "postgres.name is required") {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestLoadInvalidMongoConfig(t *testing.T) {
	t.Parallel()

	configPath := writeTempConfig(t, `
database:
  type: mongo
  mongo:
    uri: mongodb://localhost:27017
backup:
  type: full
  compression: gzip
storage:
  type: local
  local:
    path: ./backups
logging:
  level: info
`)

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Load() error = nil, want validation error")
	}

	if !strings.Contains(err.Error(), "mongo.database is required") {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestLoadPostgresPasswordEnv(t *testing.T) {
	t.Setenv("BACKUPCTL_POSTGRES_PASSWORD", "from-env")

	configPath := writeTempConfig(t, `
database:
  type: postgres
  postgres:
    host: localhost
    port: 5432
    user: postgres
    password_env: BACKUPCTL_POSTGRES_PASSWORD
    name: app
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
`)

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Database.Postgres.Password != "from-env" {
		t.Fatalf("Postgres.Password = %q, want from-env", cfg.Database.Postgres.Password)
	}
}

func TestLoadPostgresPasswordAndPasswordEnvConflict(t *testing.T) {
	t.Setenv("BACKUPCTL_POSTGRES_PASSWORD", "from-env")

	configPath := writeTempConfig(t, `
database:
  type: postgres
  postgres:
    host: localhost
    port: 5432
    user: postgres
    password: direct
    password_env: BACKUPCTL_POSTGRES_PASSWORD
    name: app
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
`)

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Load() error = nil")
	}

	if !strings.Contains(err.Error(), "postgres.password and postgres.password_env cannot be used together") {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestLoadPostgresPasswordEnvMissing(t *testing.T) {
	configPath := writeTempConfig(t, `
database:
  type: postgres
  postgres:
    host: localhost
    port: 5432
    user: postgres
    password_env: BACKUPCTL_MISSING_PASSWORD
    name: app
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
`)

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Load() error = nil")
	}

	if !strings.Contains(err.Error(), "environment variable BACKUPCTL_MISSING_PASSWORD is not set") {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestLoadMongoURIEnv(t *testing.T) {
	t.Setenv("BACKUPCTL_MONGO_URI", "mongodb://env-host:27017")

	configPath := writeTempConfig(t, `
database:
  type: mongo
  mongo:
    uri_env: BACKUPCTL_MONGO_URI
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
`)

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Database.Mongo.URI != "mongodb://env-host:27017" {
		t.Fatalf("Mongo.URI = %q, want mongodb://env-host:27017", cfg.Database.Mongo.URI)
	}
}

func TestLoadMongoURIAndURIEnvConflict(t *testing.T) {
	t.Setenv("BACKUPCTL_MONGO_URI", "mongodb://env-host:27017")

	configPath := writeTempConfig(t, `
database:
  type: mongo
  mongo:
    uri: mongodb://localhost:27017
    uri_env: BACKUPCTL_MONGO_URI
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
`)

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Load() error = nil")
	}

	if !strings.Contains(err.Error(), "mongo.uri and mongo.uri_env cannot be used together") {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestLoadMongoURIEnvMissing(t *testing.T) {
	configPath := writeTempConfig(t, `
database:
  type: mongo
  mongo:
    uri_env: BACKUPCTL_MISSING_MONGO_URI
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
`)

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Load() error = nil")
	}

	if !strings.Contains(err.Error(), "environment variable BACKUPCTL_MISSING_MONGO_URI is not set") {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestLoadEncryptionPasswordEnv(t *testing.T) {
	t.Setenv("BACKUPCTL_ENCRYPTION_PASSWORD", "encrypt-secret")

	configPath := writeTempConfig(t, `
database:
  type: postgres
  postgres:
    host: localhost
    port: 5432
    user: postgres
    name: app
backup:
  type: full
  compression: gzip
  encryption:
    enabled: true
    password_env: BACKUPCTL_ENCRYPTION_PASSWORD
storage:
  type: local
  local:
    path: ./backups
logging:
  level: info
`)

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Backup.Encryption == nil {
		t.Fatal("Backup.Encryption = nil")
	}

	if cfg.Backup.Encryption.Password != "encrypt-secret" {
		t.Fatalf("Encryption.Password = %q, want encrypt-secret", cfg.Backup.Encryption.Password)
	}
}

func TestLoadEncryptionEnabledWithoutPasswordEnv(t *testing.T) {
	configPath := writeTempConfig(t, `
database:
  type: postgres
  postgres:
    host: localhost
    port: 5432
    user: postgres
    name: app
backup:
  type: full
  compression: gzip
  encryption:
    enabled: true
storage:
  type: local
  local:
    path: ./backups
logging:
  level: info
`)

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Load() error = nil")
	}

	if !strings.Contains(err.Error(), "backup.encryption.password_env is required") {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestLoadRuntimeEnvFile(t *testing.T) {
	t.Setenv("BACKUPCTL_POSTGRES_PASSWORD", "")
	t.Setenv("BACKUPCTL_ENCRYPTION_PASSWORD", "")

	dir := t.TempDir()

	envPath := filepath.Join(dir, "backupctl.env")
	if err := os.WriteFile(envPath, []byte(`
BACKUPCTL_POSTGRES_PASSWORD=from-env-file
BACKUPCTL_ENCRYPTION_PASSWORD=encrypt-from-env-file
`), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	configPath := writeTempConfig(t, fmt.Sprintf(`
runtime:
  env_file: %q
database:
  type: postgres
  postgres:
    host: localhost
    port: 5432
    user: postgres
    name: app
    password_env: BACKUPCTL_POSTGRES_PASSWORD
backup:
  type: full
  compression: gzip
  encryption:
    enabled: true
    password_env: BACKUPCTL_ENCRYPTION_PASSWORD
storage:
  type: local
  local:
    path: ./backups
logging:
  level: info
`, envPath))

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Database.Postgres.Password != "from-env-file" {
		t.Fatalf("Postgres.Password = %q, want from-env-file", cfg.Database.Postgres.Password)
	}

	if cfg.Backup.Encryption == nil {
		t.Fatal("Backup.Encryption = nil")
	}

	if cfg.Backup.Encryption.Password != "encrypt-from-env-file" {
		t.Fatalf("Encryption.Password = %q, want encrypt-from-env-file", cfg.Backup.Encryption.Password)
	}
}

func TestActiveDatabaseName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  DatabaseConfig
		want string
	}{
		{
			name: "postgres",
			cfg: DatabaseConfig{
				Type: "postgres",
				Postgres: PostgresConfig{
					Name: "app",
				},
			},
			want: "app",
		},
		{
			name: "mongo",
			cfg: DatabaseConfig{
				Type: "mongo",
				Mongo: MongoConfig{
					Database: "analytics",
				},
			},
			want: "analytics",
		},
		{
			name: "unknown",
			cfg: DatabaseConfig{
				Type: "sqlite",
			},
			want: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.cfg.ActiveDatabaseName(); got != tt.want {
				t.Fatalf("ActiveDatabaseName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestKnownSecrets(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Database: DatabaseConfig{
			Type: "postgres",
			Postgres: PostgresConfig{
				Password: "postgres-secret",
			},
			Mongo: MongoConfig{
				URI: "mongodb://user:mongo-secret@localhost:27017/app",
			},
		},
		Backup: BackupConfig{
			Encryption: &EncryptionConfig{
				Enabled:  true,
				Password: "encrypt-secret",
			},
		},
	}

	got := cfg.KnownSecrets()
	want := []string{"postgres-secret", "encrypt-secret"}

	if len(got) != len(want) {
		t.Fatalf("KnownSecrets() = %v, want %v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("KnownSecrets() = %v, want %v", got, want)
		}
	}
}

func TestLoadSchedulerConfig(t *testing.T) {
	t.Parallel()

	configPath := writeTempConfig(t, `
database:
  type: postgres
  postgres:
    host: localhost
    port: 5432
    user: postgres
    name: app
backup:
  type: full
  compression: gzip
  scheduler:
    enabled: true
    interval: 24h
    log_file: ./logs/backupctl.log
storage:
  type: local
  local:
    path: ./backups
logging:
  level: info
`)

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Backup.Scheduler == nil {
		t.Fatal("Backup.Scheduler = nil")
	}

	if !cfg.Backup.Scheduler.Enabled {
		t.Fatal("Backup.Scheduler.Enabled = false, want true")
	}

	if cfg.Backup.Scheduler.Interval != "24h" {
		t.Fatalf("Backup.Scheduler.Interval = %q, want 24h", cfg.Backup.Scheduler.Interval)
	}
}

func TestLoadSchedulerConfigValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		block   string
		wantErr string
	}{
		{
			name: "enabled without schedule",
			block: `
  scheduler:
    enabled: true`,
			wantErr: "either backup.scheduler.interval or backup.scheduler.cron is required",
		},
		{
			name: "cron and interval together",
			block: `
  scheduler:
    enabled: true
    cron: "0 3 * * *"
    interval: 24h`,
			wantErr: "backup.scheduler.interval and backup.scheduler.cron cannot be used together",
		},
		{
			name: "invalid interval",
			block: `
  scheduler:
    enabled: true
    interval: nope`,
			wantErr: "invalid backup.scheduler.interval",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			configPath := writeTempConfig(t, `
database:
  type: postgres
  postgres:
    host: localhost
    port: 5432
    user: postgres
    name: app
backup:
  type: full
  compression: gzip
`+tt.block+`
storage:
  type: local
  local:
    path: ./backups
logging:
  level: info
`)

			_, err := Load(configPath)
			if err == nil {
				t.Fatal("Load() error = nil, want validation error")
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Load() error = %v, want to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestValidateStructureRejectsUnsupportedDatabaseType(t *testing.T) {
	t.Parallel()

	cfg := validPostgresConfig()
	cfg.Database.Type = "mysql"

	err := cfg.ValidateStructure()
	if err == nil || !strings.Contains(err.Error(), "unsupported database.type") {
		t.Fatalf("ValidateStructure() error = %v, want unsupported database.type", err)
	}
}

func TestValidateStructureRejectsUnsupportedStorageType(t *testing.T) {
	t.Parallel()

	cfg := validPostgresConfig()
	cfg.Storage.Type = "ftp"

	err := cfg.ValidateStructure()
	if err == nil || !strings.Contains(err.Error(), "unsupported storage.type") {
		t.Fatalf("ValidateStructure() error = %v, want unsupported storage.type", err)
	}
}

func TestValidateStructureRejectsMissingMongoURI(t *testing.T) {
	t.Parallel()

	cfg := validPostgresConfig()
	cfg.Database = DatabaseConfig{
		Type: "mongo",
		Mongo: MongoConfig{
			Database: "app",
		},
	}

	err := cfg.ValidateStructure()
	if err == nil || !strings.Contains(err.Error(), "database.mongo.uri or database.mongo.uri_env is required") {
		t.Fatalf("ValidateStructure() error = %v, want missing mongo uri error", err)
	}
}

func TestValidateStructureAcceptsMongoURIEnv(t *testing.T) {
	t.Parallel()

	cfg := validPostgresConfig()
	cfg.Database = DatabaseConfig{
		Type: "mongo",
		Mongo: MongoConfig{
			URIEnv:   "BACKUPCTL_MONGO_URI",
			Database: "app",
		},
	}

	if err := cfg.ValidateStructure(); err != nil {
		t.Fatalf("ValidateStructure() error = %v", err)
	}
}

func TestValidateStructureRejectsInvalidCompression(t *testing.T) {
	t.Parallel()

	cfg := validPostgresConfig()
	cfg.Backup.Compression = "zip"

	err := cfg.ValidateStructure()
	if err == nil || !strings.Contains(err.Error(), "unsupported backup.compression") {
		t.Fatalf("ValidateStructure() error = %v, want unsupported backup.compression", err)
	}
}

func TestValidateStructureAcceptsEncryptionPasswordEnv(t *testing.T) {
	t.Parallel()

	cfg := validPostgresConfig()
	cfg.Backup.Encryption = &EncryptionConfig{
		Enabled:     true,
		PasswordEnv: "BACKUPCTL_ENCRYPTION_PASSWORD",
	}

	if err := cfg.ValidateStructure(); err != nil {
		t.Fatalf("ValidateStructure() error = %v", err)
	}
}

func TestValidateRequiresResolvedEncryptionPassword(t *testing.T) {
	t.Parallel()

	cfg := validPostgresConfig()
	cfg.Backup.Encryption = &EncryptionConfig{
		Enabled:     true,
		PasswordEnv: "BACKUPCTL_ENCRYPTION_PASSWORD",
	}

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "backup.encryption password is required") {
		t.Fatalf("Validate() error = %v, want resolved encryption password error", err)
	}
}

func validPostgresConfig() *Config {
	return &Config{
		Database: DatabaseConfig{
			Type: "postgres",
			Postgres: PostgresConfig{
				Host: "localhost",
				Port: 5432,
				User: "postgres",
				Name: "app",
			},
		},
		Backup: BackupConfig{
			Type:        "full",
			Compression: "gzip",
		},
		Storage: StorageConfig{
			Type: "local",
			Local: LocalStorageConfig{
				Path: "./backups",
			},
		},
		Logging: LoggingConfig{
			Level: "info",
		},
	}
}

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	return path
}
