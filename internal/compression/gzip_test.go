package compression

import (
	"io"
	"strings"
	"testing"
)

func TestGzipCompressor_RoundTrip(t *testing.T) {
	c := NewGzipCompressor()
	input := "Hello backupctl"

	compressed := c.Compress(strings.NewReader(input))

	reader, err := c.Decompress((compressed))
	if err != nil {
		t.Fatalf("decompress: %v", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read decompressed: %v", err)
	}

	if string(data) != input {
		t.Fatalf("got %q want %q", string(data), input)
	}
}

func TestGzipCompressor_Decimpress_InvalidData(t *testing.T) {
	c := NewGzipCompressor()

	_, err := c.Decompress(strings.NewReader("not-gzip"))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
