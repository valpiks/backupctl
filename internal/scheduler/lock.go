package scheduler

import (
	"fmt"
	"os"
	"path/filepath"
)

type FileLock struct {
	path string
	file *os.File
}

func TryFileLock(path string) (*FileLock, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return nil, fmt.Errorf("lock already held: %s", path)
		}
		return nil, err
	}

	_, _ = fmt.Fprintf(file, "%d\n", os.Getpid())
	return &FileLock{path: path, file: file}, nil
}

func (l *FileLock) Unlock() error {
	if l.file != nil {
		_ = l.file.Close()
	}
	return os.Remove(l.path)
}

func jobLockPath(jobID string) string {
	if jobID == "" {
		jobID = "unknown"
	}
	return filepath.Join(".backupctl", "locks", jobID+".lock")
}
