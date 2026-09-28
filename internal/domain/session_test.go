package domain

import (
	"strings"
	"testing"
	"time"
)

func TestNormalizeNote(t *testing.T) {
	t.Parallel()
	got, err := NormalizeNote("  ")
	if err != nil || got != UntitledNote {
		t.Fatalf("got %q %v", got, err)
	}
	got, err = NormalizeNote("  Refactor  ")
	if err != nil || got != "Refactor" {
		t.Fatalf("got %q %v", got, err)
	}
}

func TestStartStop(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 3, 11, 8, 0, 0, 1500, time.UTC)
	s, err := StartSession("s1", "p1", "Work", nil, nil, nil, start)
	if err != nil {
		t.Fatal(err)
	}
	if s.Status != StatusActive || s.EndedAt != nil {
		t.Fatalf("start: %#v", s)
	}
	if s.StartedAt.Nanosecond() != 1000 {
		t.Fatalf("startedAt not truncated: %v", s.StartedAt)
	}

	stopped, err := Stop(s, start.Add(5*time.Minute+400*time.Nanosecond))
	if err != nil {
		t.Fatal(err)
	}
	if stopped.Status != StatusStopped || stopped.EndedAt == nil {
		t.Fatalf("stop: %#v", stopped)
	}
	if stopped.EndedAt.Nanosecond() != 1000 {
		t.Fatalf("endedAt not truncated: %v", stopped.EndedAt)
	}
	if _, err := Stop(stopped, start); err == nil {
		t.Fatal("stop while stopped should fail")
	}
}

func TestApplySessionPatchNoteAndTimes(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 3, 11, 8, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)
	now := end.Add(time.Hour)
	s, err := StartSession("s1", "p1", "Work", nil, nil, nil, start)
	if err != nil {
		t.Fatal(err)
	}
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
	}, now)
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
	s, err := StartSession("s1", "p1", "Work", nil, nil, nil, start)
	if err != nil {
		t.Fatal(err)
	}
	end := start.Add(time.Hour)
	if _, err := ApplySessionPatch(s, SessionPatch{EndedAt: &end, EndedSet: true}, end); err == nil {
		t.Fatal("expected invalid_body")
	}
}

func TestApplySessionPatchRejectsStoppedEndedAtClear(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 3, 11, 8, 0, 0, 0, time.UTC)
	s, err := StartSession("s1", "p1", "Work", nil, nil, nil, start)
	if err != nil {
		t.Fatal(err)
	}
	stopped, err := Stop(s, start.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplySessionPatch(stopped, SessionPatch{EndedSet: true, EndedAt: nil}, start.Add(2*time.Hour)); err == nil {
		t.Fatal("expected invalid_body")
	}
}

func TestManualSession(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 3, 11, 8, 0, 0, 0, time.UTC)
	end := start.Add(90 * time.Minute)
	s, err := ManualSession("s1", "p1", "  Forgot  ", nil, nil, nil, start, end, end)
	if err != nil {
		t.Fatal(err)
	}
	if s.Status != StatusStopped || s.Note != "Forgot" || s.EndedAt == nil {
		t.Fatalf("%#v", s)
	}
	if _, err := ManualSession("s1", "p1", "x", nil, nil, nil, end, start, end); err == nil {
		t.Fatal("expected invalid_body for reversed times")
	}
}

func TestParseISOTimeTruncatesToMicrosecond(t *testing.T) {
	t.Parallel()
	got, err := ParseISOTime("2026-03-11T08:00:00.000001999Z")
	if err != nil {
		t.Fatal(err)
	}
	if got.Nanosecond() != 1000 {
		t.Fatalf("ns = %d", got.Nanosecond())
	}
}

func TestManualSessionMicrosecondComparison(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 3, 11, 8, 0, 0, 0, time.UTC)
	now := start.Add(time.Hour)
	if _, err := ManualSession("s1", "p1", "x", nil, nil, nil, start, start.Add(time.Nanosecond), now); err == nil {
		t.Fatal("expected +1ns rejected")
	}
	if _, err := ManualSession("s1", "p1", "x", nil, nil, nil, start, start.Add(999*time.Nanosecond), now); err == nil {
		t.Fatal("expected +999ns rejected")
	}
	s, err := ManualSession("s1", "p1", "x", nil, nil, nil, start, start.Add(time.Microsecond), now)
	if err != nil {
		t.Fatal(err)
	}
	if s.EndedAt.Sub(s.StartedAt) != time.Microsecond {
		t.Fatalf("delta = %s", s.EndedAt.Sub(s.StartedAt))
	}
}

func TestSessionTimeBounds(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	year1 := time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := ManualSession("s1", "p1", "x", nil, nil, nil, year1, year1.Add(time.Hour), now); err == nil {
		t.Fatal("expected year 0001 rejected")
	}

	if _, err := ManualSession("s1", "p1", "x", nil, nil, nil, MinSessionTime, MinSessionTime.Add(time.Minute), now); err != nil {
		t.Fatal(err)
	}
	before := MinSessionTime.Add(-time.Microsecond)
	if _, err := ManualSession("s1", "p1", "x", nil, nil, nil, before, before.Add(time.Minute), now); err == nil {
		t.Fatal("expected before min rejected")
	}

	latest := now.Add(MaxFutureSkew)
	if _, err := ManualSession("s1", "p1", "x", nil, nil, nil, latest.Add(-time.Minute), latest, now); err != nil {
		t.Fatal(err)
	}
	tooLate := latest.Add(time.Microsecond)
	if _, err := ManualSession("s1", "p1", "x", nil, nil, nil, tooLate.Add(-time.Minute), tooLate, now); err == nil {
		t.Fatal("expected future skew rejected")
	}

	longStart := now.Add(-MaxSessionDuration)
	if _, err := ManualSession("s1", "p1", "x", nil, nil, nil, longStart, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := ManualSession("s1", "p1", "x", nil, nil, nil, longStart.Add(-time.Microsecond), now, now); err == nil {
		t.Fatal("expected duration over 7 days rejected")
	}
}

func TestLiveStartedAtSkew(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	s, err := StartSession("s1", "p1", "Work", nil, nil, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	future := now.Add(time.Hour)
	if _, err := ApplySessionPatch(s, SessionPatch{StartedAt: &future}, now); err == nil {
		t.Fatal("expected now+1h rejected")
	}
	past := now.Add(-3 * time.Hour)
	got, err := ApplySessionPatch(s, SessionPatch{StartedAt: &past}, now)
	if err != nil {
		t.Fatal(err)
	}
	if !got.StartedAt.Equal(past) || got.Status != StatusActive {
		t.Fatalf("%#v", got)
	}
}

func TestNoteOnlyLegacyPatchSkipsBounds(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	start := time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	legacy := Session{
		ID: "s1", ProjectID: "p1", Note: "old", Status: StatusStopped,
		StartedAt: start, EndedAt: &end,
	}
	note := "updated"
	got, err := ApplySessionPatch(legacy, SessionPatch{Note: &note}, now)
	if err != nil {
		t.Fatal(err)
	}
	if got.Note != "updated" || !got.StartedAt.Equal(start) {
		t.Fatalf("%#v", got)
	}

	legacy.Note = strings.Repeat("x", 600)
	got, err = ApplySessionPatch(legacy, SessionPatch{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if got.Note != legacy.Note {
		t.Fatal("omitted note should stay")
	}

	same := start
	broken := legacy
	broken.EndedAt = &same
	broken.Note = "still"
	got, err = ApplySessionPatch(broken, SessionPatch{Note: &note}, now)
	if err != nil {
		t.Fatal(err)
	}
	if got.Note != "updated" {
		t.Fatalf("grandfathered structural note = %q", got.Note)
	}

	y1 := start
	valid, err := StartSession("s1", "p1", "Work", nil, nil, nil, now.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	stopped, err := Stop(valid, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplySessionPatch(stopped, SessionPatch{StartedAt: &y1}, now); err == nil {
		t.Fatal("expected setting year 0001 rejected")
	}
}

func TestNormalizeTargetDurationMsBounds(t *testing.T) {
	t.Parallel()
	max := MaxTargetDurationMs
	got, err := NormalizeTargetDurationMs(&max)
	if err != nil || got == nil || *got != max {
		t.Fatalf("max: %#v %v", got, err)
	}
	over := max + 1
	if _, err := NormalizeTargetDurationMs(&over); err == nil {
		t.Fatal("expected over max rejected")
	}
	neg := int64(-1)
	if _, err := NormalizeTargetDurationMs(&neg); err == nil {
		t.Fatal("expected negative rejected")
	}
}
