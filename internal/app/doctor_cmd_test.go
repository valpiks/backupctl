package app

import (
	"reflect"
	"testing"
)

func TestRequiredDatabaseTools(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		databaseType string
		want         []string
	}{
		{
			name:         "postgres",
			databaseType: "postgres",
			want:         []string{"psql", "pg_dump"},
		},
		{
			name:         "mongo",
			databaseType: "mongo",
			want:         []string{"mongosh", "mongodump", "mongorestore"},
		},
		{
			name:         "unknown",
			databaseType: "sqlite",
			want:         []string{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := requiredDatabaseTools(tt.databaseType)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("requiredDatabaseTools(%q) = %v, want %v", tt.databaseType, got, tt.want)
			}
		})
	}
}

func TestDetectFormatFromEncryptedName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		fileName string
		want     string
	}{
		{name: "plain gzip encrypted", fileName: "app.sql.gz.enc", want: "plain"},
		{name: "custom encrypted", fileName: "app.dump.enc", want: "custom"},
		{name: "plain gzip", fileName: "app.sql.gz", want: "plain"},
		{name: "custom", fileName: "app.dump", want: "custom"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := detectFormatFromName(tt.fileName); got != tt.want {
				t.Fatalf("detectFormatFromName(%q) = %q, want %q", tt.fileName, got, tt.want)
			}
		})
	}
}

func TestBackupNameWithoutEncryptionSuffix(t *testing.T) {
	t.Parallel()

	if got := backupNameWithoutEncryptionSuffix("app.sql.gz.enc"); got != "app.sql.gz" {
		t.Fatalf("backupNameWithoutEncryptionSuffix() = %q", got)
	}
}
