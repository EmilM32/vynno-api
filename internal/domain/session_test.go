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

func TestApplySessionPatchNoteAndTimes(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 3, 11, 8, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)
	s := StartSession("s1", "p1", "Work", nil, nil, nil, nil, start)
	stopped, err := Stop(s, end)
	if err != nil {
		t.Fatal(err)
	}

	note := "  "
	newStart := start.Add(time.Hour)
	newEnd := end.Add(time.Hour)
	zero := int64(0)
	got, err := ApplySessionPatch(stopped, SessionPatch{
		Note:      &note,
		StartedAt: &newStart,
		EndedAt:   &newEnd,
		EndedSet:  true,
		PausedMs:  &zero,
	}, end)
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
	s := StartSession("s1", "p1", "Work", nil, nil, nil, nil, start)
	end := start.Add(time.Hour)
	if _, err := ApplySessionPatch(s, SessionPatch{EndedAt: &end, EndedSet: true}, start.Add(30*time.Minute)); err == nil {
		t.Fatal("expected invalid_body")
	}
}

func TestApplySessionPatchRejectsStoppedEndedAtClear(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 3, 11, 8, 0, 0, 0, time.UTC)
	s := StartSession("s1", "p1", "Work", nil, nil, nil, nil, start)
	stopped, err := Stop(s, start.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplySessionPatch(stopped, SessionPatch{EndedSet: true, EndedAt: nil}, start); err == nil {
		t.Fatal("expected invalid_body")
	}
}

func TestApplySessionPatchPausedMsExceedsInterval(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 3, 11, 8, 0, 0, 0, time.UTC)
	s := StartSession("s1", "p1", "Work", nil, nil, nil, nil, start)
	stopped, err := Stop(s, start.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	tooMuch := (2 * time.Hour).Milliseconds()
	if _, err := ApplySessionPatch(stopped, SessionPatch{PausedMs: &tooMuch}, start); err == nil {
		t.Fatal("expected invalid_body")
	}
}

func TestApplySessionPatchPausedStartedAtAfterPausedAt(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 3, 11, 8, 0, 0, 0, time.UTC)
	s := StartSession("s1", "p1", "Work", nil, nil, nil, nil, start)
	paused, err := Pause(s, start.Add(10*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	late := start.Add(20 * time.Minute)
	if _, err := ApplySessionPatch(paused, SessionPatch{StartedAt: &late}, start.Add(30*time.Minute)); err == nil {
		t.Fatal("expected invalid_body")
	}
}

func TestManualSession(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 3, 11, 8, 0, 0, 0, time.UTC)
	end := start.Add(90 * time.Minute)
	s, err := ManualSession("s1", "p1", "  Forgot  ", nil, nil, nil, nil, start, end, 0)
	if err != nil {
		t.Fatal(err)
	}
	if s.Status != StatusStopped || s.Note != "Forgot" || s.EndedAt == nil {
		t.Fatalf("%#v", s)
	}
	if _, err := ManualSession("s1", "p1", "x", nil, nil, nil, nil, end, start, 0); err == nil {
		t.Fatal("expected invalid_body for reversed times")
	}
}
