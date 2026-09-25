package agentread

import (
	"testing"
	"time"

	"github.com/EmilM32/vynno-api/internal/store"
)

func TestClipInterval(t *testing.T) {
	from := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, 1)
	now := time.Date(2026, 9, 25, 11, 30, 0, 0, time.UTC)

	insideStart := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	insideEnd := insideStart.Add(time.Hour)
	start, end, ok := clipInterval(insideStart, &insideEnd, from, to, now)
	if !ok || end.Sub(start) != time.Hour {
		t.Fatalf("inside overlap = %s %s %v", start, end, ok)
	}

	crossStart := time.Date(2026, 9, 24, 22, 0, 0, 0, time.UTC)
	crossEnd := time.Date(2026, 9, 25, 2, 0, 0, 0, time.UTC)
	start, end, ok = clipInterval(crossStart, &crossEnd, from, to, now)
	if !ok || !start.Equal(from) || !end.Equal(crossEnd) {
		t.Fatalf("cross-midnight overlap = %s %s %v", start, end, ok)
	}

	liveStart := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	start, end, ok = clipInterval(liveStart, nil, from, to, now)
	if !ok || !end.Equal(now) || end.Sub(start) != 90*time.Minute {
		t.Fatalf("live overlap = %s %s %v", start, end, ok)
	}

	endedBefore := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	if _, _, ok = clipInterval(endedBefore, &crossStart, from, to, now); ok {
		t.Fatal("session that ended before the window should not overlap")
	}
}

func TestSplitDays(t *testing.T) {
	start := time.Date(2026, 9, 24, 22, 0, 0, 0, time.UTC)
	end := time.Date(2026, 9, 25, 2, 0, 0, 0, time.UTC)
	parts := splitDays(start, end)
	if len(parts) != 2 || parts[0].day != "2026-09-24" || parts[0].ms != 2*time.Hour.Milliseconds() {
		t.Fatalf("first slice = %+v", parts)
	}
	if parts[1].day != "2026-09-25" || parts[1].ms != 2*time.Hour.Milliseconds() {
		t.Fatalf("second slice = %+v", parts)
	}
}

func TestSummarizeClipsToWindow(t *testing.T) {
	from := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, 1)
	now := time.Date(2026, 9, 25, 11, 30, 0, 0, time.UTC)
	ended := time.Date(2026, 9, 25, 2, 0, 0, 0, time.UTC)
	activityID := "act-1"
	activityName := "coding"
	rows := []store.AgentSession{
		{
			ID: "s-cross", ProjectID: "p1", ProjectName: "Identity",
			StartedAt: time.Date(2026, 9, 24, 22, 0, 0, 0, time.UTC), EndedAt: &ended,
			Status: "stopped",
		},
		{
			ID: "s-live", ProjectID: "p1", ProjectName: "Identity",
			StartedAt: time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC), Status: "active",
			ActivityTypeID: &activityID, ActivityTypeName: &activityName,
		},
		{
			ID: "s-other", ProjectID: "p2", ProjectName: "Other",
			StartedAt: time.Date(2026, 9, 25, 8, 0, 0, 0, time.UTC),
			EndedAt:   ptrTime(time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)),
			Status:    "stopped",
		},
	}

	got := summarizeRows(rows, from, to, now, groupProject)
	if got.SessionCount != 3 {
		t.Fatalf("sessions = %d", got.SessionCount)
	}
	// 2h overlap + 1.5h live + 1h other = 4.5h
	if got.TotalDurationMs != (2*time.Hour + 90*time.Minute + time.Hour).Milliseconds() {
		t.Fatalf("total = %d", got.TotalDurationMs)
	}
	if len(got.Groups) != 2 || got.Groups[0].Label != "Identity" {
		t.Fatalf("groups = %+v", got.Groups)
	}

	byDay := summarizeRows(rows, from, to, now, groupDay)
	if len(byDay.Groups) != 1 || byDay.Groups[0].Key != "2026-09-25" {
		t.Fatalf("days = %+v", byDay.Groups)
	}
	if byDay.TotalDurationMs != got.TotalDurationMs {
		t.Fatalf("day total %d != project total %d", byDay.TotalDurationMs, got.TotalDurationMs)
	}

	byActivity := summarizeRows(rows, from, to, now, groupActivity)
	if len(byActivity.Groups) != 2 || byActivity.Groups[0].Label != "No activity type" {
		t.Fatalf("activity groups = %+v", byActivity.Groups)
	}
	if byActivity.Groups[0].DurationMs != (2*time.Hour + time.Hour).Milliseconds() {
		t.Fatalf("untyped duration = %d", byActivity.Groups[0].DurationMs)
	}
	if byActivity.Groups[1].Label != "coding" || byActivity.Groups[1].DurationMs != (90*time.Minute).Milliseconds() {
		t.Fatalf("coding group = %+v", byActivity.Groups[1])
	}
}

func TestParseInclusiveDate(t *testing.T) {
	from, err := parseBound("2026-09-25", false)
	if err != nil {
		t.Fatal(err)
	}
	to, err := parseBound("2026-09-25", true)
	if err != nil {
		t.Fatal(err)
	}
	if !to.Equal(from.AddDate(0, 0, 1)) {
		t.Fatalf("inclusive end = %s", to)
	}
	instant, err := parseBound("2026-09-25T15:04:05Z", true)
	if err != nil {
		t.Fatal(err)
	}
	if instant.Hour() != 15 {
		t.Fatalf("rfc3339 end was shifted: %s", instant)
	}
}

func ptrTime(t time.Time) *time.Time { return &t }
