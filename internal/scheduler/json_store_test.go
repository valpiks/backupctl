package scheduler

import (
	"context"
	"path/filepath"
	"testing"
)

func TestJSONStoreCRUD(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewJSONStore(filepath.Join(t.TempDir(), "jobs.json"))

	job := &Job{
		ID:           "job_1",
		Interval:     "24h",
		DatabaseName: "app",
		BackupType:   "full",
		Format:       "plain",
		Status:       "scheduled",
	}

	if err := store.Create(ctx, job); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := store.Create(ctx, job); err == nil {
		t.Fatal("Create() duplicate error = nil")
	}

	jobs, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(jobs) != 1 {
		t.Fatalf("List() len = %d, want 1", len(jobs))
	}

	got, err := store.Get(ctx, "job_1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.DatabaseName != "app" {
		t.Fatalf("Get().DatabaseName = %q, want app", got.DatabaseName)
	}

	got.Status = "success"
	if err := store.Update(ctx, got); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	updated, err := store.Get(ctx, "job_1")
	if err != nil {
		t.Fatalf("Get() after update error = %v", err)
	}

	if updated.Status != "success" {
		t.Fatalf("updated Status = %q, want success", updated.Status)
	}

	if err := store.Delete(ctx, "job_1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	jobs, err = store.List(ctx)
	if err != nil {
		t.Fatalf("List() after delete error = %v", err)
	}

	if len(jobs) != 0 {
		t.Fatalf("List() after delete len = %d, want 0", len(jobs))
	}
}

func TestJSONStoreMissingJob(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewJSONStore(filepath.Join(t.TempDir(), "jobs.json"))

	if _, err := store.Get(ctx, "missing"); err == nil {
		t.Fatal("Get() missing error = nil")
	}

	if err := store.Update(ctx, &Job{ID: "missing"}); err == nil {
		t.Fatal("Update() missing error = nil")
	}

	if err := store.Delete(ctx, "missing"); err == nil {
		t.Fatal("Delete() missing error = nil")
	}
}
