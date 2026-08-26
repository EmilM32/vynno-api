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
