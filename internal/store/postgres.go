package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/EmilM32/vynno-api/internal/store/sqlcgen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

// Postgres is the system of record.
type Postgres struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewPostgres(db *sql.DB) *Postgres {
	return &Postgres{db: db, q: sqlcgen.New(db)}
}

func (p *Postgres) GetProfile(ctx context.Context, userID uuid.UUID) (domain.Profile, error) {
	row, err := p.q.GetProfile(ctx, userID)
	if err != nil {
		return domain.Profile{}, mapNotFound(err)
	}
	return domain.Profile{
		DisplayName: row.DisplayName,
		Handle:      row.Handle,
		AvatarURL:   nullStringPtr(row.AvatarUrl),
	}, nil
}

func (p *Postgres) CreateProfile(ctx context.Context, userID uuid.UUID, profile domain.Profile) error {
	return p.q.InsertProfile(ctx, sqlcgen.InsertProfileParams{
		UserID:      userID,
		DisplayName: profile.DisplayName,
		Handle:      profile.Handle,
		AvatarUrl:   ptrNullString(profile.AvatarURL),
	})
}

func (p *Postgres) UpdateProfileDisplayName(ctx context.Context, userID uuid.UUID, displayName string) error {
	_, err := p.q.UpdateProfileDisplayName(ctx, sqlcgen.UpdateProfileDisplayNameParams{
		UserID:      userID,
		DisplayName: displayName,
	})
	return mapNotFound(err)
}

func (p *Postgres) ReplaceAvatar(ctx context.Context, userID uuid.UUID, av domain.Avatar) error {
	id, err := uuid.Parse(av.ID)
	if err != nil {
		return domain.ErrInvalidBody("invalid id")
	}
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := p.q.WithTx(tx)
	if err := q.DeleteAvatarByUser(ctx, userID); err != nil {
		return err
	}
	if err := q.InsertAvatar(ctx, sqlcgen.InsertAvatarParams{
		ID:          id,
		UserID:      userID,
		ContentType: av.ContentType,
		Bytes:       av.Bytes,
	}); err != nil {
		return err
	}
	path := domain.AvatarPath(id.String())
	if err := q.SetProfileAvatarURL(ctx, sqlcgen.SetProfileAvatarURLParams{
		UserID:    userID,
		AvatarUrl: ptrNullString(&path),
	}); err != nil {
		return err
	}
	return tx.Commit()
}

func (p *Postgres) DeleteAvatarByUser(ctx context.Context, userID uuid.UUID) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := p.q.WithTx(tx)
	if err := q.DeleteAvatarByUser(ctx, userID); err != nil {
		return err
	}
	if err := q.SetProfileAvatarURL(ctx, sqlcgen.SetProfileAvatarURLParams{
		UserID:    userID,
		AvatarUrl: sql.NullString{},
	}); err != nil {
		return err
	}
	return tx.Commit()
}

func (p *Postgres) GetAvatar(ctx context.Context, id uuid.UUID) (domain.Avatar, error) {
	row, err := p.q.GetAvatar(ctx, id)
	if err != nil {
		return domain.Avatar{}, mapNotFound(err)
	}
	return domain.Avatar{
		ID:          row.ID.String(),
		ContentType: row.ContentType,
		Bytes:       row.Bytes,
	}, nil
}

func (p *Postgres) GetAccountByUsername(ctx context.Context, username string) (Account, error) {
	row, err := p.q.GetUserByUsername(ctx, sql.NullString{String: username, Valid: true})
	if err != nil {
		return Account{}, mapNotFound(err)
	}
	return accountFromUser(row), nil
}

func (p *Postgres) GetAccountByID(ctx context.Context, id uuid.UUID) (Account, error) {
	row, err := p.q.GetUserByID(ctx, id)
	if err != nil {
		return Account{}, mapNotFound(err)
	}
	return accountFromUser(row), nil
}

func (p *Postgres) CreateAccount(ctx context.Context, a Account) error {
	err := p.q.InsertUser(ctx, sqlcgen.InsertUserParams{
		ID:           a.ID,
		Username:     sql.NullString{String: a.Username, Valid: a.Username != ""},
		PasswordHash: sql.NullString{String: a.PasswordHash, Valid: a.PasswordHash != ""},
	})
	return mapUnique(err)
}

func (p *Postgres) SetAccountCredentials(ctx context.Context, id uuid.UUID, username, passwordHash string) error {
	return mapUnique(p.q.SetUserCredentials(ctx, sqlcgen.SetUserCredentialsParams{
		ID:           id,
		Username:     sql.NullString{String: username, Valid: true},
		PasswordHash: sql.NullString{String: passwordHash, Valid: true},
	}))
}

func (p *Postgres) UsernameTaken(ctx context.Context, username string, excludeID uuid.UUID) (bool, error) {
	return p.q.UsernameInUse(ctx, sqlcgen.UsernameInUseParams{
		Username: sql.NullString{String: username, Valid: true},
		ID:       excludeID,
	})
}

func (p *Postgres) CreateToken(ctx context.Context, tok Token) error {
	return p.q.InsertAuthToken(ctx, sqlcgen.InsertAuthTokenParams{
		ID:        tok.ID,
		UserID:    tok.UserID,
		TokenHash: tok.TokenHash,
		ExpiresAt: tok.ExpiresAt,
	})
}

func (p *Postgres) GetTokenByHash(ctx context.Context, hash string) (Token, error) {
	row, err := p.q.GetAuthTokenByHash(ctx, hash)
	if err != nil {
		return Token{}, mapNotFound(err)
	}
	return Token{ID: row.ID, UserID: row.UserID, TokenHash: row.TokenHash, ExpiresAt: row.ExpiresAt}, nil
}

func (p *Postgres) DeleteTokenByHash(ctx context.Context, hash string) error {
	return p.q.DeleteAuthTokenByHash(ctx, hash)
}

func (p *Postgres) ListProjects(ctx context.Context, userID uuid.UUID, includeArchived bool) ([]domain.Project, error) {
	rows, err := p.q.ListProjects(ctx, sqlcgen.ListProjectsParams{
		UserID:          userID,
		IncludeArchived: includeArchived,
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Project, 0, len(rows))
	for _, r := range rows {
		out = append(out, projectFromList(r))
	}
	return out, nil
}

func (p *Postgres) GetProject(ctx context.Context, userID, id uuid.UUID) (domain.Project, error) {
	row, err := p.q.GetProject(ctx, sqlcgen.GetProjectParams{UserID: userID, ID: id})
	if err != nil {
		return domain.Project{}, mapNotFound(err)
	}
	return projectFromGet(row), nil
}

func (p *Postgres) CreateProject(ctx context.Context, userID uuid.UUID, proj domain.Project) (domain.Project, error) {
	id, err := uuid.Parse(proj.ID)
	if err != nil {
		return domain.Project{}, domain.ErrInvalidBody("invalid id")
	}
	row, err := p.q.InsertProject(ctx, sqlcgen.InsertProjectParams{
		ID:              id,
		UserID:          userID,
		Name:            proj.Name,
		Color:           proj.Color,
		Code:            ptrNullString(proj.Code),
		ProgressPercent: ptrNullInt32(proj.ProgressPercent),
		Archived:        proj.Archived,
	})
	if err != nil {
		return domain.Project{}, mapUnique(err)
	}
	return projectFromInsert(row), nil
}

func (p *Postgres) UpdateProject(ctx context.Context, userID uuid.UUID, proj domain.Project) (domain.Project, error) {
	id, err := uuid.Parse(proj.ID)
	if err != nil {
		return domain.Project{}, domain.ErrNotFound()
	}
	row, err := p.q.UpdateProject(ctx, sqlcgen.UpdateProjectParams{
		UserID:          userID,
		ID:              id,
		Name:            proj.Name,
		Color:           proj.Color,
		Code:            ptrNullString(proj.Code),
		ProgressPercent: ptrNullInt32(proj.ProgressPercent),
		Archived:        proj.Archived,
	})
	if err != nil {
		return domain.Project{}, mapUnique(mapNotFound(err))
	}
	return projectFromUpdate(row), nil
}

func (p *Postgres) DeleteProject(ctx context.Context, userID, id uuid.UUID) error {
	return p.q.DeleteProject(ctx, sqlcgen.DeleteProjectParams{UserID: userID, ID: id})
}

func (p *Postgres) CountActiveProjects(ctx context.Context, userID uuid.UUID) (int, error) {
	n, err := p.q.CountActiveProjects(ctx, userID)
	return int(n), err
}

func (p *Postgres) CountProjectSessions(ctx context.Context, userID, projectID uuid.UUID) (int, error) {
	n, err := p.q.CountProjectSessions(ctx, sqlcgen.CountProjectSessionsParams{
		UserID:    userID,
		ProjectID: projectID,
	})
	return int(n), err
}

func (p *Postgres) CodeInUse(ctx context.Context, userID uuid.UUID, code string, excludeID uuid.UUID) (bool, error) {
	return p.q.CodeInUse(ctx, sqlcgen.CodeInUseParams{
		UserID:    userID,
		Code:      code,
		ExcludeID: excludeID,
	})
}

func (p *Postgres) ListActivityTypes(ctx context.Context, userID uuid.UUID) ([]domain.ActivityType, error) {
	rows, err := p.q.ListActivityTypes(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.ActivityType, 0, len(rows))
	for _, r := range rows {
		out = append(out, activityTypeFromList(r))
	}
	return out, nil
}

func (p *Postgres) GetActivityType(ctx context.Context, userID, id uuid.UUID) (domain.ActivityType, error) {
	row, err := p.q.GetActivityType(ctx, sqlcgen.GetActivityTypeParams{UserID: userID, ID: id})
	if err != nil {
		return domain.ActivityType{}, mapNotFound(err)
	}
	return activityTypeFromGet(row), nil
}

func (p *Postgres) CreateActivityType(ctx context.Context, userID uuid.UUID, a domain.ActivityType) (domain.ActivityType, error) {
	id, err := uuid.Parse(a.ID)
	if err != nil {
		return domain.ActivityType{}, domain.ErrInvalidBody("invalid id")
	}
	row, err := p.q.InsertActivityType(ctx, sqlcgen.InsertActivityTypeParams{
		ID:     id,
		UserID: userID,
		Name:   a.Name,
		Color:  a.Color,
	})
	if err != nil {
		return domain.ActivityType{}, mapUnique(err)
	}
	return activityTypeFromInsert(row), nil
}

func (p *Postgres) UpdateActivityType(ctx context.Context, userID uuid.UUID, a domain.ActivityType) (domain.ActivityType, error) {
	id, err := uuid.Parse(a.ID)
	if err != nil {
		return domain.ActivityType{}, domain.ErrNotFound()
	}
	row, err := p.q.UpdateActivityType(ctx, sqlcgen.UpdateActivityTypeParams{
		UserID: userID,
		ID:     id,
		Name:   a.Name,
		Color:  a.Color,
	})
	if err != nil {
		return domain.ActivityType{}, mapUnique(mapNotFound(err))
	}
	return activityTypeFromUpdate(row), nil
}

func (p *Postgres) DeleteActivityType(ctx context.Context, userID, id uuid.UUID) error {
	return p.q.DeleteActivityType(ctx, sqlcgen.DeleteActivityTypeParams{UserID: userID, ID: id})
}

func (p *Postgres) CountActivityTypeSessions(ctx context.Context, userID, activityTypeID uuid.UUID) (int, error) {
	n, err := p.q.CountActivityTypeSessions(ctx, sqlcgen.CountActivityTypeSessionsParams{
		UserID:         userID,
		ActivityTypeID: &activityTypeID,
	})
	return int(n), err
}

func (p *Postgres) ActivityTypeNameInUse(ctx context.Context, userID uuid.UUID, name string, excludeID uuid.UUID) (bool, error) {
	return p.q.ActivityTypeNameInUse(ctx, sqlcgen.ActivityTypeNameInUseParams{
		UserID:    userID,
		Name:      name,
		ExcludeID: excludeID,
	})
}

func (p *Postgres) ListSessions(ctx context.Context, userID uuid.UUID, statuses []string, limit int, cursor string) (SessionPage, error) {
	want := map[string]bool{}
	for _, s := range statuses {
		want[s] = true
	}
	useCursor := cursor != ""
	cursorStarted := time.Time{}
	cursorID := uuid.Nil
	if useCursor {
		startedAt, id, err := DecodeSessionCursor(cursor)
		if err != nil {
			return SessionPage{}, domain.ErrInvalidQuery("cursor is not valid.")
		}
		cursorStarted = startedAt
		cursorID = id
	}
	fetch := limit
	if fetch < 1 {
		fetch = 1
	}
	rows, err := p.q.ListSessions(ctx, sqlcgen.ListSessionsParams{
		UserID:         userID,
		FilterStatuses: len(statuses) > 0,
		WantActive:     want[domain.StatusActive],
		WantPaused:     want[domain.StatusPaused],
		WantStopped:    want[domain.StatusStopped],
		UseCursor:      useCursor,
		CursorStarted:  cursorStarted,
		CursorID:       cursorID,
		Lim:            int32(fetch + 1),
	})
	if err != nil {
		return SessionPage{}, err
	}
	out := make([]domain.Session, 0, len(rows))
	for _, r := range rows {
		out = append(out, sessionFromList(r))
	}
	return paginateSessions(out, limit, "")
}

func (p *Postgres) GetSession(ctx context.Context, userID, id uuid.UUID) (domain.Session, error) {
	row, err := p.q.GetSession(ctx, sqlcgen.GetSessionParams{UserID: userID, ID: id})
	if err != nil {
		return domain.Session{}, mapNotFound(err)
	}
	return sessionFromGet(row), nil
}

func (p *Postgres) GetLiveSession(ctx context.Context, userID uuid.UUID) (domain.Session, bool, error) {
	row, err := p.q.GetLiveSession(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Session{}, false, nil
	}
	if err != nil {
		return domain.Session{}, false, err
	}
	return sessionFromLive(row), true, nil
}

func (p *Postgres) CreateSession(ctx context.Context, userID uuid.UUID, s domain.Session) (domain.Session, error) {
	params, err := insertSessionParams(userID, s)
	if err != nil {
		return domain.Session{}, err
	}
	row, err := p.q.InsertSession(ctx, params)
	if err != nil {
		return domain.Session{}, mapUnique(err)
	}
	return sessionFromInsert(row), nil
}

func (p *Postgres) UpdateSession(ctx context.Context, userID uuid.UUID, s domain.Session) (domain.Session, error) {
	params, err := updateSessionParams(userID, s)
	if err != nil {
		return domain.Session{}, err
	}
	row, err := p.q.UpdateSession(ctx, params)
	if err != nil {
		return domain.Session{}, mapNotFound(err)
	}
	return sessionFromUpdate(row), nil
}

func (p *Postgres) DeleteSession(ctx context.Context, userID, id uuid.UUID) error {
	return p.q.DeleteSession(ctx, sqlcgen.DeleteSessionParams{UserID: userID, ID: id})
}

func (p *Postgres) FirstUserID(ctx context.Context) (uuid.UUID, bool, error) {
	id, err := p.q.FirstUserID(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, err
	}
	return id, true, nil
}

func (p *Postgres) Bootstrap(ctx context.Context, userID uuid.UUID, username, passwordHash string, profile domain.Profile, project domain.Project) error {
	existing, err := p.q.GetUserByID(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		n, err := p.q.CountUsers(ctx)
		if err != nil {
			return err
		}
		if n > 0 {
			return nil
		}
		if err := p.CreateAccount(ctx, Account{ID: userID, Username: username, PasswordHash: passwordHash}); err != nil {
			return err
		}
		if err := p.CreateProfile(ctx, userID, profile); err != nil {
			return err
		}
		_, err = p.CreateProject(ctx, userID, project)
		return err
	}
	if err != nil {
		return err
	}
	if existing.PasswordHash.Valid && existing.PasswordHash.String != "" {
		return nil
	}
	return p.SetAccountCredentials(ctx, userID, username, passwordHash)
}

func accountFromUser(row sqlcgen.User) Account {
	return Account{
		ID:           row.ID,
		Username:     nullStringValue(row.Username),
		PasswordHash: nullStringValue(row.PasswordHash),
	}
}

func nullStringValue(s sql.NullString) string {
	if !s.Valid {
		return ""
	}
	return s.String
}

func insertSessionParams(userID uuid.UUID, s domain.Session) (sqlcgen.InsertSessionParams, error) {
	id, err := uuid.Parse(s.ID)
	if err != nil {
		return sqlcgen.InsertSessionParams{}, domain.ErrInvalidBody("invalid id")
	}
	pid, err := uuid.Parse(s.ProjectID)
	if err != nil {
		return sqlcgen.InsertSessionParams{}, domain.ErrInvalidBody("invalid projectId")
	}
	tags, err := encodeTags(s.Tags)
	if err != nil {
		return sqlcgen.InsertSessionParams{}, err
	}
	return sqlcgen.InsertSessionParams{
		ID:               id,
		UserID:           userID,
		ProjectID:        pid,
		Note:             s.Note,
		TicketID:         ptrNullString(s.TicketID),
		ActivityTypeID:   uuidPtrFromString(s.ActivityTypeID),
		Tags:             tags,
		Status:           s.Status,
		StartedAt:        s.StartedAt,
		EndedAt:          ptrNullTime(s.EndedAt),
		PausedMs:         s.PausedMs,
		PausedAt:         ptrNullTime(s.PausedAt),
		TargetDurationMs: ptrNullInt64(s.TargetDurationMs),
	}, nil
}

func updateSessionParams(userID uuid.UUID, s domain.Session) (sqlcgen.UpdateSessionParams, error) {
	id, err := uuid.Parse(s.ID)
	if err != nil {
		return sqlcgen.UpdateSessionParams{}, domain.ErrNotFound()
	}
	pid, err := uuid.Parse(s.ProjectID)
	if err != nil {
		return sqlcgen.UpdateSessionParams{}, domain.ErrInvalidBody("invalid projectId")
	}
	tags, err := encodeTags(s.Tags)
	if err != nil {
		return sqlcgen.UpdateSessionParams{}, err
	}
	return sqlcgen.UpdateSessionParams{
		UserID:           userID,
		ID:               id,
		ProjectID:        pid,
		Note:             s.Note,
		TicketID:         ptrNullString(s.TicketID),
		ActivityTypeID:   uuidPtrFromString(s.ActivityTypeID),
		Tags:             tags,
		Status:           s.Status,
		StartedAt:        s.StartedAt,
		EndedAt:          ptrNullTime(s.EndedAt),
		PausedMs:         s.PausedMs,
		PausedAt:         ptrNullTime(s.PausedAt),
		TargetDurationMs: ptrNullInt64(s.TargetDurationMs),
	}, nil
}

func mapNotFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound()
	}
	return err
}

func mapUnique(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case "projects_user_code_uidx":
			return domain.ErrCodeInUse()
		case "activity_types_user_name_uidx":
			return domain.ErrNameInUse()
		case "sessions_one_live_per_user":
			return domain.ErrSessionAlreadyActive()
		case "users_username_key":
			return domain.ErrUsernameInUse()
		}
		return domain.ErrCodeInUse()
	}
	return err
}

func ptrNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func ptrNullInt32(n *int) sql.NullInt32 {
	if n == nil {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: int32(*n), Valid: true}
}

func ptrNullInt64(n *int64) sql.NullInt64 {
	if n == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *n, Valid: true}
}

func ptrNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func nullStringPtr(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	v := s.String
	return &v
}

func nullInt32Ptr(n sql.NullInt32) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int32)
	return &v
}

func nullInt64Ptr(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	v := n.Int64
	return &v
}

func nullTimePtr(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}

func projectFromGet(r sqlcgen.GetProjectRow) domain.Project {
	return domain.Project{
		ID: r.ID.String(), Name: r.Name, Color: r.Color,
		Code: nullStringPtr(r.Code), ProgressPercent: nullInt32Ptr(r.ProgressPercent), Archived: r.Archived,
	}
}

func projectFromList(r sqlcgen.ListProjectsRow) domain.Project {
	return domain.Project{
		ID: r.ID.String(), Name: r.Name, Color: r.Color,
		Code: nullStringPtr(r.Code), ProgressPercent: nullInt32Ptr(r.ProgressPercent), Archived: r.Archived,
	}
}

func projectFromInsert(r sqlcgen.InsertProjectRow) domain.Project {
	return domain.Project{
		ID: r.ID.String(), Name: r.Name, Color: r.Color,
		Code: nullStringPtr(r.Code), ProgressPercent: nullInt32Ptr(r.ProgressPercent), Archived: r.Archived,
	}
}

func projectFromUpdate(r sqlcgen.UpdateProjectRow) domain.Project {
	return domain.Project{
		ID: r.ID.String(), Name: r.Name, Color: r.Color,
		Code: nullStringPtr(r.Code), ProgressPercent: nullInt32Ptr(r.ProgressPercent), Archived: r.Archived,
	}
}

func sessionFromRow(id, projectID uuid.UUID, note string, ticket sql.NullString, activityID *uuid.UUID, tags []byte, status string, started time.Time, ended, pausedAt sql.NullTime, pausedMs int64, target sql.NullInt64) domain.Session {
	return domain.Session{
		ID:               id.String(),
		ProjectID:        projectID.String(),
		Note:             note,
		TicketID:         nullStringPtr(ticket),
		ActivityTypeID:   uuidPtrString(activityID),
		Tags:             decodeTags(tags),
		Status:           status,
		StartedAt:        started,
		EndedAt:          nullTimePtr(ended),
		PausedMs:         pausedMs,
		PausedAt:         nullTimePtr(pausedAt),
		TargetDurationMs: nullInt64Ptr(target),
	}
}

func sessionFromGet(r sqlcgen.GetSessionRow) domain.Session {
	return sessionFromRow(r.ID, r.ProjectID, r.Note, r.TicketID, r.ActivityTypeID, r.Tags, r.Status, r.StartedAt, r.EndedAt, r.PausedAt, r.PausedMs, r.TargetDurationMs)
}

func sessionFromList(r sqlcgen.ListSessionsRow) domain.Session {
	return sessionFromRow(r.ID, r.ProjectID, r.Note, r.TicketID, r.ActivityTypeID, r.Tags, r.Status, r.StartedAt, r.EndedAt, r.PausedAt, r.PausedMs, r.TargetDurationMs)
}

func sessionFromLive(r sqlcgen.GetLiveSessionRow) domain.Session {
	return sessionFromRow(r.ID, r.ProjectID, r.Note, r.TicketID, r.ActivityTypeID, r.Tags, r.Status, r.StartedAt, r.EndedAt, r.PausedAt, r.PausedMs, r.TargetDurationMs)
}

func sessionFromInsert(r sqlcgen.InsertSessionRow) domain.Session {
	return sessionFromRow(r.ID, r.ProjectID, r.Note, r.TicketID, r.ActivityTypeID, r.Tags, r.Status, r.StartedAt, r.EndedAt, r.PausedAt, r.PausedMs, r.TargetDurationMs)
}

func sessionFromUpdate(r sqlcgen.UpdateSessionRow) domain.Session {
	return sessionFromRow(r.ID, r.ProjectID, r.Note, r.TicketID, r.ActivityTypeID, r.Tags, r.Status, r.StartedAt, r.EndedAt, r.PausedAt, r.PausedMs, r.TargetDurationMs)
}

func uuidPtrFromString(s *string) *uuid.UUID {
	if s == nil || *s == "" {
		return nil
	}
	id, err := uuid.Parse(*s)
	if err != nil {
		return nil
	}
	return &id
}

func uuidPtrString(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	v := id.String()
	return &v
}

func activityTypeFromGet(r sqlcgen.GetActivityTypeRow) domain.ActivityType {
	return domain.ActivityType{ID: r.ID.String(), Name: r.Name, Color: r.Color}
}

func activityTypeFromList(r sqlcgen.ListActivityTypesRow) domain.ActivityType {
	return domain.ActivityType{ID: r.ID.String(), Name: r.Name, Color: r.Color}
}

func activityTypeFromInsert(r sqlcgen.InsertActivityTypeRow) domain.ActivityType {
	return domain.ActivityType{ID: r.ID.String(), Name: r.Name, Color: r.Color}
}

func activityTypeFromUpdate(r sqlcgen.UpdateActivityTypeRow) domain.ActivityType {
	return domain.ActivityType{ID: r.ID.String(), Name: r.Name, Color: r.Color}
}
