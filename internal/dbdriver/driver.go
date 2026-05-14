package database

import (
	"context"
	"io"
)

type BackupOptions struct {
	Type      string
	ShemaOnly bool
	DataOnly  bool
	Tables    []string
	Format    string
}

type RestoreOptions struct {
	TargetDatabase string
	Format         string
}

type Driver interface {
	Ping(ctx context.Context) error
	Backup(ctx context.Context, opts BackupOptions) (io.ReadCloser, error)
	Restore(ctx context.Context, input io.Reader, opts RestoreOptions) error
}
