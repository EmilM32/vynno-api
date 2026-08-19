package config

import (
	"fmt"
	"net/url"
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
	PublicAPIOrigin   string
	LogFormat         string
}

// Load reads process env. A local `.env` in the working directory is loaded first
// for keys that are not already set. DATABASE_URL, SPA_ORIGIN, and
// PUBLIC_API_ORIGIN are required. BOOTSTRAP_PASSWORD is optional here (playground
// seed/reset require it in cmd/devdata). LOG_FORMAT is "text" or "json" (default text).
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

	origins := parseOrigins(os.Getenv("SPA_ORIGIN"))
	if len(origins) == 0 {
		return Config{}, fmt.Errorf("SPA_ORIGIN is required")
	}

	publicOrigin, err := parsePublicOrigin(os.Getenv("PUBLIC_API_ORIGIN"))
	if err != nil {
		return Config{}, err
	}

	return Config{
		Addr:              addr,
		DatabaseURL:       databaseURL,
		BootstrapUsername: username,
		BootstrapPassword: password,
		SPAOrigins:        origins,
		CookieSecure:      boolEnv("COOKIE_SECURE"),
		PublicAPIOrigin:   publicOrigin,
		LogFormat:         parseLogFormat(os.Getenv("LOG_FORMAT")),
	}, nil
}

func parseLogFormat(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "json":
		return "json"
	default:
		return "text"
	}
}

func parsePublicOrigin(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("PUBLIC_API_ORIGIN is required")
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", fmt.Errorf("PUBLIC_API_ORIGIN must be an absolute http(s) origin")
	}
	return u.Scheme + "://" + u.Host, nil
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
