package utils

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// LoadDotEnvFiles loads .env from cwd and grapery/.env (when started from repo root).
// Empty values in the file are skipped so placeholders like KEY= do not wipe shell exports.
// Existing non-empty process env vars are never overridden.
func LoadDotEnvFiles() {
	for _, path := range []string{".env", "grapery/.env"} {
		if err := loadDotEnvSkipEmpty(path); err != nil {
			continue
		}
	}
}

func loadDotEnvSkipEmpty(path string) error {
	m, err := godotenv.Read(path)
	if err != nil {
		return err
	}
	for k, v := range m {
		if strings.TrimSpace(v) == "" {
			continue
		}
		if strings.TrimSpace(os.Getenv(k)) != "" {
			continue
		}
		if err := os.Setenv(k, v); err != nil {
			return fmt.Errorf("setenv %s: %w", k, err)
		}
	}
	return nil
}
