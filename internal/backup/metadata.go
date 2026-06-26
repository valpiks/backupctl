package backup

import "time"

type Metadata struct {
	DatabaseName string    `json:"database_name"`
	BackupType   string    `json:"backup_type"`
	FileName     string    `json:"file_name"`
	FileSize     int64     `json:"file_size"`
	SHA256       string    `json:"sha256"`
	Status       string    `json:"status"`
	StartedAt    time.Time `json:"started_at"`
	EndedAt      time.Time `json:"ended_at"`
	Duration     string    `json:"duration"`

	Format      string              `json:"format"`
	SchemaOnly  bool                `json:"schema_only"`
	DataOnly    bool                `json:"data_only"`
	Tables      []string            `json:"tables,omitempty"`
	Compression string              `json:"compression,omitempty"`
	Encryption  *EncryptionMetadata `json:"encryption,omitempty"`

	BackupctlVersion string `json:"backupctl_version,omitempty"`
	Hostname         string `json:"hostname,omitempty"`
}

type EncryptionMetadata struct {
	Enabled   bool   `json:"enabled"`
	Algorithm string `json:"algorithm"`
	KDF       string `json:"kdf"`
}
