package config

import (
	"fmt"
	"os"
)

const defaultAddr = ":8080"

// Config is process configuration loaded from the environment.
type Config struct {
	Addr        string
	DatabaseURL string
}

// Load reads ADDR and DATABASE_URL. ADDR defaults to :8080.
// DATABASE_URL is required so a missing store fails at boot, not later.
func Load() (Config, error) {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = defaultAddr
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	return Config{
		Addr:        addr,
		DatabaseURL: databaseURL,
	}, nil
}
