package domain

import (
	"slices"
	"strings"
	"time"
)

const UntitledNote = "Untitled session"

const (
	DefaultSessionListLimit = 20
	MaxSessionListLimit     = 100
	// MaxTargetDurationMs is 2^53−1, the largest integer JSON can round-trip.
	MaxTargetDurationMs int64 = 9007199254740991
	MaxFutureSkew             = 5 * time.Minute
	MaxSessionDuration        = 7 * 24 * time.Hour
)

// MinSessionTime is the earliest accepted startedAt.
var MinSessionTime = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

const (
	StatusActive  = "active"
	StatusStopped = "stopped"
)

var SessionStatuses = []string{StatusActive, StatusStopped}

// Session is the server-side time session (not the wire DTO).
type Session struct {
	ID               string
	ProjectID        string
	Note             string
	TicketID         *string
	ActivityTypeID   *string
	Status           string
	StartedAt        time.Time
	EndedAt          *time.Time
	TargetDurationMs *int64
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

func NormalizeTargetDurationMs(v *int64) (*int64, error) {
	if v == nil {
		return nil, nil
	}
	if *v < 0 || *v > MaxTargetDurationMs {
		return nil, ErrInvalidBody("targetDurationMs must be an integer from 0 through 9007199254740991.")
	}
	return v, nil
}

func IsLiveStatus(status string) bool {
	return status == StatusActive
}

func ValidStatusFilter(s string) bool {
	return slices.Contains(SessionStatuses, s)
}

// StartSession builds a new active session at now, truncated to microseconds.
func StartSession(id, projectID, note string, ticketID, activityTypeID *string, target *int64, now time.Time) (Session, error) {
	n, err := NormalizeNote(note)
	if err != nil {
		return Session{}, err
	}
	ticket, err := NormalizeTicketID(ticketID)
	if err != nil {
		return Session{}, err
	}
	return Session{
		ID:               id,
		ProjectID:        projectID,
		Note:             n,
		TicketID:         ticket,
		ActivityTypeID:   activityTypeID,
		Status:           StatusActive,
		StartedAt:        truncateInstant(now),
		EndedAt:          nil,
		TargetDurationMs: target,
	}, nil
}

func Stop(s Session, now time.Time) (Session, error) {
	if s.Status != StatusActive {
		return Session{}, ErrInvalidTransition()
	}
	t := truncateInstant(now)
	s.Status = StatusStopped
	s.EndedAt = &t
	return s, nil
}

// SessionPatch is a partial update. Unset pointer / Set=false means leave unchanged.
type SessionPatch struct {
	ProjectID        *string
	Note             *string
	TicketID         *string
	TicketSet        bool
	ActivityTypeID   *string
	ActivityTypeSet  bool
	StartedAt        *time.Time
	EndedAt          *time.Time
	EndedSet         bool
	TargetDurationMs *int64
	TargetSet        bool
}

func ParseISOTime(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(s))
	if err != nil {
		return time.Time{}, ErrInvalidBody("must be an ISO-8601 timestamp.")
	}
	return truncateInstant(t), nil
}

func truncateInstant(t time.Time) time.Time {
	return t.UTC().Truncate(time.Microsecond)
}

func ManualSession(id, projectID, note string, ticketID, activityTypeID *string, target *int64, startedAt, endedAt, now time.Time) (Session, error) {
	n, err := NormalizeNote(note)
	if err != nil {
		return Session{}, err
	}
	ticket, err := NormalizeTicketID(ticketID)
	if err != nil {
		return Session{}, err
	}
	end := truncateInstant(endedAt)
	s := Session{
		ID:               id,
		ProjectID:        projectID,
		Note:             n,
		TicketID:         ticket,
		ActivityTypeID:   activityTypeID,
		Status:           StatusStopped,
		StartedAt:        truncateInstant(startedAt),
		EndedAt:          &end,
		TargetDurationMs: target,
	}
	if err := validateSessionTimes(s); err != nil {
		return Session{}, err
	}
	if err := validateSessionBounds(s, now); err != nil {
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
		note, err := NormalizeNote(*p.Note)
		if err != nil {
			return Session{}, err
		}
		s.Note = note
	}
	if p.TicketSet {
		ticket, err := NormalizeTicketID(p.TicketID)
		if err != nil {
			return Session{}, err
		}
		s.TicketID = ticket
	}
	if p.ActivityTypeSet {
		s.ActivityTypeID = NormalizeOptionalString(p.ActivityTypeID)
	}
	timesSet := p.StartedAt != nil || p.EndedSet
	if p.StartedAt != nil {
		s.StartedAt = truncateInstant(*p.StartedAt)
	}
	if p.EndedSet {
		if p.EndedAt == nil {
			s.EndedAt = nil
		} else {
			t := truncateInstant(*p.EndedAt)
			s.EndedAt = &t
		}
	}
	if p.TargetSet {
		target, err := NormalizeTargetDurationMs(p.TargetDurationMs)
		if err != nil {
			return Session{}, err
		}
		s.TargetDurationMs = target
	}
	// Omitting both instants does not re-check bounds. A grandfathered row
	// (year 0001, or instants that already break the structural rule) can
	// still change note and other non-time fields. When the stored instants
	// already satisfy the structural rule, leaving them unchanged keeps it.
	if timesSet {
		if err := validateSessionTimes(s); err != nil {
			return Session{}, err
		}
		if err := validateSessionBounds(s, now); err != nil {
			return Session{}, err
		}
		return s, nil
	}
	if err := validateSessionTimes(s); err != nil {
		return s, nil
	}
	return s, nil
}

func validateSessionTimes(s Session) error {
	started := truncateInstant(s.StartedAt)
	switch s.Status {
	case StatusStopped:
		if s.EndedAt == nil {
			return ErrInvalidBody("endedAt is required on a stopped session.")
		}
		ended := truncateInstant(*s.EndedAt)
		if !ended.After(started) {
			return ErrInvalidBody("endedAt must be after startedAt.")
		}
	case StatusActive:
		if s.EndedAt != nil {
			return ErrInvalidBody("endedAt is only set on stopped sessions; use POST .../stop.")
		}
	default:
		return ErrInvalidBody("unknown session status.")
	}
	return nil
}

func validateSessionBounds(s Session, now time.Time) error {
	now = truncateInstant(now)
	started := truncateInstant(s.StartedAt)
	if started.Before(MinSessionTime) {
		return ErrInvalidBody("startedAt must be on or after 2000-01-01T00:00:00Z.")
	}
	latest := now.Add(MaxFutureSkew)
	if started.After(latest) {
		return ErrInvalidBody("startedAt is too far in the future.")
	}
	if s.EndedAt != nil {
		ended := truncateInstant(*s.EndedAt)
		if ended.After(latest) {
			return ErrInvalidBody("endedAt is too far in the future.")
		}
		if ended.Sub(started) > MaxSessionDuration {
			return ErrInvalidBody("Session duration must be at most 7 days.")
		}
	}
	return nil
}
