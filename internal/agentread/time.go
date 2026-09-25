package agentread

import (
	"fmt"
	"strings"
	"time"

	"github.com/EmilM32/vynno-api/internal/store"
)

const isoMilli = "2006-01-02T15:04:05.000Z07:00"

func formatTime(t time.Time) string {
	return t.UTC().Format(isoMilli)
}

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := formatTime(*t)
	return &s
}

// parseBound accepts RFC3339 or a UTC calendar date. A date used as the end of a
// window is exclusive midnight of the next day, so YYYY-MM-DD includes that day.
func parseBound(raw string, inclusiveDate bool) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("time is required")
	}
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t.UTC(), nil
	}
	d, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("time %q must be RFC3339 or YYYY-MM-DD", raw)
	}
	d = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
	if inclusiveDate {
		return d.AddDate(0, 0, 1), nil
	}
	return d, nil
}

func parseOptionalBound(raw string, inclusiveDate bool) (*time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	t, err := parseBound(raw, inclusiveDate)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func sessionDuration(row store.AgentSession, now time.Time) int64 {
	end := now
	if row.EndedAt != nil {
		end = *row.EndedAt
	}
	if !end.After(row.StartedAt) {
		return 0
	}
	return end.Sub(row.StartedAt).Milliseconds()
}

// clipInterval returns the portion of a session inside [from, to). A live session
// runs until now. The boolean is false when the overlap is empty.
func clipInterval(started time.Time, ended *time.Time, from, to, now time.Time) (time.Time, time.Time, bool) {
	start := started.UTC()
	end := now.UTC()
	if ended != nil {
		end = ended.UTC()
	}
	if start.Before(from) {
		start = from.UTC()
	}
	if end.After(to) {
		end = to.UTC()
	}
	if !end.After(start) {
		return time.Time{}, time.Time{}, false
	}
	return start, end, true
}

type daySlice struct {
	day string
	ms  int64
}

// splitDays divides [start, end) into UTC calendar days.
func splitDays(start, end time.Time) []daySlice {
	cur := start.UTC()
	end = end.UTC()
	var out []daySlice
	for cur.Before(end) {
		next := time.Date(cur.Year(), cur.Month(), cur.Day()+1, 0, 0, 0, 0, time.UTC)
		sliceEnd := end
		if next.Before(sliceEnd) {
			sliceEnd = next
		}
		out = append(out, daySlice{
			day: cur.Format("2006-01-02"),
			ms:  sliceEnd.Sub(cur).Milliseconds(),
		})
		cur = sliceEnd
	}
	return out
}
