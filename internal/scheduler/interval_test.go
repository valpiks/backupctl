package scheduler

import (
	"context"
	"strings"
	"testing"
)

func TestIntervalSchedulerStart(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		jobs    []Job
		wantErr string
	}{
		{
			name: "no interval jobs",
			jobs: []Job{
				{ID: "job_1", Cron: "0 3 * * *"},
			},
		},
		{
			name: "invalid interval",
			jobs: []Job{
				{ID: "job_1", Interval: "bad"},
			},
			wantErr: "parse interval",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scheduler := NewIntervalScheduler(nil, &memoryJobStore{jobs: tt.jobs})
			err := scheduler.Start(context.Background())

			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("Start() error = nil")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Start() error = %v, want to contain %q", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("Start() error = %v", err)
			}
		})
	}
}

func TestIntervalSchedulerStartListError(t *testing.T) {
	t.Parallel()

	scheduler := NewIntervalScheduler(nil, &memoryJobStore{err: context.Canceled})

	if err := scheduler.Start(context.Background()); err == nil {
		t.Fatal("Start() error = nil")
	}
}
