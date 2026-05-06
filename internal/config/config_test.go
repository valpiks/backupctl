package config

import (
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

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	return path
}
