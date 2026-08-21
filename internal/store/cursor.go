package store

import (
	"bytes"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/google/uuid"
)

const sessionCursorSep = "|"

// SessionPage is one newest-first slice of sessions plus an opaque next cursor.
type SessionPage struct {
	Items      []domain.Session
	NextCursor *string
}

func EncodeSessionCursor(startedAt time.Time, id uuid.UUID) string {
	raw := startedAt.UTC().Format(time.RFC3339Nano) + sessionCursorSep + id.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func DecodeSessionCursor(raw string) (time.Time, uuid.UUID, error) {
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	s := string(b)
	i := strings.LastIndex(s, sessionCursorSep)
	if i <= 0 || i == len(s)-1 {
		return time.Time{}, uuid.Nil, errInvalidCursor
	}
	startedAt, err := time.Parse(time.RFC3339Nano, s[:i])
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	id, err := uuid.Parse(s[i+1:])
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	return startedAt, id, nil
}

var errInvalidCursor = errors.New("cursor is not valid")

func paginateSessions(sorted []domain.Session, limit int, cursor string) (SessionPage, error) {
	out := sorted
	if cursor != "" {
		startedAt, id, err := DecodeSessionCursor(cursor)
		if err != nil {
			return SessionPage{}, domain.ErrInvalidQuery("cursor is not valid.")
		}
		start := -1
		for i, s := range out {
			if sessionAfterCursor(s, startedAt, id) {
				start = i
				break
			}
		}
		if start < 0 {
			return SessionPage{Items: []domain.Session{}}, nil
		}
		out = out[start:]
	}
	if limit < 1 {
		return SessionPage{Items: out}, nil
	}
	if len(out) > limit {
		last := out[limit-1]
		id, err := uuid.Parse(last.ID)
		if err != nil {
			return SessionPage{}, err
		}
		next := EncodeSessionCursor(last.StartedAt, id)
		return SessionPage{Items: out[:limit], NextCursor: &next}, nil
	}
	if out == nil {
		out = []domain.Session{}
	}
	return SessionPage{Items: out}, nil
}

// sessionAfterCursor reports whether s sorts after the cursor in
// started_at DESC, id DESC (i.e. older, or same instant with a smaller id).
func sessionAfterCursor(s domain.Session, startedAt time.Time, id uuid.UUID) bool {
	if s.StartedAt.Before(startedAt) {
		return true
	}
	if !s.StartedAt.Equal(startedAt) {
		return false
	}
	sid, err := uuid.Parse(s.ID)
	if err != nil {
		return s.ID < id.String()
	}
	return bytes.Compare(sid[:], id[:]) < 0
}
