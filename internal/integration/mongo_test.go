package integration

import (
	"context"
	"fmt"
	"os/exec"
	"testing"

	"github.com/valpiks/backupctl/internal/backup"
	"github.com/valpiks/backupctl/internal/compression"
	"github.com/valpiks/backupctl/internal/config"
	dbfactory "github.com/valpiks/backupctl/internal/database/factory"
	database "github.com/valpiks/backupctl/internal/dbdriver"
	storagefactory "github.com/valpiks/backupctl/internal/storage/factory"
)

func integrationMongoConfig(t *testing.T) config.Config {
	t.Helper()

	return config.Config{
		Database: config.DatabaseConfig{
			Type: "mongo",
			Mongo: config.MongoConfig{
				URI:      requiredEnv(t, "BACKUPCTL_MONGO_URI"),
				Database: requiredEnv(t, "BACKUPCTL_MONGO_DB"),
			},
		},
		Backup: config.BackupConfig{
			Type:        "full",
			Compression: "gzip",
		},
		Storage: config.StorageConfig{
			Type: "local",
			Path: t.TempDir(),
		},
		Logging: config.LoggingConfig{
			Level: "info",
		},
	}
}

func runMongoShell(t *testing.T, uri string, script string) {
	t.Helper()

	args := []string{
		uri,
		"--quiet",
		"--eval", script,
	}

	cmd := exec.Command("mongosh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("mongosh failed: %v: %s", err, string(output))
	}
}

func dropMongoDatabase(t *testing.T, uri string, dbName string) {
	t.Helper()

	runMongoShell(t, uri, fmt.Sprintf(`db.getSiblingDB(%q).dropDatabase()`, dbName))
}

func seedMongoData(t *testing.T, uri string, dbName string) {
	t.Helper()

	runMongoShell(t, uri, fmt.Sprintf(`
		const dbRef = db.getSiblingDB(%q);
		dbRef.integration_users.deleteMany({});
		dbRef.integration_users.insertOne({ name: "alice" });
	`, dbName))
}

func assertMongoCollectionHasDocuments(t *testing.T, uri string, dbName string, collection string) {
	t.Helper()

	script := fmt.Sprintf(`
		const count = db.getSiblingDB(%q).getCollection(%q).countDocuments({});
		if (count < 1) {
			quit(1);
		}
	`, dbName, collection)

	runMongoShell(t, uri, script)
}

func TestMongoBackupRestoreSmoke(t *testing.T) {
	requireIntegrationTest(t)

	cfg := integrationMongoConfig(t)
	targetDB := "backupctl_restore_smoke"

	dropMongoDatabase(t, cfg.Database.Mongo.URI, cfg.Database.Mongo.Database)
	dropMongoDatabase(t, cfg.Database.Mongo.URI, targetDB)
	t.Cleanup(func() {
		dropMongoDatabase(t, cfg.Database.Mongo.URI, targetDB)
		dropMongoDatabase(t, cfg.Database.Mongo.URI, cfg.Database.Mongo.Database)
	})

	seedMongoData(t, cfg.Database.Mongo.URI, cfg.Database.Mongo.Database)

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
		DatabaseName: cfg.Database.ActiveDatabaseName(),
		BackupType:   cfg.Backup.Type,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

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

	assertMongoCollectionHasDocuments(t, cfg.Database.Mongo.URI, targetDB, "integration_users")
}
