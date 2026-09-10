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
)

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
	return status == StatusActive
}

func ValidStatusFilter(s string) bool {
	return slices.Contains(SessionStatuses, s)
}

// StartSession builds a new active session at now.
func StartSession(id, projectID, note string, ticketID, activityTypeID *string, target *int64, now time.Time) Session {
	return Session{
		ID:               id,
		ProjectID:        projectID,
		Note:             NormalizeNote(note),
		TicketID:         NormalizeOptionalString(ticketID),
		ActivityTypeID:   activityTypeID,
		Status:           StatusActive,
		StartedAt:        now.UTC(),
		EndedAt:          nil,
		TargetDurationMs: target,
	}
}

func Stop(s Session, now time.Time) (Session, error) {
	if s.Status != StatusActive {
		return Session{}, ErrInvalidTransition()
	}
	t := now.UTC()
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
	return t.UTC(), nil
}

func ManualSession(id, projectID, note string, ticketID, activityTypeID *string, target *int64, startedAt, endedAt time.Time) (Session, error) {
	end := endedAt.UTC()
	s := Session{
		ID:               id,
		ProjectID:        projectID,
		Note:             NormalizeNote(note),
		TicketID:         NormalizeOptionalString(ticketID),
		ActivityTypeID:   activityTypeID,
		Status:           StatusStopped,
		StartedAt:        startedAt.UTC(),
		EndedAt:          &end,
		TargetDurationMs: target,
	}
	if err := validateSessionTimes(s); err != nil {
		return Session{}, err
	}
	return s, nil
}

func ApplySessionPatch(s Session, p SessionPatch) (Session, error) {
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
	if p.TargetSet {
		target, err := NormalizeTargetDurationMs(p.TargetDurationMs)
		if err != nil {
			return Session{}, err
		}
		s.TargetDurationMs = target
	}
	if err := validateSessionTimes(s); err != nil {
		return Session{}, err
	}
	return s, nil
}

func validateSessionTimes(s Session) error {
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
	case StatusActive:
		if s.EndedAt != nil {
			return ErrInvalidBody("endedAt is only set on stopped sessions; use POST .../stop.")
		}
	default:
		return ErrInvalidBody("unknown session status.")
	}
	return nil
}
