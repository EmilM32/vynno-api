package config

import (
	"fmt"
	"os"
)

// DatabaseURL loads .env for unset keys and returns DATABASE_URL.
// It does not require SPA or mail settings. Processes that only talk to Postgres use this.
func DatabaseURL() (string, error) {
	if err := loadDotEnv(".env"); err != nil {
		return "", err
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return "", fmt.Errorf("DATABASE_URL is required")
	}
	return databaseURL, nil
}
