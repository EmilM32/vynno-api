package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/google/uuid"
)

// Account is a login identity. Username and hash are never on the wire.
type Account struct {
	ID           uuid.UUID
	Username     string
	PasswordHash string
}

// Token is a hashed session. The raw secret is not stored.
type Token struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
}

// Store is the persistence port. Postgres is the system of record; Memory is a test double.
type Store interface {
	GetProfile(ctx context.Context, userID uuid.UUID) (domain.Profile, error)
	CreateProfile(ctx context.Context, userID uuid.UUID, p domain.Profile) error
	UpdateProfileDisplayName(ctx context.Context, userID uuid.UUID, displayName string) error
	ReplaceAvatar(ctx context.Context, userID uuid.UUID, a domain.Avatar) error
	DeleteAvatarByUser(ctx context.Context, userID uuid.UUID) error
	GetAvatar(ctx context.Context, id uuid.UUID) (domain.Avatar, error)

	GetAccountByUsername(ctx context.Context, username string) (Account, error)
	GetAccountByID(ctx context.Context, id uuid.UUID) (Account, error)
	CreateAccount(ctx context.Context, a Account) error
	SetAccountCredentials(ctx context.Context, id uuid.UUID, username, passwordHash string) error
	UsernameTaken(ctx context.Context, username string, excludeID uuid.UUID) (bool, error)

	CreateToken(ctx context.Context, tok Token) error
	GetTokenByHash(ctx context.Context, hash string) (Token, error)
	DeleteTokenByHash(ctx context.Context, hash string) error

	ListProjects(ctx context.Context, userID uuid.UUID, includeArchived bool) ([]domain.Project, error)
	GetProject(ctx context.Context, userID, id uuid.UUID) (domain.Project, error)
	CreateProject(ctx context.Context, userID uuid.UUID, p domain.Project) (domain.Project, error)
	UpdateProject(ctx context.Context, userID uuid.UUID, p domain.Project) (domain.Project, error)
	DeleteProject(ctx context.Context, userID, id uuid.UUID) error
	CountActiveProjects(ctx context.Context, userID uuid.UUID) (int, error)
	CountProjectSessions(ctx context.Context, userID, projectID uuid.UUID) (int, error)
	CodeInUse(ctx context.Context, userID uuid.UUID, code string, excludeID uuid.UUID) (bool, error)

	ListActivityTypes(ctx context.Context, userID uuid.UUID) ([]domain.ActivityType, error)
	GetActivityType(ctx context.Context, userID, id uuid.UUID) (domain.ActivityType, error)
	CreateActivityType(ctx context.Context, userID uuid.UUID, a domain.ActivityType) (domain.ActivityType, error)
	UpdateActivityType(ctx context.Context, userID uuid.UUID, a domain.ActivityType) (domain.ActivityType, error)
	DeleteActivityType(ctx context.Context, userID, id uuid.UUID) error
	CountActivityTypeSessions(ctx context.Context, userID, activityTypeID uuid.UUID) (int, error)
	ActivityTypeNameInUse(ctx context.Context, userID uuid.UUID, name string, excludeID uuid.UUID) (bool, error)

	ListSessions(ctx context.Context, userID uuid.UUID, statuses []string, limit int, cursor string) (SessionPage, error)
	GetSession(ctx context.Context, userID, id uuid.UUID) (domain.Session, error)
	GetLiveSession(ctx context.Context, userID uuid.UUID) (domain.Session, bool, error)
	CreateSession(ctx context.Context, userID uuid.UUID, s domain.Session) (domain.Session, error)
	UpdateSession(ctx context.Context, userID uuid.UUID, s domain.Session) (domain.Session, error)
	DeleteSession(ctx context.Context, userID, id uuid.UUID) error

	FirstUserID(ctx context.Context) (uuid.UUID, bool, error)
	Bootstrap(ctx context.Context, userID uuid.UUID, username, passwordHash string, profile domain.Profile, project domain.Project) error
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
