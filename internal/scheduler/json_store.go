package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type JSONStore struct {
	path string
	mu   sync.Mutex
}

func NewJSONStore(path string) *JSONStore {
	return &JSONStore{
		path: path,
	}
}

func (s *JSONStore) load(ctx context.Context) ([]Job, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Job{}, nil
		}

		return nil, fmt.Errorf("read jobs file: %w", err)
	}

	if len(data) == 0 {
		return []Job{}, nil
	}

	var jobs []Job
	if err := json.Unmarshal(data, &jobs); err != nil {
		return nil, fmt.Errorf("parse jobs file: %w", err)
	}

	return jobs, nil
}

func (s *JSONStore) save(ctx context.Context, jobs []Job) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return fmt.Errorf("create jobs directory: %w", err)
	}

	data, err := json.MarshalIndent(jobs, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal jobs: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0644); err != nil {
		return fmt.Errorf("write jobs file: %w", err)
	}

	return nil
}

func (s *JSONStore) List(ctx context.Context) ([]Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.load(ctx)
}

func (s *JSONStore) Get(ctx context.Context, id string) (*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobs, err := s.load(ctx)
	if err != nil {
		return nil, err
	}

	for i := range jobs {
		if jobs[i].ID == id {
			return &jobs[i], nil
		}
	}

	return nil, fmt.Errorf("job %q not found", id)
}

func (s *JSONStore) Create(ctx context.Context, job *Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobs, err := s.load(ctx)
	if err != nil {
		return err
	}

	for _, existing := range jobs {
		if existing.ID == job.ID {
			return fmt.Errorf("job %q already exists", job.ID)
		}
	}

	jobs = append(jobs, *job)

	return s.save(ctx, jobs)
}

func (s *JSONStore) Update(ctx context.Context, job *Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobs, err := s.load(ctx)
	if err != nil {
		return err
	}

	for i := range jobs {
		if jobs[i].ID == job.ID {
			jobs[i] = *job
			return s.save(ctx, jobs)
		}
	}

	return fmt.Errorf("job %q not found", job.ID)
}

func (s *JSONStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobs, err := s.load(ctx)
	if err != nil {
		return err
	}

	for i := range jobs {
		if jobs[i].ID == id {
			jobs = append(jobs[:i], jobs[i+1:]...)
			return s.save(ctx, jobs)
		}
	}

	return fmt.Errorf("job %q not found", id)
}
