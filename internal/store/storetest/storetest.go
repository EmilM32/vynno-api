// Package storetest gives tests a real Postgres store. It is imported only by
// _test files.
package storetest

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/EmilM32/vynno-api/internal/store"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// DatabaseEnv names a Postgres server the tests may use. They never touch a
// database that already exists there: each call to NewPostgres creates its
// own vynno_test_<random> database, migrates it from zero, and drops it.
const DatabaseEnv = "TEST_DATABASE_URL"

// NewPostgres returns a Postgres store on a fresh, fully migrated database.
// Without TEST_DATABASE_URL the test is skipped, except in CI, where a missing
// server must not pass silently.
func NewPostgres(t testing.TB) (*store.Postgres, *sql.DB) {
	t.Helper()
	raw := os.Getenv(DatabaseEnv)
	if raw == "" {
		if os.Getenv("CI") != "" {
			t.Fatalf("%s is required in CI", DatabaseEnv)
		}
		t.Skipf("%s is not set; skipping Postgres tests", DatabaseEnv)
	}
	base, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("%s: %v", DatabaseEnv, err)
	}

	admin, err := sql.Open("pgx", raw)
	if err != nil {
		t.Fatalf("open admin: %v", err)
	}
	name := "vynno_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := admin.Exec("CREATE DATABASE " + name); err != nil {
		_ = admin.Close()
		t.Fatalf("create database: %v", err)
	}

	scoped := *base
	scoped.Path = "/" + name
	db, err := sql.Open("pgx", scoped.String())
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		// name is generated above, never input.
		if _, err := admin.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", name)); err != nil {
			t.Errorf("drop database %s: %v", name, err)
		}
		_ = admin.Close()
	})
	goose.SetLogger(goose.NopLogger())
	if err := store.Migrate(db); err != nil {
		t.Fatalf("migrate from zero: %v", err)
	}
	return store.NewPostgres(db), db
}
