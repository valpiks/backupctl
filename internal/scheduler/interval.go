package scheduler

import (
	"context"
	"fmt"
	"time"
)

type IntervalScheduler struct {
	service *Service
	jobs    JobStore
}

func NewIntervalScheduler(service *Service, jobs JobStore) *IntervalScheduler {
	return &IntervalScheduler{
		service: service,
		jobs:    jobs,
	}
}

func (s *IntervalScheduler) Start(ctx context.Context) error {
	jobs, err := s.jobs.List(ctx)
	if err != nil {
		return err
	}

	for i := range jobs {
		var job Job = jobs[i]

		if job.Interval == "" {
			continue
		}

		duration, err := time.ParseDuration(job.Interval)
		if err != nil {
			return fmt.Errorf("parse interval for job %q: %w", job.ID, err)
		}

		s.startJob(ctx, job, duration)
	}

	return nil
}

func (s *IntervalScheduler) startJob(ctx context.Context, job Job, duration time.Duration) {
	ticker := time.NewTicker(duration)

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker.C:
				jobCopy := job

				if err := s.service.RunJob(ctx, &jobCopy); err != nil {
					s.service.logger.Error("interval job failed", "id", job.ID, "error", err)
				}
			}
		}
	}()
}
