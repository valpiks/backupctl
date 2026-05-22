package scheduler

import (
	"context"
	"time"
)

type Job struct {
	ID           string     `json:"id"`
	Cron         string     `json:"cron"`
	Interval     string     `json:"interval,omitempty"`
	DatabaseName string     `json:"database_name"`
	BackupType   string     `json:"backup_type"`
	Format       string     `json:"format"`
	Status       string     `json:"status"`
	LastRun      *time.Time `json:"last_run,omitempty"`
	NextRun      *time.Time `json:"next_run,omitempty"`
	LogFile      string     `json:"log_file,omitempty"`
}

type JobStore interface {
	List(ctx context.Context) ([]Job, error)
	Get(ctx context.Context, id string) (*Job, error)
	Create(ctx context.Context, job *Job) error
	Update(ctx context.Context, job *Job) error
	Delete(ctx context.Context, id string) error
}
