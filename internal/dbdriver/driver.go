package database

import (
	"context"
	"io"
)

type BackupOptions struct {
	Type string
}

type RestoreOptions struct {
	TargetDatabase string
}

type Driver interface {
	Ping(ctx context.Context) error
	Backup(ctx context.Context, opts BackupOptions) (io.ReadCloser, error)
	Restore(ctx context.Context, input io.Reader, opts RestoreOptions) error
}
