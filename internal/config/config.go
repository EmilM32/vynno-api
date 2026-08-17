package config

import (
	"fmt"
	"os"
	"strings"
)

const defaultAddr = ":8080"

// Config is process configuration loaded from the environment.
type Config struct {
	Addr              string
	DatabaseURL       string
	BootstrapUsername string
	BootstrapPassword string
	SPAOrigins        []string
	CookieSecure      bool
}

// Load reads process env. A local `.env` in the working directory is loaded first
// for keys that are not already set. DATABASE_URL, BOOTSTRAP_PASSWORD, and SPA_ORIGIN are required.
func Load() (Config, error) {
	if err := loadDotEnv(".env"); err != nil {
		return Config{}, err
	}

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = defaultAddr
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	username := strings.TrimSpace(os.Getenv("BOOTSTRAP_USERNAME"))
	if username == "" {
		username = "alexdev"
	}

	password := os.Getenv("BOOTSTRAP_PASSWORD")
	if password == "" {
		return Config{}, fmt.Errorf("BOOTSTRAP_PASSWORD is required")
	}

	origins := parseOrigins(os.Getenv("SPA_ORIGIN"))
	if len(origins) == 0 {
		return Config{}, fmt.Errorf("SPA_ORIGIN is required")
	}

	return Config{
		Addr:              addr,
		DatabaseURL:       databaseURL,
		BootstrapUsername: username,
		BootstrapPassword: password,
		SPAOrigins:        origins,
		CookieSecure:      boolEnv("COOKIE_SECURE"),
	}, nil
}

func parseOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, strings.TrimRight(p, "/"))
		}
	}
	return out
}

func boolEnv(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}
