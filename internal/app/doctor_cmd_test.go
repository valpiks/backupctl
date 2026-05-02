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
