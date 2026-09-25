package agentread

import (
	"context"
	"time"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/EmilM32/vynno-api/internal/store"
	"github.com/google/uuid"
)

// Reader loads one account at a time. userID is the authenticated account,
// never a value taken from tool arguments.
type Reader interface {
	GetProfile(ctx context.Context, userID uuid.UUID) (domain.Profile, error)
	ListAgentProjects(ctx context.Context, userID uuid.UUID, includeArchived bool) ([]domain.Project, error)
	ListAgentSessions(ctx context.Context, userID uuid.UUID, f store.AgentSessionFilter) ([]store.AgentSession, error)
	GetAgentLiveSession(ctx context.Context, userID uuid.UUID) (store.AgentSession, bool, error)
	ListAgentSessionsInWindow(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]store.AgentSession, error)
}
