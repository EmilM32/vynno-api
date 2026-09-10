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

func TestStartStop(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 3, 11, 8, 0, 0, 0, time.UTC)
	s := StartSession("s1", "p1", "Work", nil, nil, nil, start)
	if s.Status != StatusActive || s.EndedAt != nil {
		t.Fatalf("start: %#v", s)
	}

	stopped, err := Stop(s, start.Add(5*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if stopped.Status != StatusStopped || stopped.EndedAt == nil {
		t.Fatalf("stop: %#v", stopped)
	}
	if _, err := Stop(stopped, start); err == nil {
		t.Fatal("stop while stopped should fail")
	}
}

func TestApplySessionPatchNoteAndTimes(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 3, 11, 8, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)
	s := StartSession("s1", "p1", "Work", nil, nil, nil, start)
	stopped, err := Stop(s, end)
	if err != nil {
		t.Fatal(err)
	}

	note := "  "
	newStart := start.Add(time.Hour)
	newEnd := end.Add(time.Hour)
	got, err := ApplySessionPatch(stopped, SessionPatch{
		Note:      &note,
		StartedAt: &newStart,
		EndedAt:   &newEnd,
		EndedSet:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Note != UntitledNote {
		t.Fatalf("note = %q", got.Note)
	}
	if !got.StartedAt.Equal(newStart) || got.EndedAt == nil || !got.EndedAt.Equal(newEnd) {
		t.Fatalf("times: %#v", got)
	}
}

func TestApplySessionPatchRejectsLiveEndedAt(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 3, 11, 8, 0, 0, 0, time.UTC)
	s := StartSession("s1", "p1", "Work", nil, nil, nil, start)
	end := start.Add(time.Hour)
	if _, err := ApplySessionPatch(s, SessionPatch{EndedAt: &end, EndedSet: true}); err == nil {
		t.Fatal("expected invalid_body")
	}
}

func TestApplySessionPatchRejectsStoppedEndedAtClear(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 3, 11, 8, 0, 0, 0, time.UTC)
	s := StartSession("s1", "p1", "Work", nil, nil, nil, start)
	stopped, err := Stop(s, start.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplySessionPatch(stopped, SessionPatch{EndedSet: true, EndedAt: nil}); err == nil {
		t.Fatal("expected invalid_body")
	}
}

func TestManualSession(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 3, 11, 8, 0, 0, 0, time.UTC)
	end := start.Add(90 * time.Minute)
	s, err := ManualSession("s1", "p1", "  Forgot  ", nil, nil, nil, start, end)
	if err != nil {
		t.Fatal(err)
	}
	if s.Status != StatusStopped || s.Note != "Forgot" || s.EndedAt == nil {
		t.Fatalf("%#v", s)
	}
	if _, err := ManualSession("s1", "p1", "x", nil, nil, nil, end, start); err == nil {
		t.Fatal("expected invalid_body for reversed times")
	}
}
