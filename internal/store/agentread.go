package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/EmilM32/vynno-api/internal/store/sqlcgen"
	"github.com/google/uuid"
)

const (
	// AgentSessionLimitMax is the largest page the read-only MCP may request.
	AgentSessionLimitMax = 200
	// AgentWindowCap is the most sessions a summary will load. The SQL limit is one higher
	// so a full page can be told apart from a truncated one.
	AgentWindowCap = 10000
)

// ErrSummaryWindowTooWide means the summary range matched more sessions than AgentWindowCap.
var ErrSummaryWindowTooWide = errors.New("more than 10000 sessions match this window; narrow from and to")

// AgentSessionFilter is a read of one account's sessions. User id is not a field:
// callers pass it separately and every query includes it.
type AgentSessionFilter struct {
	ProjectID *uuid.UUID
	Status    string
	From      *time.Time
	To        *time.Time
	Limit     int
}

// AgentSession is one session plus the names an agent needs. It has no auth columns.
type AgentSession struct {
	ID               string
	ProjectID        string
	ProjectName      string
	Note             string
	TicketID         *string
	ActivityTypeID   *string
	ActivityTypeName *string
	Status           string
	StartedAt        time.Time
	EndedAt          *time.Time
	TargetDurationMs *int64
}

func (p *Postgres) ListAgentProjects(ctx context.Context, userID uuid.UUID, includeArchived bool) ([]domain.Project, error) {
	rows, err := p.q.ListAgentProjects(ctx, sqlcgen.ListAgentProjectsParams{
		UserID:          userID,
		IncludeArchived: includeArchived,
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Project, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.Project{
			ID:              r.ID.String(),
			Name:            r.Name,
			Color:           r.Color,
			Code:            nullStringPtr(r.Code),
			ProgressPercent: nullInt32Ptr(r.ProgressPercent),
			Archived:        r.Archived,
		})
	}
	return out, nil
}

func (p *Postgres) ListAgentSessions(ctx context.Context, userID uuid.UUID, f AgentSessionFilter) ([]AgentSession, error) {
	if f.Limit < 1 || f.Limit > AgentSessionLimitMax {
		return nil, fmt.Errorf("session list limit must be 1..%d", AgentSessionLimitMax)
	}
	arg := sqlcgen.ListAgentSessionsParams{
		UserID:       userID,
		FilterStatus: f.Status != "",
		Status:       f.Status,
		FilterFrom:   f.From != nil,
		FilterTo:     f.To != nil,
		Lim:          int32(f.Limit),
	}
	if f.ProjectID != nil {
		arg.FilterProject = true
		arg.ProjectID = *f.ProjectID
	}
	if f.From != nil {
		arg.FromTs = *f.From
	}
	if f.To != nil {
		arg.ToTs = *f.To
	}
	rows, err := p.q.ListAgentSessions(ctx, arg)
	if err != nil {
		return nil, err
	}
	out := make([]AgentSession, 0, len(rows))
	for _, r := range rows {
		out = append(out, agentSessionFrom(r.ID, r.ProjectID, r.ProjectName, r.Note, r.TicketID, r.ActivityTypeID, r.ActivityTypeName, r.Status, r.StartedAt, r.EndedAt, r.TargetDurationMs))
	}
	return out, nil
}

func (p *Postgres) GetAgentLiveSession(ctx context.Context, userID uuid.UUID) (AgentSession, bool, error) {
	r, err := p.q.GetAgentLiveSession(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AgentSession{}, false, nil
		}
		return AgentSession{}, false, err
	}
	return agentSessionFrom(r.ID, r.ProjectID, r.ProjectName, r.Note, r.TicketID, r.ActivityTypeID, r.ActivityTypeName, r.Status, r.StartedAt, r.EndedAt, r.TargetDurationMs), true, nil
}

func (p *Postgres) ListAgentSessionsInWindow(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]AgentSession, error) {
	rows, err := p.q.ListAgentSessionsInWindow(ctx, sqlcgen.ListAgentSessionsInWindowParams{
		UserID: userID,
		FromTs: from,
		ToTs:   to,
	})
	if err != nil {
		return nil, err
	}
	if len(rows) > AgentWindowCap {
		return nil, ErrSummaryWindowTooWide
	}
	out := make([]AgentSession, 0, len(rows))
	for _, r := range rows {
		out = append(out, agentSessionFrom(r.ID, r.ProjectID, r.ProjectName, r.Note, r.TicketID, r.ActivityTypeID, r.ActivityTypeName, r.Status, r.StartedAt, r.EndedAt, r.TargetDurationMs))
	}
	return out, nil
}

func agentSessionFrom(id, projectID uuid.UUID, projectName, note string, ticket sql.NullString, activityID *uuid.UUID, activityName, status string, started time.Time, ended sql.NullTime, target sql.NullInt64) AgentSession {
	var activityTypeID *string
	if activityID != nil {
		s := activityID.String()
		activityTypeID = &s
	}
	var activityTypeName *string
	if activityName != "" {
		activityTypeName = &activityName
	}
	return AgentSession{
		ID:               id.String(),
		ProjectID:        projectID.String(),
		ProjectName:      projectName,
		Note:             note,
		TicketID:         nullStringPtr(ticket),
		ActivityTypeID:   activityTypeID,
		ActivityTypeName: activityTypeName,
		Status:           status,
		StartedAt:        started,
		EndedAt:          nullTimePtr(ended),
		TargetDurationMs: nullInt64Ptr(target),
	}
}
