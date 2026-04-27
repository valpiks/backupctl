package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/valpiks/backupctl/internal/storage"
)

type Storage struct {
	path string
}

func NewStorage(path string) (*Storage, error) {
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("create backup directory: %w", err)
	}

	return &Storage{path}, nil
}

func (s *Storage) Save(ctx context.Context, name string, data io.Reader) error {
	filepath := filepath.Join(s.path, name)

	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("create backup  file %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, data)
	if err != nil {
		return fmt.Errorf("write backup file %w", err)
	}

	return nil
}

func (s *Storage) Open(ctx context.Context, name string) (io.ReadCloser, error) {
	filepath := filepath.Join(s.path, name)

	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("open backup file %w", err)
	}

	return file, nil
}

func (s *Storage) List(ctx context.Context) ([]storage.BackupFile, error) {
	entries, err := os.ReadDir(s.path)
	if err != nil {
		return nil, fmt.Errorf("read backup directory %w", err)
	}

	files := make([]storage.BackupFile, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("read file info %w", err)
		}

		files = append(files, storage.BackupFile{
			Name: entry.Name(),
			Size: info.Size(),
		})
	}

	return files, nil
}

func (s *Storage) Delete(name string) error {
	filepath := filepath.Join(s.path, name)
	return os.Remove(filepath)
}
