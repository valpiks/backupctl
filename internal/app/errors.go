package app

import (
	"errors"
	"fmt"
	"io"
)

type HintError struct {
	Err  error
	Hint string
}

func (e HintError) Error() string {
	return e.Err.Error()
}

func (e HintError) Unwrap() error {
	return e.Err
}

func WithHint(err error, hint string) error {
	if err == nil {
		return nil
	}

	return HintError{Err: err, Hint: hint}
}

func PrintError(out io.Writer, err error) {
	if err == nil {
		return
	}

	fmt.Fprintln(out, err)

	var hintErr HintError
	if errors.As(err, &hintErr) && hintErr.Hint != "" {
		fmt.Fprintf(out, "hint: %s\n", hintErr.Hint)
	}
}
