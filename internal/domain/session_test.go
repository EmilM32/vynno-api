package domain

import (
	"testing"
	"time"
)

func TestNormalizeNote(t *testing.T) {
	t.Parallel()
	if got := NormalizeNote("  "); got != UntitledNote {
		t.Fatalf("got %q", got)
	}
	if got := NormalizeNote("  Refactor  "); got != "Refactor" {
		t.Fatalf("got %q", got)
	}
}

func TestPauseResumeStop(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 3, 11, 8, 0, 0, 0, time.UTC)
	s := StartSession("s1", "p1", "Work", nil, nil, nil, nil, start)

	pausedAt := start.Add(10 * time.Minute)
	paused, err := Pause(s, pausedAt)
	if err != nil {
		t.Fatal(err)
	}
	if paused.Status != StatusPaused || paused.PausedAt == nil {
		t.Fatalf("pause: %#v", paused)
	}

	resumedAt := pausedAt.Add(2 * time.Minute)
	resumed, err := Resume(paused, resumedAt)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Status != StatusActive || resumed.PausedAt != nil {
		t.Fatalf("resume: %#v", resumed)
	}
	if resumed.PausedMs != (2 * time.Minute).Milliseconds() {
		t.Fatalf("pausedMs = %d", resumed.PausedMs)
	}

	if _, err := Resume(resumed, resumedAt); err == nil {
		t.Fatal("resume while active should fail")
	}

	stopped, err := Stop(resumed, resumedAt.Add(5*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if stopped.Status != StatusStopped || stopped.EndedAt == nil {
		t.Fatalf("stop: %#v", stopped)
	}
	if _, err := Stop(stopped, resumedAt); err == nil {
		t.Fatal("stop while stopped should fail")
	}
}

func TestStopFromPausedFoldsPause(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 3, 11, 8, 0, 0, 0, time.UTC)
	s := StartSession("s1", "p1", "Work", nil, nil, nil, nil, start)
	paused, err := Pause(s, start.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	stopped, err := Stop(paused, start.Add(4*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if stopped.PausedMs != (3 * time.Minute).Milliseconds() {
		t.Fatalf("pausedMs = %d", stopped.PausedMs)
	}
	if stopped.PausedAt != nil {
		t.Fatal("pausedAt should be cleared")
	}
}

func TestPauseWhilePaused(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 3, 11, 8, 0, 0, 0, time.UTC)
	s := StartSession("s1", "p1", "Work", nil, nil, nil, nil, now)
	paused, err := Pause(s, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Pause(paused, now); err == nil {
		t.Fatal("expected invalid_transition")
	}
}
