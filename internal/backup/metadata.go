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
}
