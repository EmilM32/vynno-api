package store

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// testDatabaseEnv names a Postgres server the tests may use. They never touch a
// database that already exists there: each call to newTestPostgres creates its
// own vynno_test_<random> database, migrates it from zero, and drops it.
const testDatabaseEnv = "TEST_DATABASE_URL"

// newTestPostgres returns a Postgres store on a fresh, fully migrated database.
// Without TEST_DATABASE_URL the test is skipped, except in CI, where a missing
// server must not pass silently.
func newTestPostgres(t *testing.T) (*Postgres, *sql.DB) {
	t.Helper()
	raw := os.Getenv(testDatabaseEnv)
	if raw == "" {
		if os.Getenv("CI") != "" {
			t.Fatalf("%s is required in CI", testDatabaseEnv)
		}
		t.Skipf("%s is not set; skipping Postgres tests", testDatabaseEnv)
	}
	base, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("%s: %v", testDatabaseEnv, err)
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
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate from zero: %v", err)
	}
	return NewPostgres(db), db
}

func TestPostgresContract(t *testing.T) {
	pg, _ := newTestPostgres(t)
	runStoreContract(t, pg)
}

// Migrate is run at every process start, so a second run on a current schema
// must be a no-op.
func TestPostgresMigrateTwice(t *testing.T) {
	_, db := newTestPostgres(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
	var n int
	if err := db.QueryRow("SELECT count(*) FROM goose_db_version WHERE is_applied").Scan(&n); err != nil {
		t.Fatal(err)
	}
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		t.Fatal(err)
	}
	// goose keeps one extra row for version 0.
	if n != len(entries)+1 {
		t.Fatalf("applied versions = %d, want %d migration files + 1", n, len(entries))
	}
}

// Migration 00014: the database itself refuses a session or a default project
// that points at another account's rows.
func TestPostgresOwnerScopedForeignKeys(t *testing.T) {
	pg, _ := newTestPostgres(t)
	ctx := context.Background()
	alice := contractUser(t, pg)
	bob := contractUser(t, pg)
	bobProject := contractProject(t, pg, bob, "Bob", nil)
	aliceProject := contractProject(t, pg, alice, "Alice", nil)
	bobType, err := pg.CreateActivityType(ctx, bob, domain.ActivityType{ID: uuid.NewString(), Name: "Deep", Color: "#112233"})
	if err != nil {
		t.Fatal(err)
	}

	sess := contractStopped(aliceProject.ID, contractT0, time.Hour)
	sess.ProjectID = bobProject.ID
	if _, err := pg.CreateSession(ctx, alice, sess); err == nil {
		t.Fatal("session on another account's project was stored")
	}

	sess = contractStopped(aliceProject.ID, contractT0, time.Hour)
	sess.ActivityTypeID = &bobType.ID
	if _, err := pg.CreateSession(ctx, alice, sess); err == nil {
		t.Fatal("session with another account's activity type was stored")
	}

	err = pg.SavePrefs(ctx, alice, domain.Prefs{DefaultProjectID: &bobProject.ID})
	wantCode(t, err, domain.CodeNotFound)
}

// A project with sessions cannot be hard-deleted even if the service check is
// skipped or loses a race.
func TestPostgresProjectWithSessionsIsNotDeleted(t *testing.T) {
	pg, _ := newTestPostgres(t)
	ctx := context.Background()
	user := contractUser(t, pg)
	project := contractProject(t, pg, user, "Kept", nil)
	if _, err := pg.CreateSession(ctx, user, contractStopped(project.ID, contractT0, time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := pg.DeleteProject(ctx, user, uuid.MustParse(project.ID)); err == nil {
		t.Fatal("project with sessions was deleted")
	}
	if _, err := pg.GetProject(ctx, user, uuid.MustParse(project.ID)); err != nil {
		t.Fatalf("project is gone: %v", err)
	}
}

func TestPostgresBootstrap(t *testing.T) {
	pg, _ := newTestPostgres(t)
	ctx := context.Background()

	if _, ok, err := pg.FirstUserID(ctx); err != nil || ok {
		t.Fatalf("FirstUserID on empty database = %v, %v", ok, err)
	}

	id := DefaultUserID()
	if err := pg.Bootstrap(ctx, id, "alexdev@vynno.local", "hash-1", DefaultProfile(), DefaultProject()); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	got, ok, err := pg.FirstUserID(ctx)
	if err != nil || !ok || got != id {
		t.Fatalf("FirstUserID = %v, %v, %v", got, ok, err)
	}
	profile, err := pg.GetProfile(ctx, id)
	if err != nil || profile.DisplayName != DefaultProfile().DisplayName || profile.Email != "alexdev@vynno.local" {
		t.Fatalf("profile = %+v, %v", profile, err)
	}
	projects, err := pg.ListProjects(ctx, id, true)
	if err != nil || len(projects) != 1 || projects[0].Name != DefaultProject().Name {
		t.Fatalf("projects = %+v, %v", projects, err)
	}

	// A second bootstrap keeps the stored password and adds no second account.
	if err := pg.Bootstrap(ctx, id, "alexdev@vynno.local", "hash-2", DefaultProfile(), DefaultProject()); err != nil {
		t.Fatal(err)
	}
	if err := pg.Bootstrap(ctx, uuid.New(), "other@vynno.local", "hash-3", DefaultProfile(), DefaultProject()); err != nil {
		t.Fatal(err)
	}
	acc, err := pg.GetAccountByID(ctx, id)
	if err != nil || acc.PasswordHash != "hash-1" {
		t.Fatalf("account = %+v, %v", acc, err)
	}
	if _, err := pg.GetAccountByEmail(ctx, "other@vynno.local"); err == nil {
		t.Fatal("bootstrap created a second account")
	}
}

// The read-only MCP queries (internal/store/queries/agentread.sql).
func TestPostgresAgentRead(t *testing.T) {
	pg, _ := newTestPostgres(t)
	ctx := context.Background()
	user := contractUser(t, pg)
	other := contractUser(t, pg)
	project := contractProject(t, pg, user, "Identity", nil)
	archived := contractProject(t, pg, user, "Old", nil)
	archived.Archived = true
	if _, err := pg.UpdateProject(ctx, user, archived); err != nil {
		t.Fatal(err)
	}
	otherProject := contractProject(t, pg, other, "Theirs", nil)
	typ, err := pg.CreateActivityType(ctx, user, domain.ActivityType{ID: uuid.NewString(), Name: "Review", Color: "#112233"})
	if err != nil {
		t.Fatal(err)
	}

	first := contractStopped(project.ID, contractT0, time.Hour)
	first.ActivityTypeID = &typ.ID
	second := contractStopped(archived.ID, contractT0.Add(2*time.Hour), time.Hour)
	live := contractLive(project.ID, contractT0.Add(4*time.Hour))
	for _, s := range []domain.Session{first, second, live} {
		if _, err := pg.CreateSession(ctx, user, s); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pg.CreateSession(ctx, other, contractStopped(otherProject.ID, contractT0, time.Hour)); err != nil {
		t.Fatal(err)
	}

	projects, err := pg.ListAgentProjects(ctx, user, false)
	if err != nil || len(projects) != 1 || projects[0].ID != project.ID {
		t.Fatalf("active projects = %+v, %v", projects, err)
	}
	projects, err = pg.ListAgentProjects(ctx, user, true)
	if err != nil || len(projects) != 2 {
		t.Fatalf("all projects = %+v, %v", projects, err)
	}

	all, err := pg.ListAgentSessions(ctx, user, AgentSessionFilter{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 || all[0].ID != live.ID || all[1].ID != second.ID || all[2].ID != first.ID {
		t.Fatalf("sessions newest first = %+v", all)
	}
	if all[2].ProjectName != "Identity" || all[2].ActivityTypeName == nil || *all[2].ActivityTypeName != "Review" {
		t.Fatalf("joined names = %+v", all[2])
	}
	if all[1].ProjectName != "Old" || all[1].ActivityTypeName != nil || all[1].ActivityTypeID != nil {
		t.Fatalf("session without activity type = %+v", all[1])
	}

	pid := uuid.MustParse(archived.ID)
	byProject, err := pg.ListAgentSessions(ctx, user, AgentSessionFilter{ProjectID: &pid, Limit: 10})
	if err != nil || len(byProject) != 1 || byProject[0].ID != second.ID {
		t.Fatalf("by project = %+v, %v", byProject, err)
	}
	stopped, err := pg.ListAgentSessions(ctx, user, AgentSessionFilter{Status: domain.StatusStopped, Limit: 1})
	if err != nil || len(stopped) != 1 || stopped[0].ID != second.ID {
		t.Fatalf("stopped, limit 1 = %+v, %v", stopped, err)
	}
	if _, err := pg.ListAgentSessions(ctx, user, AgentSessionFilter{Limit: AgentSessionLimitMax + 1}); err == nil {
		t.Fatal("limit above the cap was accepted")
	}

	got, ok, err := pg.GetAgentLiveSession(ctx, user)
	if err != nil || !ok || got.ID != live.ID || got.EndedAt != nil {
		t.Fatalf("live = %+v, %v, %v", got, ok, err)
	}
	if _, ok, err := pg.GetAgentLiveSession(ctx, other); err != nil || ok {
		t.Fatalf("other account live = %v, %v", ok, err)
	}

	// [T0+30m, T0+2h30m) overlaps the first two; the live one starts later.
	window, err := pg.ListAgentSessionsInWindow(ctx, user, contractT0.Add(30*time.Minute), contractT0.Add(150*time.Minute))
	if err != nil || len(window) != 2 || window[0].ID != first.ID || window[1].ID != second.ID {
		t.Fatalf("window oldest first = %+v, %v", window, err)
	}
}
