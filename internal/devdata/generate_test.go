package devdata

import (
	"slices"
	"testing"
	"time"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/EmilM32/vynno-api/internal/store"
)

func testNow() time.Time {
	return time.Date(2026, 8, 19, 15, 30, 0, 0, time.UTC)
}

func testOpts() Options {
	return Options{
		Now:               testNow(),
		BootstrapEmail:    "alexdev@vynno.local",
		BootstrapPassword: "local-dev-password",
		SeedPassword:      "local-dev-password",
	}
}

func TestBuildReset(t *testing.T) {
	ds := BuildReset(testOpts())
	if len(ds.Accounts) != 1 {
		t.Fatalf("accounts = %d", len(ds.Accounts))
	}
	a := ds.Accounts[0]
	if a.ID != store.DefaultUserID() {
		t.Fatalf("id = %s", a.ID)
	}
	if a.Email != "alexdev@vynno.local" || a.Password != "local-dev-password" {
		t.Fatalf("creds = %s / %s", a.Email, a.Password)
	}
	if len(a.Projects) != 1 || a.Projects[0].ID != store.DefaultProject().ID {
		t.Fatalf("projects = %+v", a.Projects)
	}
	if len(a.Sessions) != 0 {
		t.Fatalf("sessions = %d", len(a.Sessions))
	}
	if a.Projects[0].Archived {
		t.Fatal("bootstrap project is archived")
	}
}

func TestBuildSeedInvariants(t *testing.T) {
	ds := BuildSeed(testOpts())
	if len(ds.Accounts) != 3 {
		t.Fatalf("accounts = %d", len(ds.Accounts))
	}

	wantUsers := []string{"alexdev@vynno.local", "maya@vynno.local", "rio@vynno.local"}
	for i, want := range wantUsers {
		if ds.Accounts[i].Email != want {
			t.Fatalf("account[%d] = %s", i, ds.Accounts[i].Email)
		}
	}
	if ds.Accounts[0].ID != store.DefaultUserID() {
		t.Fatalf("alexdev id = %s", ds.Accounts[0].ID)
	}
	if ds.Accounts[0].Password != "local-dev-password" {
		t.Fatal("alexdev password")
	}
	if ds.Accounts[1].Password != "local-dev-password" || ds.Accounts[2].Password != "local-dev-password" {
		t.Fatal("demo passwords")
	}

	assertAccount(t, ds.Accounts[0], 7, 220, 400, true)
	assertAccount(t, ds.Accounts[1], 4, 60, 140, false)
	assertAccount(t, ds.Accounts[2], 2, 8, 40, false)
}

func assertAccount(t *testing.T, a Account, projects, sessMin, sessMax int, live bool) {
	t.Helper()
	if len(a.Projects) != projects {
		t.Fatalf("%s projects = %d want %d", a.Email, len(a.Projects), projects)
	}
	n := len(a.Sessions)
	t.Logf("%s: %d projects, %d sessions (live=%v)", a.Email, len(a.Projects), n, live)
	if n < sessMin || n > sessMax {
		t.Fatalf("%s sessions = %d want %d–%d", a.Email, n, sessMin, sessMax)
	}

	active := 0
	codes := map[string]int{}
	archivedIDs := map[string]bool{}
	for _, p := range a.Projects {
		if !p.Archived {
			active++
		} else {
			archivedIDs[p.ID] = true
		}
		if p.Code != nil {
			key := *p.Code
			codes[key]++
			if codes[key] > 1 {
				t.Fatalf("%s duplicate code %s", a.Email, key)
			}
		}
		if _, err := domain.NormalizeColor(p.Color); err != nil {
			t.Fatalf("%s color %s: %v", a.Email, p.Color, err)
		}
		if p.Code != nil {
			if _, err := domain.NormalizeCode(p.Code); err != nil {
				t.Fatalf("%s code %s: %v", a.Email, *p.Code, err)
			}
		}
	}
	if active < 1 {
		t.Fatalf("%s has no active project", a.Email)
	}

	typeIDs := map[string]struct{}{}
	for _, at := range a.ActivityTypes {
		if _, err := domain.NormalizeActivityTypeName(at.Name); err != nil {
			t.Fatalf("%s activity name %s: %v", a.Email, at.Name, err)
		}
		if _, err := domain.NormalizeActivityColor(at.Color); err != nil {
			t.Fatalf("%s activity color %s: %v", a.Email, at.Color, err)
		}
		typeIDs[at.ID] = struct{}{}
	}

	var liveCount int
	byStart := append([]domain.Session(nil), a.Sessions...)
	slices.SortFunc(byStart, func(x, y domain.Session) int {
		return x.StartedAt.Compare(y.StartedAt)
	})
	var prevEnd time.Time
	for i, s := range byStart {
		if domain.IsLiveStatus(s.Status) {
			liveCount++
			if s.Status != domain.StatusActive {
				t.Fatalf("%s live status = %s", a.Email, s.Status)
			}
			if s.EndedAt != nil {
				t.Fatalf("%s live session has endedAt", a.Email)
			}
			if s.PausedAt != nil {
				t.Fatalf("%s live session has pausedAt", a.Email)
			}
		} else {
			if s.Status != domain.StatusStopped {
				t.Fatalf("%s status = %s", a.Email, s.Status)
			}
			if s.EndedAt == nil || !s.EndedAt.After(s.StartedAt) {
				t.Fatalf("%s stopped times %s → %v", a.Email, s.StartedAt, s.EndedAt)
			}
			if s.PausedAt != nil {
				t.Fatalf("%s stopped session still paused", a.Email)
			}
			wall := s.EndedAt.Sub(s.StartedAt).Milliseconds()
			if s.PausedMs < 0 || s.PausedMs >= wall {
				t.Fatalf("%s pausedMs=%d wall=%d", a.Email, s.PausedMs, wall)
			}
		}
		if s.ActivityTypeID != nil {
			if _, ok := typeIDs[*s.ActivityTypeID]; !ok {
				t.Fatalf("%s activityTypeId %s not in account types", a.Email, *s.ActivityTypeID)
			}
		}
		if s.Note == "" {
			t.Fatalf("%s empty note (want Untitled session)", a.Email)
		}
		if archivedIDs[s.ProjectID] && s.Status != domain.StatusStopped {
			t.Fatalf("%s live session on archived project", a.Email)
		}
		if i > 0 && s.StartedAt.Before(prevEnd) {
			t.Fatalf("%s overlap: %s starts before previous end %s", a.Email, s.StartedAt, prevEnd)
		}
		if s.EndedAt != nil {
			prevEnd = *s.EndedAt
		} else {
			prevEnd = s.StartedAt
		}
	}
	if live && liveCount != 1 {
		t.Fatalf("%s live sessions = %d", a.Email, liveCount)
	}
	if !live && liveCount != 0 {
		t.Fatalf("%s unexpected live sessions = %d", a.Email, liveCount)
	}
}

func TestBuildSeedDeterministicShape(t *testing.T) {
	opts := testOpts()
	a := BuildSeed(opts)
	b := BuildSeed(opts)
	if len(a.Accounts) != len(b.Accounts) {
		t.Fatal("account count drifted")
	}
	for i := range a.Accounts {
		if len(a.Accounts[i].Sessions) != len(b.Accounts[i].Sessions) {
			t.Fatalf("%s session count %d vs %d", a.Accounts[i].Email, len(a.Accounts[i].Sessions), len(b.Accounts[i].Sessions))
		}
		if len(a.Accounts[i].Projects) != len(b.Accounts[i].Projects) {
			t.Fatalf("%s project count drifted", a.Accounts[i].Email)
		}
	}
}

func TestBuildSeedHasTodayAndThisWeek(t *testing.T) {
	now := testNow()
	ds := BuildSeed(testOpts())
	alex := ds.Accounts[0]
	var today, week int
	weekStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -int(now.Weekday()))
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	for _, s := range alex.Sessions {
		if !s.StartedAt.Before(todayStart) {
			today++
		}
		if !s.StartedAt.Before(weekStart) {
			week++
		}
	}
	if today < 1 {
		t.Fatalf("no sessions today (have %d)", today)
	}
	if week < 5 {
		t.Fatalf("thin week: %d sessions", week)
	}
}
