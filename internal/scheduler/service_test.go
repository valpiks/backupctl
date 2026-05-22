package scheduler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
)

func TestServiceRegisterCronJobs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		jobs    []Job
		wantErr string
	}{
		{
			name: "valid cron",
			jobs: []Job{
				{ID: "job_1", Cron: "0 3 * * *"},
			},
		},
		{
			name: "invalid cron",
			jobs: []Job{
				{ID: "job_1", Cron: "not a cron"},
			},
			wantErr: "register cron job",
		},
		{
			name: "interval jobs are skipped",
			jobs: []Job{
				{ID: "job_1", Interval: "24h"},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := NewService(&memoryJobStore{jobs: tt.jobs}, nil, discardLogger())
			err := service.RegisterCronJobs(context.Background())

			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("RegisterCronJobs() error = nil")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("RegisterCronJobs() error = %v, want to contain %q", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("RegisterCronJobs() error = %v", err)
			}
		})
	}
}

func TestServiceRegisterCronJobsListError(t *testing.T) {
	t.Parallel()

	service := NewService(&memoryJobStore{err: errors.New("list failed")}, nil, discardLogger())

	if err := service.RegisterCronJobs(context.Background()); err == nil {
		t.Fatal("RegisterCronJobs() error = nil")
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type memoryJobStore struct {
	jobs []Job
	err  error
}

func (s *memoryJobStore) List(ctx context.Context) ([]Job, error) {
	if s.err != nil {
		return nil, s.err
	}

	return s.jobs, nil
}

func (s *memoryJobStore) Get(ctx context.Context, id string) (*Job, error) {
	return nil, errors.New("not implemented")
}

func (s *memoryJobStore) Create(ctx context.Context, job *Job) error {
	return errors.New("not implemented")
}

func (s *memoryJobStore) Update(ctx context.Context, job *Job) error {
	return errors.New("not implemented")
}

func (s *memoryJobStore) Delete(ctx context.Context, id string) error {
	return errors.New("not implemented")
}
