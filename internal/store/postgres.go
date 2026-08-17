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

func (p *Postgres) ListSessions(ctx context.Context, userID uuid.UUID, statuses []string, limit int) ([]domain.Session, error) {
	want := map[string]bool{}
	for _, s := range statuses {
		want[s] = true
	}
	rows, err := p.q.ListSessions(ctx, sqlcgen.ListSessionsParams{
		UserID:         userID,
		FilterStatuses: len(statuses) > 0,
		WantActive:     want[domain.StatusActive],
		WantPaused:     want[domain.StatusPaused],
		WantStopped:    want[domain.StatusStopped],
		Lim:            int32(limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Session, 0, len(rows))
	for _, r := range rows {
		out = append(out, sessionFromList(r))
	}
	return out, nil
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

func (p *Postgres) SeedEmpty(ctx context.Context, userID uuid.UUID, profile domain.Profile, project domain.Project) error {
	n, err := p.q.CountUsers(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	if err := p.q.InsertUser(ctx, userID); err != nil {
		return err
	}
	if err := p.q.InsertProfile(ctx, sqlcgen.InsertProfileParams{
		UserID:      userID,
		DisplayName: profile.DisplayName,
		Handle:      profile.Handle,
		AvatarUrl:   ptrNullString(profile.AvatarURL),
	}); err != nil {
		return err
	}
	_, err = p.CreateProject(ctx, userID, project)
	return err
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
		ActivityType:     ptrNullString(s.ActivityType),
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
	tags, err := encodeTags(s.Tags)
	if err != nil {
		return sqlcgen.UpdateSessionParams{}, err
	}
	return sqlcgen.UpdateSessionParams{
		UserID:           userID,
		ID:               id,
		Note:             s.Note,
		TicketID:         ptrNullString(s.TicketID),
		ActivityType:     ptrNullString(s.ActivityType),
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
		case "sessions_one_live_per_user":
			return domain.ErrSessionAlreadyActive()
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

func sessionFromRow(id, projectID uuid.UUID, note string, ticket, activity sql.NullString, tags []byte, status string, started time.Time, ended, pausedAt sql.NullTime, pausedMs int64, target sql.NullInt64) domain.Session {
	return domain.Session{
		ID:               id.String(),
		ProjectID:        projectID.String(),
		Note:             note,
		TicketID:         nullStringPtr(ticket),
		ActivityType:     nullStringPtr(activity),
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
	return sessionFromRow(r.ID, r.ProjectID, r.Note, r.TicketID, r.ActivityType, r.Tags, r.Status, r.StartedAt, r.EndedAt, r.PausedAt, r.PausedMs, r.TargetDurationMs)
}

func sessionFromList(r sqlcgen.ListSessionsRow) domain.Session {
	return sessionFromRow(r.ID, r.ProjectID, r.Note, r.TicketID, r.ActivityType, r.Tags, r.Status, r.StartedAt, r.EndedAt, r.PausedAt, r.PausedMs, r.TargetDurationMs)
}

func sessionFromLive(r sqlcgen.GetLiveSessionRow) domain.Session {
	return sessionFromRow(r.ID, r.ProjectID, r.Note, r.TicketID, r.ActivityType, r.Tags, r.Status, r.StartedAt, r.EndedAt, r.PausedAt, r.PausedMs, r.TargetDurationMs)
}

func sessionFromInsert(r sqlcgen.InsertSessionRow) domain.Session {
	return sessionFromRow(r.ID, r.ProjectID, r.Note, r.TicketID, r.ActivityType, r.Tags, r.Status, r.StartedAt, r.EndedAt, r.PausedAt, r.PausedMs, r.TargetDurationMs)
}

func sessionFromUpdate(r sqlcgen.UpdateSessionRow) domain.Session {
	return sessionFromRow(r.ID, r.ProjectID, r.Note, r.TicketID, r.ActivityType, r.Tags, r.Status, r.StartedAt, r.EndedAt, r.PausedAt, r.PausedMs, r.TargetDurationMs)
}
