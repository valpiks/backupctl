package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"testing"

	"github.com/valpiks/backupctl/internal/backup"
	"github.com/valpiks/backupctl/internal/compression"
	"github.com/valpiks/backupctl/internal/config"
	dbfactory "github.com/valpiks/backupctl/internal/database/factory"
	database "github.com/valpiks/backupctl/internal/dbdriver"
	storagefactory "github.com/valpiks/backupctl/internal/storage/factory"
)

func requireIntegrationTest(t *testing.T) {
	t.Helper()

	if os.Getenv("BACKUPCTL_RUN_INTEGRATION") != "1" {
		t.Skip("set BACKUPCTL_RUN_INTEGRATION=1 to run integration tests")
	}
}

func requiredEnv(t *testing.T, key string) string {
	t.Helper()

	value := os.Getenv(key)
	if value == "" {
		t.Fatalf("environment variable %s is required", key)
	}

	return value
}

func integrationPostgresConfig(t *testing.T) config.Config {
	t.Helper()

	port, err := strconv.Atoi(requiredEnv(t, "BACKUPCTL_PG_PORT"))
	if err != nil {
		t.Fatalf("invalid BACKUPCTL_PG_PORT: %v", err)
	}

	return config.Config{
		Database: config.DatabaseConfig{
			Type: "postgres",
			Postgres: config.PostgresConfig{
				Host:     requiredEnv(t, "BACKUPCTL_PG_HOST"),
				Port:     port,
				User:     requiredEnv(t, "BACKUPCTL_PG_USER"),
				Password: requiredEnv(t, "BACKUPCTL_PG_PASSWORD"),
				Name:     requiredEnv(t, "BACKUPCTL_PG_DB"),
				SSLMode:  "disable",
			},
		},
		Backup: config.BackupConfig{
			Type:        "full",
			Compression: "gzip",
		},
		Storage: config.StorageConfig{
			Type:  "local",
			Local: config.LocalStorageConfig{Path: t.TempDir()},
		},
		Logging: config.LoggingConfig{
			Level: "info",
		},
	}
}

func runPSQL(t *testing.T, cfg config.PostgresConfig, dbName string, sql string) {
	t.Helper()

	args := []string{
		"-h", cfg.Host,
		"-p", strconv.Itoa(cfg.Port),
		"-U", cfg.User,
		"-d", dbName,
		"-c", sql,
	}

	cmd := exec.Command("psql", args...)
	cmd.Env = append(os.Environ(), "PGPASSWORD="+cfg.Password)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("psql failed: %v: %s", err, string(output))
	}
}

func createDatabase(t *testing.T, cfg config.PostgresConfig, dbName string) {
	t.Helper()

	runPSQL(t, cfg, "postgres", fmt.Sprintf("DROP DATABASE IF EXISTS %s;", dbName))
	runPSQL(t, cfg, "postgres", fmt.Sprintf("CREATE DATABASE %s;", dbName))
}

func clearTestData(t *testing.T, cfg config.PostgresConfig, targetDb string) {
	t.Helper()

	runPSQL(t, cfg, "postgres", fmt.Sprintf("DROP DATABASE IF EXISTS %s;", targetDb))
	runPSQL(t, cfg, "postgres", fmt.Sprintf("DROP DATABASE IF EXISTS %s;", cfg.Name))
}

func TestPostgresBackupRestoreSmoke(t *testing.T) {
	requireIntegrationTest(t)

	cfg := integrationPostgresConfig(t)
	targetDB := "backupctl_restore_smoke"

	createDatabase(t, cfg.Database.Postgres, cfg.Database.Postgres.Name)

	runPSQL(t, cfg.Database.Postgres, cfg.Database.Postgres.Name, `
  		CREATE TABLE IF NOT EXISTS integration_users (
  			id SERIAL PRIMARY KEY,
  			name TEXT NOT NULL
  		);
  	`)
	runPSQL(t, cfg.Database.Postgres, cfg.Database.Postgres.Name, `
  		INSERT INTO integration_users (name) VALUES ('alice');
  	`)

	driver, err := dbfactory.NewDriver(cfg.Database)
	if err != nil {
		t.Fatalf("NewDriver() error = %v", err)
	}

	storage, err := storagefactory.NewStorage(cfg.Storage)
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	service := backup.NewService(driver, storage, compression.NewGzipCompressor())

	result, err := service.Run(context.Background(), backup.Options{
		DatabaseName: cfg.Database.Postgres.Name,
		BackupType:   cfg.Backup.Type,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	createDatabase(t, cfg.Database.Postgres, targetDB)

	reader, err := storage.Open(context.Background(), result.FileName)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer reader.Close()

	decompressedReader, err := compression.NewGzipCompressor().Decompress(reader)
	if err != nil {
		t.Fatalf("Decompress() error = %v", err)
	}
	defer decompressedReader.Close()

	err = driver.Restore(context.Background(), decompressedReader, database.RestoreOptions{
		TargetDatabase: targetDB,
	})
	if err != nil {
		t.Fatalf("Restore() error = %v", err)
	}

	runPSQL(t, cfg.Database.Postgres, targetDB, "SELECT count(*) FROM integration_users;")

	clearTestData(t, cfg.Database.Postgres, targetDB)
}
