package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/valpiks/backupctl/internal/backup"
)

type Service struct {
	cron               *cron.Cron
	jobs               JobStore
	service            *backup.Service
	logger             *slog.Logger
	encryptionEnabled  bool
	encryptionPassword string
	backupctlVersion   string

	mu      sync.Mutex
	running map[string]struct{}
}

func NewService(jobs JobStore, service *backup.Service, logger *slog.Logger) *Service {
	return &Service{
		cron:    cron.New(),
		jobs:    jobs,
		service: service,
		logger:  logger,
		running: make(map[string]struct{}),
	}
}

func (s *Service) SetEncryption(enabled bool, password string) {
	s.encryptionEnabled = enabled
	s.encryptionPassword = password
}

func (s *Service) SetBackupctlVersion(version string) {
	s.backupctlVersion = version
}

func (s *Service) CreateJob(ctx context.Context, job *Job) error {
	now := time.Now().UTC()

	if job.Status == "" {
		job.Status = "scheduled"
	}

	if job.Format == "" {
		job.Format = "plain"
	}

	if job.CreatedAt.IsZero() {
		job.CreatedAt = now
	}
	job.UpdatedAt = now

	if err := s.jobs.Create(ctx, job); err != nil {
		s.logger.Error("create scheduled job failed", "error", err)
		return err
	}

	s.logger.Info("scheduled job created", "id", job.ID, "database", job.DatabaseName, "cron", job.Cron, "interval", job.Interval)

	return nil
}

func (s *Service) RunJob(ctx context.Context, job *Job) error {
	if job.Disabled {
		s.logger.Info("scheduled backup skipped because job is disabled", "id", job.ID)
		return nil
	}

	if !s.tryLockJob(job.ID) {
		s.logger.Warn("scheduled backup already running", "id", job.ID)
		return fmt.Errorf("job already running: %s", job.ID)
	}
	defer s.unlockJob(job.ID)

	lock, err := TryFileLock(jobLockPath(job.ID))
	if err != nil {
		s.logger.Warn("scheduled backup lock already held", "id", job.ID, "error", err)
		return err
	}
	defer lock.Unlock()

	startedAt := time.Now().UTC()

	job.Status = "running"
	job.LastRun = &startedAt
	job.LastError = ""
	job.UpdatedAt = startedAt

	if err := s.jobs.Update(ctx, job); err != nil {
		return err
	}

	s.logger.Info("scheduled backup started", "id", job.ID, "database", job.DatabaseName)

	result, err := s.service.Run(ctx, backup.Options{
		DatabaseName:       job.DatabaseName,
		BackupType:         job.BackupType,
		Format:             job.Format,
		EncryptionEnabled:  s.encryptionEnabled,
		EncryptionPassword: s.encryptionPassword,
		BackupctlVersion:   s.backupctlVersion,
	})

	if err != nil {
		now := time.Now().UTC()
		job.Status = "failed"
		job.LastError = err.Error()
		job.UpdatedAt = now
		_ = s.jobs.Update(ctx, job)

		s.logger.Error("scheduled backup failed", "id", job.ID, "error", err)

		return err
	}

	now := time.Now().UTC()
	job.Status = "success"
	job.LastError = ""
	job.UpdatedAt = now

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

		if job.Disabled {
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

func (s *Service) tryLockJob(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.running[id]; ok {
		return false
	}

	s.running[id] = struct{}{}
	return true
}

func (s *Service) unlockJob(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.running, id)
}
