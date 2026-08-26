package devdata

import (
	"context"
	"testing"

	"github.com/EmilM32/vynno-api/internal/store"
	"golang.org/x/crypto/bcrypt"
)

func TestApplyMemory(t *testing.T) {
	ctx := context.Background()
	ds := BuildSeed(testOpts())
	mem := store.NewEmptyMemory()
	if err := Apply(ctx, mem, ds); err != nil {
		t.Fatal(err)
	}
	for _, acc := range ds.Accounts {
		got, err := mem.GetAccountByEmail(ctx, acc.Email)
		if err != nil {
			t.Fatalf("lookup %s: %v", acc.Email, err)
		}
		if err := bcrypt.CompareHashAndPassword([]byte(got.PasswordHash), []byte(acc.Password)); err != nil {
			t.Fatalf("password %s: %v", acc.Email, err)
		}
		projects, err := mem.ListProjects(ctx, acc.ID, true)
		if err != nil {
			t.Fatal(err)
		}
		if len(projects) != len(acc.Projects) {
			t.Fatalf("%s projects stored %d want %d", acc.Email, len(projects), len(acc.Projects))
		}
		sessions, err := mem.ListSessions(ctx, acc.ID, nil, 10000, "")
		if err != nil {
			t.Fatal(err)
		}
		if len(sessions.Items) != len(acc.Sessions) {
			t.Fatalf("%s sessions stored %d want %d", acc.Email, len(sessions.Items), len(acc.Sessions))
		}
		_, live, err := mem.GetLiveSession(ctx, acc.ID)
		if err != nil {
			t.Fatal(err)
		}
		if acc.Email == "alexdev@vynno.local" && !live {
			t.Fatal("alexdev should have a live session")
		}
		if acc.Email != "alexdev@vynno.local" && live {
			t.Fatalf("%s should be idle", acc.Email)
		}
	}
}

func TestApplyResetMemory(t *testing.T) {
	ctx := context.Background()
	ds := BuildReset(testOpts())
	mem := store.NewEmptyMemory()
	if err := Apply(ctx, mem, ds); err != nil {
		t.Fatal(err)
	}
	sessions, err := mem.ListSessions(ctx, ds.Accounts[0].ID, nil, 10000, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions.Items) != 0 {
		t.Fatalf("reset sessions = %d", len(sessions.Items))
	}
	projects, err := mem.ListProjects(ctx, ds.Accounts[0].ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 {
		t.Fatalf("reset projects = %d", len(projects))
	}
}
