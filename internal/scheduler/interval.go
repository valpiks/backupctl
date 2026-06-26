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

		if job.Disabled {
			continue
		}

		duration, err := time.ParseDuration(job.Interval)
		if err != nil {
			return fmt.Errorf("parse interval for job %q: %w", job.ID, err)
		}

		if shouldRunMissed(job, time.Now().UTC()) {
			jobCopy := job
			go func() {
				if err := s.service.RunJob(ctx, &jobCopy); err != nil {
					s.service.logger.Error("missed interval job failed", "id", job.ID, "error", err)
				}
			}()
		}

		s.startJob(ctx, job, duration)
	}

	return nil
}

func shouldRunMissed(job Job, now time.Time) bool {
	if job.NextRun == nil || job.NextRun.After(now) {
		return false
	}

	return job.MissedRunPolicy == "" || job.MissedRunPolicy == "run_once"
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
