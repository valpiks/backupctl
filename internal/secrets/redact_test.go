package secrets

import (
	"strings"
	"testing"
)

func TestRedactKnown(t *testing.T) {
	got := RedactKnown("password=supersecret token=abc", []string{"supersecret", "abc"})
	want := "password=**** token=****"

	if got != want {
		t.Fatalf("RedactKnown() = %q, want %q", got, want)
	}
}

func TestRedactKnownIgnoresEmptySecrets(t *testing.T) {
	got := RedactKnown("hello world", []string{"", "world"})
	want := "hello ****"

	if got != want {
		t.Fatalf("RedactKnown() = %q, want %q", got, want)
	}
}

func TestRedactPostgresURLPassword(t *testing.T) {
	input := "connect failed postgres://user:secret@localhost:5432/app"
	got := RedactURLPasswords(input)
	want := "connect failed postgres://user:****@localhost:5432/app"

	if got != want {
		t.Fatalf("RedactURLPasswords() = %q, want %q", got, want)
	}
}

func TestRedactMongoURLPassword(t *testing.T) {
	input := "connect failed mongodb://user:secret@localhost:27017/app"
	got := RedactURLPasswords(input)
	want := "connect failed mongodb://user:****@localhost:27017/app"

	if got != want {
		t.Fatalf("RedactURLPasswords() = %q, want %q", got, want)
	}
}

func TestRedactMongoSrvURLPassword(t *testing.T) {
	input := "connect failed mongodb+srv://user:secret@example.mongodb.net/app"
	got := RedactURLPasswords(input)
	want := "connect failed mongodb+srv://user:****@example.mongodb.net/app"

	if got != want {
		t.Fatalf("RedactURLPasswords() = %q, want %q", got, want)
	}
}

func TestRedactURLPasswordsIgnoresNonSensitiveURLs(t *testing.T) {
	input := "open https://user:secret@example.com/path"
	got := RedactURLPasswords(input)

	if got != input {
		t.Fatalf("RedactURLPasswords() = %q, want %q", got, input)
	}
}

func TestRedactCombinesKnownSecretsAndURLs(t *testing.T) {
	input := "password=supersecret uri=mongodb://user:mongopass@localhost:27017/app"
	got := Redact(input, []string{"supersecret"})

	if strings.Contains(got, "supersecret") {
		t.Fatalf("Redact() leaked known secret: %q", got)
	}

	if strings.Contains(got, "mongopass") {
		t.Fatalf("Redact() leaked URL password: %q", got)
	}

	want := "password=**** uri=mongodb://user:****@localhost:27017/app"
	if got != want {
		t.Fatalf("Redact() = %q, want %q", got, want)
	}
}
