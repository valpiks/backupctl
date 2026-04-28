package local

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestStorage_SaveOpenList(t *testing.T) {
	dir := t.TempDir()

	s, err := NewStorage(dir)
	if err != nil {
		t.Fatalf("new storage: %v", err)
	}

	err = s.Save(context.Background(), "a.txt", strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	r, err := s.Open(context.Background(), "a.txt")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if string(data) != "hello" {
		t.Fatalf("got %q", string(data))
	}

	files, err := s.List(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
}
