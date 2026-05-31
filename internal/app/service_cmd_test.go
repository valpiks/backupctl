package app

import (
	"os"
	"testing"
)

func TestDefaultServiceBinaryPathUsesCurrentExecutable(t *testing.T) {
	got := defaultServiceBinaryPath()

	want, err := os.Executable()
	if err != nil {
		t.Fatalf("Executable() error = %v", err)
	}

	if got != want {
		t.Fatalf("defaultServiceBinaryPath() = %q, want %q", got, want)
	}
}
