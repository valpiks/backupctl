package backup

import "time"

type Metadata struct {
	DatabaseName string    `json:"database_name"`
	BackupType   string    `json:"backup_type"`
	FileName     string    `json:"file_name"`
	Status       string    `json:"status"`
	StartedAt    time.Time `json:"started_at"`
	EndedAt      time.Time `json:"ended_at"`
	Duration     string    `json:"duration"`

	Format      string   `json:"format"`
	SchemaOnly  bool     `json:"schema_only"`
	DataOnly    bool     `json:"data_only"`
	Tabels      []string `json:"tables,omitempty"`
	Compression string   `json:"compression,omitempty"`
}
