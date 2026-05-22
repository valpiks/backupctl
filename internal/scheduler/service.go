package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/valpiks/backupctl/internal/backup"
)

type Service struct {
	cron    *cron.Cron
	jobs    JobStore
	service *backup.Service
	logger  *slog.Logger
}

func NewService(jobs JobStore, service *backup.Service, logger *slog.Logger) *Service {
	return &Service{
		cron:    cron.New(),
		jobs:    jobs,
		service: service,
		logger:  logger,
	}
}

func (s *Service) CreateJob(ctx context.Context, job *Job) error {
	if job.Status == "" {
		job.Status = "scheduled"
	}

	if job.Format == "" {
		job.Format = "plain"
	}

	if err := s.jobs.Create(ctx, job); err != nil {
		s.logger.Error("create scheduled job failed", "error", err)
		return err
	}

	s.logger.Info("scheduled job created", "id", job.ID, "database", job.DatabaseName, "cron", job.Cron, "interval", job.Interval)

	return nil
}

func (s *Service) RunJob(ctx context.Context, job *Job) error {
	startedAt := time.Now().UTC()

	job.Status = "running"
	job.LastRun = &startedAt

	if err := s.jobs.Update(ctx, job); err != nil {
		return err
	}

	s.logger.Info("scheduled backup started", "id", job.ID, "database", job.DatabaseName)

	result, err := s.service.Run(ctx, backup.Options{DatabaseName: job.DatabaseName, BackupType: job.BackupType, Format: job.Format})

	if err != nil {
		job.Status = "failed"
		_ = s.jobs.Update(ctx, job)

		s.logger.Error("scheduled backup failed", "id", job.ID, "error", err)

		return err
	}

	job.Status = "success"

	if err := s.jobs.Update(ctx, job); err != nil {
		return err
	}

	s.logger.Info("scheduled backup finished", "id", job.ID, "file", result.FileName, "duration", result.EndedAt.Sub(result.StartedAt).String())

	return nil
}

func (s *Service) Start() {
	s.cron.Start()
	s.logger.Info("scheduler started")
}

func (s *Service) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	s.logger.Info("scheduler stopped")
}

func (s *Service) RegisterCronJobs(ctx context.Context) error {
	jobs, err := s.jobs.List(ctx)
	if err != nil {
		return err
	}

	for i := range jobs {
		job := jobs[i]

		if job.Cron == "" {
			continue
		}

		_, err := s.cron.AddFunc(job.Cron, func() {
			jobCopy := job
			if err := s.RunJob(ctx, &jobCopy); err != nil {
				s.logger.Error("cron job failed", "id", job.ID, "error", err)
			}
		})

		if err != nil {
			return fmt.Errorf("register cron job %q: %w", job.ID, err)
		}
	}

	return nil
}
