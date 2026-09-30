package domain

import (
	"testing"
	"time"
	_ "time/tzdata"
)

func mustRange(t *testing.T, from, to, tz string) DayRange {
	t.Helper()
	r, err := ParseDayRange(from, to, tz)
	if err != nil {
		t.Fatalf("ParseDayRange(%q, %q, %q): %v", from, to, tz, err)
	}
	return r
}

func stopped(project string, activity *string, start string, d time.Duration) Session {
	s, _ := time.Parse(time.RFC3339Nano, start)
	end := s.Add(d)
	return Session{ProjectID: project, ActivityTypeID: activity, Status: StatusStopped, StartedAt: s, EndedAt: &end}
}

func TestParseDayRangeRejects(t *testing.T) {
	for _, tc := range []struct{ from, to, tz string }{
		{"", "2026-09-30", "UTC"},
		{"2026-9-1", "2026-09-30", "UTC"},
		{"2026-02-30", "2026-03-01", "UTC"},
		{"2026-09-30", "2026-09-29", "UTC"},
		{"2025-01-01", "2026-02-05", "UTC"}, // 401 days
		{"2026-09-01", "2026-09-30", ""},
		{"2026-09-01", "2026-09-30", "Local"},
		{"2026-09-01", "2026-09-30", "Mars/Olympus"},
		{"2026-09-01", "2026-09-30", "../etc/passwd"},
		{"2026-09-01", "2026-09-30", "/etc/localtime"},
	} {
		_, err := ParseDayRange(tc.from, tc.to, tc.tz)
		de, ok := err.(*Error)
		if !ok || de.Code != CodeInvalidQuery {
			t.Errorf("ParseDayRange(%q, %q, %q) = %v, want invalid_query", tc.from, tc.to, tc.tz, err)
		}
	}
	mustRange(t, "2025-01-01", "2026-02-04", "UTC") // 400 days
	mustRange(t, "2026-09-30", "2026-09-30", "Europe/Warsaw")
}

func TestBucketDayTotalsUsesLocalStartDate(t *testing.T) {
	coding := "act-coding"
	r := mustRange(t, "2026-03-28", "2026-03-30", "Europe/Warsaw")
	sessions := []Session{
		// 23:30 local on the 28th, runs past midnight: all of it counts on the 28th.
		stopped("p1", nil, "2026-03-28T22:30:00Z", 90*time.Minute),
		// 00:30 local on the 29th (22:30Z on the 28th, before the DST switch).
		stopped("p1", nil, "2026-03-28T23:30:00Z", time.Hour),
		// Same day, same project, an activity: a separate row.
		stopped("p1", &coding, "2026-03-29T08:00:00Z", 30*time.Minute),
		stopped("p1", &coding, "2026-03-29T09:00:00Z", 30*time.Minute),
		// Outside the range in local time (00:10 on the 31st, CEST).
		stopped("p1", nil, "2026-03-30T22:10:00Z", time.Hour),
		// Live sessions never count.
		{ProjectID: "p1", Status: StatusActive, StartedAt: time.Date(2026, 3, 30, 8, 0, 0, 0, time.UTC)},
	}
	got := BucketDayTotals(sessions, r)
	want := []DayTotal{
		{Date: "2026-03-28", ProjectID: "p1", DurationMs: 90 * 60_000, SessionCount: 1},
		{Date: "2026-03-29", ProjectID: "p1", DurationMs: 60 * 60_000, SessionCount: 1},
		{Date: "2026-03-29", ProjectID: "p1", ActivityTypeID: &coding, DurationMs: 60 * 60_000, SessionCount: 2},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d rows: %+v", len(got), got)
	}
	for i := range want {
		g, w := got[i], want[i]
		if g.Date != w.Date || g.ProjectID != w.ProjectID || g.DurationMs != w.DurationMs || g.SessionCount != w.SessionCount ||
			(g.ActivityTypeID == nil) != (w.ActivityTypeID == nil) {
			t.Errorf("row %d = %+v, want %+v", i, g, w)
		}
	}
}

func TestBucketDayTotalsMatchesWireMilliseconds(t *testing.T) {
	r := mustRange(t, "2026-09-30", "2026-09-30", "UTC")
	start := time.Date(2026, 9, 30, 8, 0, 0, 999_900_000, time.UTC) // .999900 → .999 on the wire
	end := time.Date(2026, 9, 30, 9, 0, 0, 999_000, time.UTC)       // .000999 → .000 on the wire
	got := BucketDayTotals([]Session{{ProjectID: "p", Status: StatusStopped, StartedAt: start, EndedAt: &end}}, r)
	if len(got) != 1 || got[0].DurationMs != 3_600_000-999 {
		t.Fatalf("got %+v, want %d ms", got, 3_600_000-999)
	}
}

func TestDayRangeQueryWindowCoversTheLocalDays(t *testing.T) {
	r := mustRange(t, "2026-09-01", "2026-09-30", "Pacific/Kiritimati") // UTC+14
	from, to := r.QueryWindow()
	firstLocal := time.Date(2026, 9, 1, 0, 0, 0, 0, r.Location)
	lastLocal := time.Date(2026, 9, 30, 23, 59, 59, 0, r.Location)
	if firstLocal.Before(from) || !lastLocal.Before(to) {
		t.Fatalf("window [%s, %s) misses [%s, %s]", from, to, firstLocal.UTC(), lastLocal.UTC())
	}
}
