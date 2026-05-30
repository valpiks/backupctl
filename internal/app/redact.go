package app

import (
	"errors"

	"github.com/valpiks/backupctl/internal/secrets"
)

func redactError(err error, knownSecrets []string) error {
	if err == nil {
		return nil
	}

	return errors.New(secrets.Redact(err.Error(), knownSecrets))
}
