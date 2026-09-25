package agentread

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/EmilM32/vynno-api/internal/store"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	groupProject  = "project"
	groupActivity = "activity"
	groupDay      = "day"
)

// errUnauthorized is what a tool returns when the process token does not name a live session.
var errUnauthorized = errors.New("authentication required: VYNNO_MCP_TOKEN is missing, expired, or revoked. Issue one with `go run ./cmd/mcp token` and export it as VYNNO_MCP_TOKEN")

// Tools serves the vynno_* MCP tools for a single authenticated account.
type Tools struct {
	Read    Reader
	Resolve func(context.Context) (uuid.UUID, error)
	Now     func() time.Time
}

// Register adds the read-only tools. Each one resolves the session token again
// and passes that user id into Reader. Tool arguments cannot name another account.
func Register(server *mcp.Server, tools *Tools) {
	ann := readOnly()
	mcp.AddTool(server, &mcp.Tool{
		Name:        "vynno_whoami",
		Description: "Email and display name of the account authenticated by VYNNO_MCP_TOKEN. Every other tool returns only this account's history.",
		Annotations: ann,
	}, tools.whoami)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "vynno_list_projects",
		Description: "Projects owned by the authenticated account. includeArchived defaults to false. Archived projects are omitted unless it is true.",
		Annotations: ann,
	}, tools.listProjects)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "vynno_list_sessions",
		Description: "Sessions for the authenticated account, newest first. Optional overlap window (from, to), projectId, and status (active or stopped). limit defaults to 50 and is capped at 200. durationMs is the session's own length; a live session runs until now.",
		Annotations: ann,
	}, tools.listSessions)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "vynno_get_active_session",
		Description: "The live timer for the authenticated account. active is false when nothing is running.",
		Annotations: ann,
	}, tools.activeSession)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "vynno_summarize_time",
		Description: "Focus time for the authenticated account in a window. from and to are RFC3339, or YYYY-MM-DD in UTC (a date used as to includes that whole day). groupBy is project (default), activity, or day. Duration counts only the overlap with the window, so a timer that crosses midnight is split.",
		Annotations: ann,
	}, tools.summarize)
}

func readOnly() *mcp.ToolAnnotations {
	closed := false
	return &mcp.ToolAnnotations{
		ReadOnlyHint:    true,
		DestructiveHint: &closed,
		IdempotentHint:  true,
		OpenWorldHint:   &closed,
	}
}

func (t *Tools) now() time.Time {
	if t != nil && t.Now != nil {
		return t.Now().UTC()
	}
	return time.Now().UTC()
}

func (t *Tools) userID(ctx context.Context) (uuid.UUID, error) {
	if t == nil || t.Resolve == nil {
		return uuid.Nil, errUnauthorized
	}
	id, err := t.Resolve(ctx)
	if err != nil {
		if de, ok := domain.AsError(err); ok && (de.Code == domain.CodeUnauthorized || de.Code == domain.CodeNotFound) {
			return uuid.Nil, errUnauthorized
		}
		return uuid.Nil, err
	}
	if id == uuid.Nil {
		return uuid.Nil, errUnauthorized
	}
	return id, nil
}

type WhoAmI struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
}

func (t *Tools) whoami(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, WhoAmI, error) {
	id, err := t.userID(ctx)
	if err != nil {
		return nil, WhoAmI{}, err
	}
	profile, err := t.Read.GetProfile(ctx, id)
	if err != nil {
		return nil, WhoAmI{}, err
	}
	return nil, WhoAmI{Email: profile.Email, DisplayName: profile.DisplayName}, nil
}

type ListProjectsInput struct {
	IncludeArchived bool `json:"includeArchived,omitempty" jsonschema:"When true, include archived projects. Default false."`
}

type Project struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Color           string  `json:"color"`
	Code            *string `json:"code"`
	ProgressPercent *int    `json:"progressPercent"`
	Archived        bool    `json:"archived"`
}

type ProjectList struct {
	Projects []Project `json:"projects"`
}

func (t *Tools) listProjects(ctx context.Context, _ *mcp.CallToolRequest, in ListProjectsInput) (*mcp.CallToolResult, ProjectList, error) {
	id, err := t.userID(ctx)
	if err != nil {
		return nil, ProjectList{}, err
	}
	rows, err := t.Read.ListAgentProjects(ctx, id, in.IncludeArchived)
	if err != nil {
		return nil, ProjectList{}, err
	}
	out := make([]Project, 0, len(rows))
	for _, p := range rows {
		out = append(out, Project{
			ID:              p.ID,
			Name:            p.Name,
			Color:           p.Color,
			Code:            p.Code,
			ProgressPercent: p.ProgressPercent,
			Archived:        p.Archived,
		})
	}
	return nil, ProjectList{Projects: out}, nil
}

type ListSessionsInput struct {
	ProjectID string `json:"projectId,omitempty" jsonschema:"Project UUID owned by this account. Omit for every project."`
	From      string `json:"from,omitempty" jsonschema:"Overlap range start. RFC3339, or YYYY-MM-DD as 00:00:00Z."`
	To        string `json:"to,omitempty" jsonschema:"Overlap range end. RFC3339, or YYYY-MM-DD including that UTC day."`
	Status    string `json:"status,omitempty" jsonschema:"active or stopped. Omit for both."`
	Limit     int    `json:"limit,omitempty" jsonschema:"Page size. Default 50. Maximum 200."`
}

type Session struct {
	ID               string  `json:"id"`
	ProjectID        string  `json:"projectId"`
	ProjectName      string  `json:"projectName"`
	Note             string  `json:"note"`
	TicketID         *string `json:"ticketId"`
	ActivityTypeID   *string `json:"activityTypeId"`
	ActivityTypeName *string `json:"activityTypeName"`
	Status           string  `json:"status"`
	StartedAt        string  `json:"startedAt"`
	EndedAt          *string `json:"endedAt"`
	TargetDurationMs *int64  `json:"targetDurationMs"`
	DurationMs       int64   `json:"durationMs"`
}

type SessionList struct {
	Sessions []Session `json:"sessions"`
	Limit    int       `json:"limit"`
}

func (t *Tools) listSessions(ctx context.Context, _ *mcp.CallToolRequest, in ListSessionsInput) (*mcp.CallToolResult, SessionList, error) {
	id, err := t.userID(ctx)
	if err != nil {
		return nil, SessionList{}, err
	}
	filter, limit, err := sessionFilter(in)
	if err != nil {
		return nil, SessionList{}, err
	}
	rows, err := t.Read.ListAgentSessions(ctx, id, filter)
	if err != nil {
		return nil, SessionList{}, err
	}
	if len(rows) > limit {
		rows = rows[:limit]
	}
	now := t.now()
	out := make([]Session, 0, len(rows))
	for _, row := range rows {
		out = append(out, toSession(row, now))
	}
	return nil, SessionList{Sessions: out, Limit: limit}, nil
}

func sessionFilter(in ListSessionsInput) (store.AgentSessionFilter, int, error) {
	limit := in.Limit
	if limit == 0 {
		limit = 50
	}
	if limit < 1 {
		return store.AgentSessionFilter{}, 0, fmt.Errorf("limit must be between 1 and %d", store.AgentSessionLimitMax)
	}
	if limit > store.AgentSessionLimitMax {
		limit = store.AgentSessionLimitMax
	}
	status := strings.TrimSpace(in.Status)
	if status != "" && !domain.ValidStatusFilter(status) {
		return store.AgentSessionFilter{}, 0, fmt.Errorf("status must be active or stopped")
	}
	projectID, err := parseProjectID(in.ProjectID)
	if err != nil {
		return store.AgentSessionFilter{}, 0, err
	}
	from, err := parseOptionalBound(in.From, false)
	if err != nil {
		return store.AgentSessionFilter{}, 0, fmt.Errorf("from: %w", err)
	}
	to, err := parseOptionalBound(in.To, true)
	if err != nil {
		return store.AgentSessionFilter{}, 0, fmt.Errorf("to: %w", err)
	}
	if from != nil && to != nil && !to.After(*from) {
		return store.AgentSessionFilter{}, 0, fmt.Errorf("to must be after from")
	}
	return store.AgentSessionFilter{
		ProjectID: projectID,
		Status:    status,
		From:      from,
		To:        to,
		Limit:     limit,
	}, limit, nil
}

func parseProjectID(raw string) (*uuid.UUID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("projectId must be a UUID")
	}
	return &id, nil
}

func toSession(row store.AgentSession, now time.Time) Session {
	return Session{
		ID:               row.ID,
		ProjectID:        row.ProjectID,
		ProjectName:      row.ProjectName,
		Note:             row.Note,
		TicketID:         row.TicketID,
		ActivityTypeID:   row.ActivityTypeID,
		ActivityTypeName: row.ActivityTypeName,
		Status:           row.Status,
		StartedAt:        formatTime(row.StartedAt),
		EndedAt:          formatTimePtr(row.EndedAt),
		TargetDurationMs: row.TargetDurationMs,
		DurationMs:       sessionDuration(row, now),
	}
}

type ActiveSession struct {
	Active  bool     `json:"active"`
	Session *Session `json:"session"`
}

func (t *Tools) activeSession(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, ActiveSession, error) {
	id, err := t.userID(ctx)
	if err != nil {
		return nil, ActiveSession{}, err
	}
	row, ok, err := t.Read.GetAgentLiveSession(ctx, id)
	if err != nil {
		return nil, ActiveSession{}, err
	}
	if !ok {
		return nil, ActiveSession{Active: false}, nil
	}
	s := toSession(row, t.now())
	return nil, ActiveSession{Active: true, Session: &s}, nil
}

type SummarizeInput struct {
	From    string `json:"from" jsonschema:"Window start. RFC3339, or YYYY-MM-DD as 00:00:00Z. Required."`
	To      string `json:"to" jsonschema:"Window end. RFC3339, or YYYY-MM-DD including that UTC day. Required."`
	GroupBy string `json:"groupBy,omitempty" jsonschema:"project, activity, or day. Default project."`
}

type SummaryGroup struct {
	Key          string `json:"key"`
	Label        string `json:"label"`
	DurationMs   int64  `json:"durationMs"`
	SessionCount int    `json:"sessionCount"`
}

type Summary struct {
	From            string         `json:"from"`
	To              string         `json:"to"`
	GroupBy         string         `json:"groupBy"`
	TotalDurationMs int64          `json:"totalDurationMs"`
	SessionCount    int            `json:"sessionCount"`
	Groups          []SummaryGroup `json:"groups"`
}

func (t *Tools) summarize(ctx context.Context, _ *mcp.CallToolRequest, in SummarizeInput) (*mcp.CallToolResult, Summary, error) {
	id, err := t.userID(ctx)
	if err != nil {
		return nil, Summary{}, err
	}
	from, err := parseBound(in.From, false)
	if err != nil {
		return nil, Summary{}, fmt.Errorf("from: %w", err)
	}
	to, err := parseBound(in.To, true)
	if err != nil {
		return nil, Summary{}, fmt.Errorf("to: %w", err)
	}
	if !to.After(from) {
		return nil, Summary{}, fmt.Errorf("to must be after from")
	}
	groupBy, err := normalizeGroupBy(in.GroupBy)
	if err != nil {
		return nil, Summary{}, err
	}
	rows, err := t.Read.ListAgentSessionsInWindow(ctx, id, from, to)
	if err != nil {
		return nil, Summary{}, err
	}
	return nil, summarizeRows(rows, from, to, t.now(), groupBy), nil
}

func normalizeGroupBy(raw string) (string, error) {
	g := strings.ToLower(strings.TrimSpace(raw))
	if g == "" {
		return groupProject, nil
	}
	switch g {
	case groupProject, groupActivity, groupDay:
		return g, nil
	default:
		return "", fmt.Errorf("groupBy must be project, activity, or day")
	}
}

type bucket struct {
	key   string
	label string
	ms    int64
	count int
}

func summarizeRows(rows []store.AgentSession, from, to, now time.Time, groupBy string) Summary {
	buckets := map[string]*bucket{}
	var keys []string
	var total int64
	var sessions int
	touch := func(key, label string, ms int64) {
		if ms <= 0 {
			return
		}
		b, ok := buckets[key]
		if !ok {
			b = &bucket{key: key, label: label}
			buckets[key] = b
			keys = append(keys, key)
		}
		b.ms += ms
		b.count++
		total += ms
	}
	for _, row := range rows {
		start, end, ok := clipInterval(row.StartedAt, row.EndedAt, from, to, now)
		if !ok {
			continue
		}
		sessions++
		switch groupBy {
		case groupDay:
			for _, part := range splitDays(start, end) {
				touch(part.day, part.day, part.ms)
			}
		case groupActivity:
			key, label := "none", "No activity type"
			if row.ActivityTypeID != nil {
				key = *row.ActivityTypeID
			}
			if row.ActivityTypeName != nil {
				label = *row.ActivityTypeName
			}
			touch(key, label, end.Sub(start).Milliseconds())
		default:
			touch(row.ProjectID, row.ProjectName, end.Sub(start).Milliseconds())
		}
	}
	if groupBy == groupDay {
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	} else {
		sort.Slice(keys, func(i, j int) bool {
			if buckets[keys[i]].ms != buckets[keys[j]].ms {
				return buckets[keys[i]].ms > buckets[keys[j]].ms
			}
			return buckets[keys[i]].label < buckets[keys[j]].label
		})
	}
	groups := make([]SummaryGroup, 0, len(keys))
	for _, key := range keys {
		b := buckets[key]
		groups = append(groups, SummaryGroup{
			Key:          b.key,
			Label:        b.label,
			DurationMs:   b.ms,
			SessionCount: b.count,
		})
	}
	return Summary{
		From:            formatTime(from),
		To:              formatTime(to),
		GroupBy:         groupBy,
		TotalDurationMs: total,
		SessionCount:    sessions,
		Groups:          groups,
	}
}
