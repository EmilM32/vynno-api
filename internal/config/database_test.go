package config

import (
	"os"
	"testing"
)

func TestDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://vynno:vynno@localhost:5433/vynno?sslmode=disable")
	got, err := DatabaseURL()
	if err != nil {
		t.Fatal(err)
	}
	if got != "postgres://vynno:vynno@localhost:5433/vynno?sslmode=disable" {
		t.Fatalf("url = %s", got)
	}
}

func TestDatabaseURLRequired(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://set")
	if err := os.Unsetenv("DATABASE_URL"); err != nil {
		t.Fatal(err)
	}
	if _, err := DatabaseURL(); err == nil {
		t.Fatal("expected an error when DATABASE_URL is missing")
	}
}
