package domain

import (
	"sort"
	"strings"
	"time"
)

// DayTotalsMaxDays caps one GET /stats/days request. A year heatmap (53 weeks)
// and a 366-day Insights range both fit.
const DayTotalsMaxDays = 400

const civilDateLayout = "2006-01-02"

// DayTotal is tracked time for one local date, project, and activity type.
type DayTotal struct {
	Date           string
	ProjectID      string
	ActivityTypeID *string
	DurationMs     int64
	SessionCount   int
}

// DayRange is an inclusive span of civil dates in one IANA time zone.
type DayRange struct {
	From     string
	To       string
	Location *time.Location
}

// ParseDayRange validates the GET /stats/days query. Every failure is invalid_query.
func ParseDayRange(from, to, timeZone string) (DayRange, error) {
	fromDay, err := time.Parse(civilDateLayout, from)
	if err != nil {
		return DayRange{}, ErrInvalidQuery("from must be a date (YYYY-MM-DD).")
	}
	toDay, err := time.Parse(civilDateLayout, to)
	if err != nil {
		return DayRange{}, ErrInvalidQuery("to must be a date (YYYY-MM-DD).")
	}
	if toDay.Before(fromDay) {
		return DayRange{}, ErrInvalidQuery("to must not be before from.")
	}
	if days := int(toDay.Sub(fromDay).Hours()/24) + 1; days > DayTotalsMaxDays {
		return DayRange{}, ErrInvalidQuery("The range must be at most 400 days.")
	}
	loc, err := loadTimeZone(timeZone)
	if err != nil {
		return DayRange{}, err
	}
	return DayRange{From: fromDay.Format(civilDateLayout), To: toDay.Format(civilDateLayout), Location: loc}, nil
}

// loadTimeZone accepts an IANA name. "Local" and paths would depend on the host.
func loadTimeZone(name string) (*time.Location, error) {
	if name == "" || name == "Local" || strings.HasPrefix(name, "/") || strings.Contains(name, "..") {
		return nil, ErrInvalidQuery("timeZone must be an IANA time zone name.")
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, ErrInvalidQuery("timeZone must be an IANA time zone name.")
	}
	return loc, nil
}

// QueryWindow is a UTC window that holds every instant whose local date is in the
// range. It is a day wider on each side, so DST and zones where midnight is
// skipped cannot drop a session; BucketDayTotals filters by date.
func (r DayRange) QueryWindow() (time.Time, time.Time) {
	from, _ := time.ParseInLocation(civilDateLayout, r.From, r.Location)
	to, _ := time.ParseInLocation(civilDateLayout, r.To, r.Location)
	return from.AddDate(0, 0, -1).UTC(), to.AddDate(0, 0, 2).UTC()
}

// BucketDayTotals sums stopped sessions by the local date of startedAt, then project,
// then activity type. A session counts in full on the day it started, like the SPA.
// Instants are cut to milliseconds first, as they are on the wire, so the totals
// match what the SPA would compute from the same sessions.
func BucketDayTotals(sessions []Session, r DayRange) []DayTotal {
	type key struct {
		date, project, activity string
		hasActivity             bool
	}
	totals := map[key]*DayTotal{}
	for _, s := range sessions {
		if s.Status != StatusStopped || s.EndedAt == nil {
			continue
		}
		started := s.StartedAt.Truncate(time.Millisecond)
		date := started.In(r.Location).Format(civilDateLayout)
		if date < r.From || date > r.To {
			continue
		}
		ms := s.EndedAt.Truncate(time.Millisecond).Sub(started).Milliseconds()
		if ms < 0 {
			ms = 0
		}
		k := key{date: date, project: s.ProjectID}
		if s.ActivityTypeID != nil {
			k.activity, k.hasActivity = *s.ActivityTypeID, true
		}
		t, ok := totals[k]
		if !ok {
			t = &DayTotal{Date: date, ProjectID: s.ProjectID}
			if k.hasActivity {
				id := k.activity
				t.ActivityTypeID = &id
			}
			totals[k] = t
		}
		t.DurationMs += ms
		t.SessionCount++
	}
	out := make([]DayTotal, 0, len(totals))
	for _, t := range totals {
		out = append(out, *t)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Date != b.Date {
			return a.Date < b.Date
		}
		if a.ProjectID != b.ProjectID {
			return a.ProjectID < b.ProjectID
		}
		if (a.ActivityTypeID == nil) != (b.ActivityTypeID == nil) {
			return a.ActivityTypeID == nil
		}
		return a.ActivityTypeID != nil && *a.ActivityTypeID < *b.ActivityTypeID
	})
	return out
}
