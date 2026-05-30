package secrets

import (
	"net/url"
	"strings"
)

const Mask = "****"

func Redact(input string, knownSecrets []string) string {
	output := input

	output = RedactKnown(output, knownSecrets)
	output = RedactURLPasswords(output)

	return output
}

func RedactKnown(input string, knownSecrets []string) string {
	output := input

	for _, secret := range knownSecrets {
		if secret == "" {
			continue
		}

		output = strings.ReplaceAll(output, secret, Mask)
	}

	return output
}

func RedactURLPasswords(input string) string {
	fields := strings.Fields(input)
	if len(fields) == 0 {
		return input
	}

	output := input

	for _, field := range fields {
		redacted := redactURLPasswordsInField(field)
		if redacted != field {
			output = strings.ReplaceAll(output, field, redacted)
		}
	}

	return output
}

func redactURLPasswordsInField(value string) string {
	for _, scheme := range []string{"postgres://", "postgresql://", "mongodb://", "mongodb+srv://"} {
		index := strings.Index(value, scheme)
		if index == -1 {
			continue
		}

		prefix := value[:index]
		urlValue := value[index:]
		redacted := redactURLPassword(urlValue)

		return prefix + redacted
	}

	return value
}

func redactURLPassword(value string) string {
	parsed, err := url.Parse(value)
	if err != nil {
		return value
	}

	if !isSensitiveURLScheme(parsed.Scheme) {
		return value
	}

	if parsed.User == nil {
		return value
	}

	password, hasPassword := parsed.User.Password()
	if !hasPassword || password == "" {
		return value
	}

	username := parsed.User.Username()
	parsed.User = url.UserPassword(username, Mask)

	return strings.ReplaceAll(parsed.String(), "%2A%2A%2A%2A", Mask)
}

func isSensitiveURLScheme(scheme string) bool {
	switch scheme {
	case "postgres", "postgresql", "mongodb", "mongodb+srv":
		return true
	default:
		return false
	}
}
