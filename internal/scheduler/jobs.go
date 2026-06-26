package scheduler

import (
	"context"
	"time"
)

type Job struct {
	ID              string     `json:"id"`
	Cron            string     `json:"cron,omitempty"`
	Interval        string     `json:"interval,omitempty"`
	DatabaseName    string     `json:"database_name"`
	BackupType      string     `json:"backup_type"`
	Format          string     `json:"format"`
	Status          string     `json:"status"`
	Disabled        bool       `json:"disabled,omitempty"`
	MissedRunPolicy string     `json:"missed_run_policy,omitempty"`
	LastRun         *time.Time `json:"last_run,omitempty"`
	NextRun         *time.Time `json:"next_run,omitempty"`
	LastError       string     `json:"last_error,omitempty"`
	LogFile         string     `json:"log_file,omitempty"`
	ConfigPath      string     `json:"config_path,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type JobStore interface {
	List(ctx context.Context) ([]Job, error)
	Get(ctx context.Context, id string) (*Job, error)
	Create(ctx context.Context, job *Job) error
	Update(ctx context.Context, job *Job) error
	Delete(ctx context.Context, id string) error
}
