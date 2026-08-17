package config

import (
	"os"
	"testing"
)

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	if err := os.Unsetenv("DATABASE_URL"); err != nil {
		t.Fatal(err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected an error when DATABASE_URL is missing")
	}
}

func TestLoadDefaultsAddr(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://vynno:vynno@localhost:5432/vynno?sslmode=disable")
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
	if cfg.DatabaseURL == "" {
		t.Fatal("DatabaseURL is empty")
	}
}

func TestLoadUsesAddr(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://vynno:vynno@localhost:5432/vynno?sslmode=disable")
	t.Setenv("ADDR", ":9090")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != ":9090" {
		t.Fatalf("Addr = %q, want :9090", cfg.Addr)
	}
}
