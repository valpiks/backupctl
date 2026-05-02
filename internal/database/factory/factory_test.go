package factory_test

import (
	"testing"

	"github.com/valpiks/backupctl/internal/config"
	dbfactory "github.com/valpiks/backupctl/internal/database/factory"
)

func TestNewDriver(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     config.DatabaseConfig
		wantErr string
	}{
		{
			name: "valid postgres driver",
			cfg: config.DatabaseConfig{
				Type: "postgres",
				Postgres: config.PostgresConfig{
					Host: "localhost",
					Port: 5432,
					User: "postgres",
					Name: "app",
				},
			},
		},
		{
			name: "valid mongo driver",
			cfg: config.DatabaseConfig{
				Type: "mongo",
				Mongo: config.MongoConfig{
					URI:      "mongodb://localhost:27017",
					Database: "app",
				},
			},
		},
		{
			name: "unsupported driver type",
			cfg: config.DatabaseConfig{
				Type: "sqlite",
			},
			wantErr: "unsupported database type: sqlite",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			driver, err := dbfactory.NewDriver(tt.cfg)

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("NewDriver() error = %v", err)
				}
				if driver == nil {
					t.Fatal("NewDriver() returned nil driver")
				}
				return
			}

			if err == nil {
				t.Fatalf("NewDriver() error = nil, want %q", tt.wantErr)
			}

			if driver != nil {
				t.Fatal("NewDriver() returned non-nil driver on error")
			}

			if err.Error() != tt.wantErr {
				t.Fatalf("NewDriver() error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}
