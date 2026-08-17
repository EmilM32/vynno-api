package store

import (
	"context"
	"encoding/json"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/google/uuid"
)

// Store is the persistence port. Postgres is the system of record; Memory is a test double.
type Store interface {
	GetProfile(ctx context.Context, userID uuid.UUID) (domain.Profile, error)

	ListProjects(ctx context.Context, userID uuid.UUID, includeArchived bool) ([]domain.Project, error)
	GetProject(ctx context.Context, userID, id uuid.UUID) (domain.Project, error)
	CreateProject(ctx context.Context, userID uuid.UUID, p domain.Project) (domain.Project, error)
	UpdateProject(ctx context.Context, userID uuid.UUID, p domain.Project) (domain.Project, error)
	DeleteProject(ctx context.Context, userID, id uuid.UUID) error
	CountActiveProjects(ctx context.Context, userID uuid.UUID) (int, error)
	CountProjectSessions(ctx context.Context, userID, projectID uuid.UUID) (int, error)
	CodeInUse(ctx context.Context, userID uuid.UUID, code string, excludeID uuid.UUID) (bool, error)

	ListSessions(ctx context.Context, userID uuid.UUID, statuses []string, limit int) ([]domain.Session, error)
	GetSession(ctx context.Context, userID, id uuid.UUID) (domain.Session, error)
	GetLiveSession(ctx context.Context, userID uuid.UUID) (domain.Session, bool, error)
	CreateSession(ctx context.Context, userID uuid.UUID, s domain.Session) (domain.Session, error)
	UpdateSession(ctx context.Context, userID uuid.UUID, s domain.Session) (domain.Session, error)

	FirstUserID(ctx context.Context) (uuid.UUID, bool, error)
	SeedEmpty(ctx context.Context, userID uuid.UUID, profile domain.Profile, project domain.Project) error
}

func encodeTags(tags []string) (json.RawMessage, error) {
	if tags == nil {
		tags = []string{}
	}
	return json.Marshal(tags)
}

func decodeTags(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var tags []string
	if err := json.Unmarshal(raw, &tags); err != nil || tags == nil {
		return []string{}
	}
	return tags
}
