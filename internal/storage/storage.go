package storage

import (
	"context"
	"io"
)

type BackupFile struct {
	Name string
	Size int64
}

type Storage interface {
	Save(ctx context.Context, name string, data io.Reader) error
	Open(ctx context.Context, name string) (io.ReadCloser, error)
	List(ctx context.Context) ([]BackupFile, error)
	Delete(ctx context.Context, name string) error
}
