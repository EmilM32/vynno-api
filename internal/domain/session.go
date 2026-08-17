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

var ActivityTypes = []string{
	"deep_work",
	"meeting",
	"maintenance",
	"coding",
	"debugging",
	"docs",
	"research",
	"other",
}

var SessionStatuses = []string{StatusActive, StatusPaused, StatusStopped}

// Session is the server-side time session (not the wire DTO).
type Session struct {
	ID               string
	ProjectID        string
	Note             string
	TicketID         *string
	ActivityType     *string
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

func NormalizeActivityType(v *string) (*string, error) {
	if v == nil {
		return nil, nil
	}
	t := strings.TrimSpace(*v)
	if t == "" {
		return nil, nil
	}
	if !slices.Contains(ActivityTypes, t) {
		return nil, ErrInvalidBody("activityType is not a known value.")
	}
	return &t, nil
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
func StartSession(id, projectID, note string, ticketID, activityType *string, tags []string, target *int64, now time.Time) Session {
	return Session{
		ID:               id,
		ProjectID:        projectID,
		Note:             NormalizeNote(note),
		TicketID:         NormalizeOptionalString(ticketID),
		ActivityType:     activityType,
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
