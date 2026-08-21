package domain

import (
	"slices"
	"strings"
	"time"
)

const UntitledNote = "Untitled session"

const (
	StatusActive  = "active"
	StatusPaused  = "paused"
	StatusStopped = "stopped"
)

var SessionStatuses = []string{StatusActive, StatusPaused, StatusStopped}

// Session is the server-side time session (not the wire DTO).
type Session struct {
	ID               string
	ProjectID        string
	Note             string
	TicketID         *string
	ActivityTypeID   *string
	Tags             []string
	Status           string
	StartedAt        time.Time
	EndedAt          *time.Time
	PausedMs         int64
	PausedAt         *time.Time
	TargetDurationMs *int64
}

func NormalizeNote(note string) string {
	n := strings.TrimSpace(note)
	if n == "" {
		return UntitledNote
	}
	return n
}

func NormalizeOptionalString(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

func NormalizeTags(tags []string) []string {
	if tags == nil {
		return []string{}
	}
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func NormalizeTargetDurationMs(v *int64) (*int64, error) {
	if v == nil {
		return nil, nil
	}
	if *v < 0 {
		return nil, ErrInvalidBody("targetDurationMs must be >= 0.")
	}
	return v, nil
}

func IsLiveStatus(status string) bool {
	return status == StatusActive || status == StatusPaused
}

func ValidStatusFilter(s string) bool {
	return slices.Contains(SessionStatuses, s)
}

// StartSession builds a new active session at now.
func StartSession(id, projectID, note string, ticketID, activityTypeID *string, tags []string, target *int64, now time.Time) Session {
	return Session{
		ID:               id,
		ProjectID:        projectID,
		Note:             NormalizeNote(note),
		TicketID:         NormalizeOptionalString(ticketID),
		ActivityTypeID:   activityTypeID,
		Tags:             NormalizeTags(tags),
		Status:           StatusActive,
		StartedAt:        now.UTC(),
		EndedAt:          nil,
		PausedMs:         0,
		PausedAt:         nil,
		TargetDurationMs: target,
	}
}

func Pause(s Session, now time.Time) (Session, error) {
	if s.Status != StatusActive {
		return Session{}, ErrInvalidTransition()
	}
	t := now.UTC()
	s.Status = StatusPaused
	s.PausedAt = &t
	return s, nil
}

func Resume(s Session, now time.Time) (Session, error) {
	if s.Status != StatusPaused {
		return Session{}, ErrInvalidTransition()
	}
	s.PausedMs = foldPause(s, now)
	s.PausedAt = nil
	s.Status = StatusActive
	return s, nil
}

func Stop(s Session, now time.Time) (Session, error) {
	switch s.Status {
	case StatusPaused:
		s.PausedMs = foldPause(s, now)
		s.PausedAt = nil
	case StatusActive:
		// keep pausedMs
	default:
		return Session{}, ErrInvalidTransition()
	}
	t := now.UTC()
	s.Status = StatusStopped
	s.EndedAt = &t
	return s, nil
}

func foldPause(s Session, now time.Time) int64 {
	if s.PausedAt == nil {
		return s.PausedMs
	}
	delta := now.UTC().Sub(s.PausedAt.UTC()).Milliseconds()
	if delta < 0 {
		delta = 0
	}
	return s.PausedMs + delta
}

// SessionPatch is a partial update. Unset pointer / Set=false means leave unchanged.
type SessionPatch struct {
	ProjectID        *string
	Note             *string
	TicketID         *string
	TicketSet        bool
	ActivityTypeID   *string
	ActivityTypeSet  bool
	Tags             *[]string
	StartedAt        *time.Time
	EndedAt          *time.Time
	EndedSet         bool
	PausedMs         *int64
	TargetDurationMs *int64
	TargetSet        bool
}

func ParseISOTime(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(s))
	if err != nil {
		return time.Time{}, ErrInvalidBody("must be an ISO-8601 timestamp.")
	}
	return t.UTC(), nil
}

func ManualSession(id, projectID, note string, ticketID, activityTypeID *string, tags []string, target *int64, startedAt, endedAt time.Time, pausedMs int64) (Session, error) {
	end := endedAt.UTC()
	s := Session{
		ID:               id,
		ProjectID:        projectID,
		Note:             NormalizeNote(note),
		TicketID:         NormalizeOptionalString(ticketID),
		ActivityTypeID:   activityTypeID,
		Tags:             NormalizeTags(tags),
		Status:           StatusStopped,
		StartedAt:        startedAt.UTC(),
		EndedAt:          &end,
		PausedMs:         pausedMs,
		PausedAt:         nil,
		TargetDurationMs: target,
	}
	if err := validateSessionTimes(s, end); err != nil {
		return Session{}, err
	}
	return s, nil
}

func ApplySessionPatch(s Session, p SessionPatch, now time.Time) (Session, error) {
	if p.ProjectID != nil {
		id := strings.TrimSpace(*p.ProjectID)
		if id == "" {
			return Session{}, ErrInvalidBody("projectId is required.")
		}
		s.ProjectID = id
	}
	if p.Note != nil {
		s.Note = NormalizeNote(*p.Note)
	}
	if p.TicketSet {
		s.TicketID = NormalizeOptionalString(p.TicketID)
	}
	if p.ActivityTypeSet {
		s.ActivityTypeID = NormalizeOptionalString(p.ActivityTypeID)
	}
	if p.Tags != nil {
		s.Tags = NormalizeTags(*p.Tags)
	}
	if p.StartedAt != nil {
		s.StartedAt = p.StartedAt.UTC()
	}
	if p.EndedSet {
		if p.EndedAt == nil {
			s.EndedAt = nil
		} else {
			t := p.EndedAt.UTC()
			s.EndedAt = &t
		}
	}
	if p.PausedMs != nil {
		s.PausedMs = *p.PausedMs
	}
	if p.TargetSet {
		target, err := NormalizeTargetDurationMs(p.TargetDurationMs)
		if err != nil {
			return Session{}, err
		}
		s.TargetDurationMs = target
	}
	if err := validateSessionTimes(s, now); err != nil {
		return Session{}, err
	}
	return s, nil
}

func validateSessionTimes(s Session, now time.Time) error {
	if s.PausedMs < 0 {
		return ErrInvalidBody("pausedMs must be >= 0 and must not exceed the interval.")
	}
	started := s.StartedAt.UTC()
	switch s.Status {
	case StatusStopped:
		if s.EndedAt == nil {
			return ErrInvalidBody("endedAt is required on a stopped session.")
		}
		ended := s.EndedAt.UTC()
		if !ended.After(started) {
			return ErrInvalidBody("endedAt must be after startedAt.")
		}
		if s.PausedMs > ended.Sub(started).Milliseconds() {
			return ErrInvalidBody("pausedMs must be >= 0 and must not exceed the interval.")
		}
	case StatusActive:
		if s.EndedAt != nil {
			return ErrInvalidBody("endedAt is only set on stopped sessions; use POST .../stop.")
		}
		dur := now.UTC().Sub(started).Milliseconds()
		if dur < 0 {
			dur = 0
		}
		if s.PausedMs > dur {
			return ErrInvalidBody("pausedMs must be >= 0 and must not exceed the interval.")
		}
	case StatusPaused:
		if s.EndedAt != nil {
			return ErrInvalidBody("endedAt is only set on stopped sessions; use POST .../stop.")
		}
		if s.PausedAt == nil {
			return ErrInvalidBody("pausedAt is required while paused.")
		}
		pausedAt := s.PausedAt.UTC()
		if pausedAt.Before(started) {
			return ErrInvalidBody("startedAt must be at or before pausedAt.")
		}
		if s.PausedMs > pausedAt.Sub(started).Milliseconds() {
			return ErrInvalidBody("pausedMs must be >= 0 and must not exceed the interval.")
		}
	default:
		return ErrInvalidBody("unknown session status.")
	}
	return nil
}
