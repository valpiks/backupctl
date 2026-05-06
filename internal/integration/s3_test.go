package integration

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/valpiks/backupctl/internal/storage/s3"
)

func TestStorage_SaveOpenList(t *testing.T) {
	requireIntegrationTest(t)

	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := s3.Config{
		Bucket:         "backupctl-test",
		Region:         "us-east-1",
		Prefix:         "test/",
		Endpoint:       "http://localhost:9000",
		ForcePathStyle: true,
	}

	s, err := s3.NewStorage(cfg)
	if err != nil {
		t.Fatalf("new storage: %v", err)
	}

	ctx := context.Background()

	err = s.Save(ctx, "a.txt", strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	t.Cleanup(func() {
		s.Delete(context.Background(), "a.txt")
	})

	r, err := s.Open(ctx, "a.txt")
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

	files, err := s.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	found := false
	for _, f := range files {
		if f.Name == "a.txt" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("a.txt not found in list, got %d files", len(files))
	}
}
