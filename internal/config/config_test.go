package config

import (
	"os"
	"testing"
)

func requiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://vynno:vynno@localhost:5432/vynno?sslmode=disable")
	t.Setenv("BOOTSTRAP_PASSWORD", "local-only-password")
	t.Setenv("SPA_ORIGIN", "http://localhost:5173")
	t.Setenv("PUBLIC_API_ORIGIN", "http://localhost:8080")
	t.Setenv("MAIL_MODE", "")
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_PORT", "")
	t.Setenv("SMTP_USERNAME", "")
	t.Setenv("SMTP_PASSWORD", "")
	t.Setenv("SMTP_STARTTLS", "")
	t.Setenv("MAIL_FROM", "")
	t.Setenv("MAIL_FROM_NAME", "")
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	requiredEnv(t)
	if err := os.Unsetenv("DATABASE_URL"); err != nil {
		t.Fatal(err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected an error when DATABASE_URL is missing")
	}
}

func TestLoadAllowsEmptyBootstrapPassword(t *testing.T) {
	requiredEnv(t)
	if err := os.Unsetenv("BOOTSTRAP_PASSWORD"); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BootstrapPassword != "" {
		t.Fatalf("BootstrapPassword = %q, want empty", cfg.BootstrapPassword)
	}
}

func TestLoadRequiresSPAOrigin(t *testing.T) {
	requiredEnv(t)
	if err := os.Unsetenv("SPA_ORIGIN"); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Fatal("expected an error when SPA_ORIGIN is missing")
	}
}

func TestLoadDefaultsAddr(t *testing.T) {
	requiredEnv(t)
	if err := os.Unsetenv("ADDR"); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != defaultAddr {
		t.Fatalf("Addr = %q, want %q", cfg.Addr, defaultAddr)
	}
	if cfg.BootstrapEmail != "alexdev@vynno.local" {
		t.Fatalf("BootstrapEmail = %q", cfg.BootstrapEmail)
	}
	if len(cfg.SPAOrigins) != 1 || cfg.SPAOrigins[0] != "http://localhost:5173" {
		t.Fatalf("SPAOrigins = %#v", cfg.SPAOrigins)
	}
	if cfg.CookieSecure {
		t.Fatal("CookieSecure should default false")
	}
	if cfg.PublicAPIOrigin != "http://localhost:8080" {
		t.Fatalf("PublicAPIOrigin = %q", cfg.PublicAPIOrigin)
	}
	if cfg.LogFormat != "text" {
		t.Fatalf("LogFormat = %q, want text", cfg.LogFormat)
	}
	if cfg.Mail.Mode != "discard" {
		t.Fatalf("Mail.Mode = %q, want discard", cfg.Mail.Mode)
	}
	if cfg.Mail.Port != defaultSMTPPort {
		t.Fatalf("Mail.Port = %d, want %d", cfg.Mail.Port, defaultSMTPPort)
	}
}

func TestLoadLogFormatJSON(t *testing.T) {
	requiredEnv(t)
	t.Setenv("LOG_FORMAT", "json")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LogFormat != "json" {
		t.Fatalf("LogFormat = %q, want json", cfg.LogFormat)
	}
}

func TestLoadRequiresPublicAPIOrigin(t *testing.T) {
	requiredEnv(t)
	if err := os.Unsetenv("PUBLIC_API_ORIGIN"); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Fatal("expected an error when PUBLIC_API_ORIGIN is missing")
	}
}

func TestLoadDotEnvFillsUnsetKeys(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	content := "BOOTSTRAP_PASSWORD=from-file\nSPA_ORIGIN=http://localhost:5173\nDATABASE_URL=postgres://from-file\nPUBLIC_API_ORIGIN=http://localhost:8080\n"
	if err := os.WriteFile(".env", []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DATABASE_URL", "postgres://already-set")
	if err := os.Unsetenv("BOOTSTRAP_PASSWORD"); err != nil {
		t.Fatal(err)
	}
	if err := os.Unsetenv("SPA_ORIGIN"); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabaseURL != "postgres://already-set" {
		t.Fatalf("should not override existing env: %q", cfg.DatabaseURL)
	}
	if cfg.BootstrapPassword != "from-file" {
		t.Fatalf("BootstrapPassword = %q", cfg.BootstrapPassword)
	}
}

func TestLoadMailModeEmptyIsDiscard(t *testing.T) {
	requiredEnv(t)
	if err := os.Unsetenv("MAIL_MODE"); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mail.Mode != "discard" {
		t.Fatalf("Mail.Mode = %q, want discard", cfg.Mail.Mode)
	}
}

func TestLoadMailModeSMTPRequiresHostAndFrom(t *testing.T) {
	requiredEnv(t)
	t.Setenv("MAIL_MODE", "smtp")
	if err := os.Unsetenv("SMTP_HOST"); err != nil {
		t.Fatal(err)
	}
	if err := os.Unsetenv("MAIL_FROM"); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Fatal("expected an error when MAIL_MODE=smtp without SMTP_HOST")
	}

	t.Setenv("SMTP_HOST", "127.0.0.1")
	if _, err := Load(); err == nil {
		t.Fatal("expected an error when MAIL_MODE=smtp without MAIL_FROM")
	}

	t.Setenv("MAIL_FROM", "vynno@localhost")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mail.Mode != "smtp" || cfg.Mail.Host != "127.0.0.1" || cfg.Mail.From != "vynno@localhost" {
		t.Fatalf("Mail = %+v", cfg.Mail)
	}
	if cfg.Mail.Port != defaultSMTPPort {
		t.Fatalf("Mail.Port = %d, want %d", cfg.Mail.Port, defaultSMTPPort)
	}
	if cfg.Mail.StartTLS {
		t.Fatal("StartTLS should default false")
	}
}

func TestLoadMailModeLogDoesNotRequireSMTP(t *testing.T) {
	requiredEnv(t)
	t.Setenv("MAIL_MODE", "log")
	if err := os.Unsetenv("SMTP_HOST"); err != nil {
		t.Fatal(err)
	}
	if err := os.Unsetenv("MAIL_FROM"); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mail.Mode != "log" {
		t.Fatalf("Mail.Mode = %q, want log", cfg.Mail.Mode)
	}
}

func TestLoadMailModeUnknown(t *testing.T) {
	requiredEnv(t)
	t.Setenv("MAIL_MODE", "resend")
	if _, err := Load(); err == nil {
		t.Fatal("expected an error for unknown MAIL_MODE")
	}
}

func TestLoadSMTPPortAndStartTLS(t *testing.T) {
	requiredEnv(t)
	t.Setenv("MAIL_MODE", "smtp")
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_PORT", "587")
	t.Setenv("SMTP_STARTTLS", "true")
	t.Setenv("SMTP_USERNAME", "user")
	t.Setenv("SMTP_PASSWORD", "secret")
	t.Setenv("MAIL_FROM", "vynno@example.com")
	t.Setenv("MAIL_FROM_NAME", "Vynno")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mail.Port != 587 {
		t.Fatalf("Mail.Port = %d, want 587", cfg.Mail.Port)
	}
	if !cfg.Mail.StartTLS {
		t.Fatal("StartTLS = false, want true")
	}
	if cfg.Mail.Username != "user" || cfg.Mail.Password != "secret" {
		t.Fatalf("credentials: %+v", cfg.Mail)
	}
	if cfg.Mail.FromName != "Vynno" {
		t.Fatalf("FromName = %q", cfg.Mail.FromName)
	}
}

func TestLoadSMTPPortInvalid(t *testing.T) {
	requiredEnv(t)
	t.Setenv("SMTP_PORT", "not-a-port")
	if _, err := Load(); err == nil {
		t.Fatal("expected an error for invalid SMTP_PORT")
	}
}

func TestLoadUsesAddr(t *testing.T) {
	requiredEnv(t)
	t.Setenv("ADDR", ":9090")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != ":9090" {
		t.Fatalf("Addr = %q, want :9090", cfg.Addr)
	}
}
