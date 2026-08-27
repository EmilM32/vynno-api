package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	defaultAddr     = ":8080"
	defaultSMTPPort = 1025
)

// Config is process configuration loaded from the environment.
type Config struct {
	Addr              string
	DatabaseURL       string
	BootstrapEmail    string
	BootstrapPassword string
	SPAOrigins        []string
	CookieSecure      bool
	PublicAPIOrigin   string
	LogFormat         string
	Mail              Mail
}

// Mail is outbound mail settings (ADR-0015).
type Mail struct {
	Mode     string
	Host     string
	Port     int
	Username string
	Password string
	StartTLS bool
	From     string
	FromName string
}

// Load reads process env. A local `.env` in the working directory is loaded first
// for keys that are not already set. DATABASE_URL, SPA_ORIGIN, and
// PUBLIC_API_ORIGIN are required. BOOTSTRAP_PASSWORD is optional here (playground
// seed/reset require it in cmd/devdata). LOG_FORMAT is "text" or "json" (default text).
// MAIL_MODE is smtp, log, or discard (empty is discard). smtp requires SMTP_HOST
// and MAIL_FROM.
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

	email := bootstrapEmailFromEnv()

	password := os.Getenv("BOOTSTRAP_PASSWORD")

	origins := parseOrigins(os.Getenv("SPA_ORIGIN"))
	if len(origins) == 0 {
		return Config{}, fmt.Errorf("SPA_ORIGIN is required")
	}

	publicOrigin, err := parsePublicOrigin(os.Getenv("PUBLIC_API_ORIGIN"))
	if err != nil {
		return Config{}, err
	}

	mail, err := parseMail()
	if err != nil {
		return Config{}, err
	}

	return Config{
		Addr:              addr,
		DatabaseURL:       databaseURL,
		BootstrapEmail:    email,
		BootstrapPassword: password,
		SPAOrigins:        origins,
		CookieSecure:      boolEnv("COOKIE_SECURE"),
		PublicAPIOrigin:   publicOrigin,
		LogFormat:         parseLogFormat(os.Getenv("LOG_FORMAT")),
		Mail:              mail,
	}, nil
}

func parseMail() (Mail, error) {
	mode, err := parseMailMode(os.Getenv("MAIL_MODE"))
	if err != nil {
		return Mail{}, err
	}
	port, err := parseSMTPPort(os.Getenv("SMTP_PORT"))
	if err != nil {
		return Mail{}, err
	}
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	from := strings.TrimSpace(os.Getenv("MAIL_FROM"))
	if mode == "smtp" {
		if host == "" {
			return Mail{}, fmt.Errorf("SMTP_HOST is required when MAIL_MODE=smtp")
		}
		if from == "" {
			return Mail{}, fmt.Errorf("MAIL_FROM is required when MAIL_MODE=smtp")
		}
	}
	return Mail{
		Mode:     mode,
		Host:     host,
		Port:     port,
		Username: os.Getenv("SMTP_USERNAME"),
		Password: os.Getenv("SMTP_PASSWORD"),
		StartTLS: boolEnv("SMTP_STARTTLS"),
		From:     from,
		FromName: strings.TrimSpace(os.Getenv("MAIL_FROM_NAME")),
	}, nil
}

func parseMailMode(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "discard":
		return "discard", nil
	case "smtp":
		return "smtp", nil
	case "log":
		return "log", nil
	default:
		return "", fmt.Errorf("MAIL_MODE must be smtp, log, or discard")
	}
}

func parseSMTPPort(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultSMTPPort, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > 65535 {
		return 0, fmt.Errorf("SMTP_PORT must be an integer 1–65535")
	}
	return n, nil
}

func bootstrapEmailFromEnv() string {
	email := strings.TrimSpace(os.Getenv("BOOTSTRAP_EMAIL"))
	if email == "" {
		email = strings.TrimSpace(os.Getenv("BOOTSTRAP_USERNAME"))
	}
	if email == "" {
		return "alexdev@vynno.local"
	}
	if !strings.Contains(email, "@") {
		return strings.ToLower(email) + "@vynno.local"
	}
	return strings.ToLower(email)
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
