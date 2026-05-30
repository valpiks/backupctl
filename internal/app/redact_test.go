package app

import (
	"errors"
	"strings"
	"testing"
)

func TestRedactError(t *testing.T) {
	err := errors.New("backup failed with password=supersecret")

	got := redactError(err, []string{"supersecret"})
	if got == nil {
		t.Fatal("redactError() = nil")
	}

	if strings.Contains(got.Error(), "supersecret") {
		t.Fatalf("redactError() leaked secret: %q", got.Error())
	}

	if got.Error() != "backup failed with password=****" {
		t.Fatalf("redactError() = %q", got.Error())
	}
}

func TestRedactErrorNil(t *testing.T) {
	if got := redactError(nil, []string{"secret"}); got != nil {
		t.Fatalf("redactError(nil) = %v, want nil", got)
	}
}
