package envfile

import (
	"fmt"
	"os"
	"strings"
)

func Load(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read env file: %w", err)
	}

	values := make(map[string]string)
	lines := strings.Split(string(data), "\n")

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("line %d: expected KEY=value", i+1)
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		if !validKey(key) {
			return nil, fmt.Errorf("line %d: invalid key %q", i+1, key)
		}

		value = strings.Trim(value, `"'`)
		values[key] = value
	}

	return values, nil
}

func Apply(values map[string]string) error {
	for key, value := range values {
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set %s: %w", key, err)
		}
	}

	return nil
}

func LoadAndApply(path string) error {
	values, err := Load(path)
	if err != nil {
		return err
	}

	return Apply(values)
}

func validKey(key string) bool {
	if key == "" {
		return false
	}

	for _, r := range key {
		if r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			continue
		}
		return false
	}

	return true
}
