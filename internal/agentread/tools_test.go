package agentread

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/EmilM32/vynno-api/internal/store"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type fakeReader struct {
	seen     []uuid.UUID
	profile  map[uuid.UUID]domain.Profile
	projects map[uuid.UUID][]domain.Project
	sessions map[uuid.UUID][]store.AgentSession
	live     map[uuid.UUID]store.AgentSession
	window   map[uuid.UUID][]store.AgentSession
	filter   store.AgentSessionFilter
}

func (f *fakeReader) note(id uuid.UUID) {
	f.seen = append(f.seen, id)
}

func (f *fakeReader) GetProfile(_ context.Context, userID uuid.UUID) (domain.Profile, error) {
	f.note(userID)
	p, ok := f.profile[userID]
	if !ok {
		return domain.Profile{}, domain.ErrNotFound()
	}
	return p, nil
}

func (f *fakeReader) ListAgentProjects(_ context.Context, userID uuid.UUID, _ bool) ([]domain.Project, error) {
	f.note(userID)
	return f.projects[userID], nil
}

func (f *fakeReader) ListAgentSessions(_ context.Context, userID uuid.UUID, filter store.AgentSessionFilter) ([]store.AgentSession, error) {
	f.note(userID)
	f.filter = filter
	return f.sessions[userID], nil
}

func (f *fakeReader) GetAgentLiveSession(_ context.Context, userID uuid.UUID) (store.AgentSession, bool, error) {
	f.note(userID)
	row, ok := f.live[userID]
	return row, ok, nil
}

func (f *fakeReader) ListAgentSessionsInWindow(_ context.Context, userID uuid.UUID, _, _ time.Time) ([]store.AgentSession, error) {
	f.note(userID)
	return f.window[userID], nil
}

func TestToolsStayOnTheAuthenticatedAccount(t *testing.T) {
	owner := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	other := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	hour := time.Hour
	start := time.Date(2026, 9, 25, 8, 0, 0, 0, time.UTC)
	end := start.Add(hour)
	fake := &fakeReader{
		profile: map[uuid.UUID]domain.Profile{
			owner: {Email: "owner@example.com", DisplayName: "Owner"},
			other: {Email: "other@example.com", DisplayName: "Other"},
		},
		window: map[uuid.UUID][]store.AgentSession{
			owner: {{
				ID: "own", ProjectID: "p", ProjectName: "Mine",
				StartedAt: start, EndedAt: &end, Status: "stopped",
			}},
			other: {{
				ID: "theirs", ProjectID: "p2", ProjectName: "Theirs",
				StartedAt: start, EndedAt: ptrTime(start.Add(10 * time.Hour)), Status: "stopped",
			}},
		},
	}
	tools := &Tools{
		Read: fake,
		Resolve: func(context.Context) (uuid.UUID, error) {
			return owner, nil
		},
		Now: func() time.Time { return end },
	}

	_, who, err := tools.whoami(context.Background(), nil, struct{}{})
	if err != nil {
		t.Fatal(err)
	}
	if who.Email != "owner@example.com" {
		t.Fatalf("whoami = %+v", who)
	}
	_, sum, err := tools.summarize(context.Background(), nil, SummarizeInput{From: "2026-09-25", To: "2026-09-25"})
	if err != nil {
		t.Fatal(err)
	}
	if sum.TotalDurationMs != hour.Milliseconds() {
		t.Fatalf("summary included another account: %+v", sum)
	}
	for _, id := range fake.seen {
		if id != owner {
			t.Fatalf("reader saw %s", id)
		}
	}
}

func TestWhoAmIRejectsABadToken(t *testing.T) {
	tools := &Tools{
		Read: &fakeReader{},
		Resolve: func(context.Context) (uuid.UUID, error) {
			return uuid.Nil, domain.ErrUnauthorized()
		},
	}
	_, _, err := tools.whoami(context.Background(), nil, struct{}{})
	if !errors.Is(err, errUnauthorized) {
		t.Fatalf("err = %v", err)
	}
}

func TestListSessionsCapsLimitAndRejectsStatus(t *testing.T) {
	owner := uuid.New()
	fake := &fakeReader{sessions: map[uuid.UUID][]store.AgentSession{}}
	tools := &Tools{Read: fake, Resolve: func(context.Context) (uuid.UUID, error) { return owner, nil }}

	_, got, err := tools.listSessions(context.Background(), nil, ListSessionsInput{Limit: 500})
	if err != nil {
		t.Fatal(err)
	}
	if got.Limit != store.AgentSessionLimitMax || fake.filter.Limit != store.AgentSessionLimitMax {
		t.Fatalf("limit = %d filter %d", got.Limit, fake.filter.Limit)
	}

	_, _, err = tools.listSessions(context.Background(), nil, ListSessionsInput{Status: "paused"})
	if err == nil || !strings.Contains(err.Error(), "active or stopped") {
		t.Fatalf("err = %v", err)
	}
}

func TestActiveSessionIdle(t *testing.T) {
	owner := uuid.New()
	tools := &Tools{
		Read:    &fakeReader{live: map[uuid.UUID]store.AgentSession{}},
		Resolve: func(context.Context) (uuid.UUID, error) { return owner, nil },
	}
	_, got, err := tools.activeSession(context.Background(), nil, struct{}{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Active || got.Session != nil {
		t.Fatalf("idle = %+v", got)
	}
}

func TestRegisteredToolsDoNotAcceptAnAccount(t *testing.T) {
	ctx := context.Background()
	owner := uuid.New()
	fake := &fakeReader{
		profile: map[uuid.UUID]domain.Profile{
			owner: {Email: "owner@example.com", DisplayName: "Owner"},
		},
	}
	server := mcp.NewServer(&mcp.Implementation{Name: "vynno", Version: "test"}, nil)
	Register(server, &Tools{
		Read:    fake,
		Resolve: func(context.Context) (uuid.UUID, error) { return owner, nil },
	})

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, nil)
	left, right := mcp.NewInMemoryTransports()
	if _, err := server.Connect(ctx, left, nil); err != nil {
		t.Fatal(err)
	}
	session, err := client.Connect(ctx, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = session.Close() }()

	names := map[string]bool{}
	for tool, err := range session.Tools(ctx, nil) {
		if err != nil {
			t.Fatal(err)
		}
		names[tool.Name] = true
		raw, err := json.Marshal(tool.InputSchema)
		if err != nil {
			t.Fatal(err)
		}
		schema := strings.ToLower(string(raw))
		if strings.Contains(schema, "email") || strings.Contains(schema, "userid") || strings.Contains(schema, "user_id") {
			t.Fatalf("%s schema lets the caller pick an account: %s", tool.Name, raw)
		}
		if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
			t.Fatalf("%s is not marked read-only", tool.Name)
		}
	}
	for _, name := range []string{
		"vynno_whoami",
		"vynno_list_projects",
		"vynno_list_sessions",
		"vynno_get_active_session",
		"vynno_summarize_time",
	} {
		if !names[name] {
			t.Fatalf("missing tool %s in %v", name, names)
		}
	}

	if _, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "vynno_whoami",
		Arguments: map[string]any{"email": "other@example.com"},
	}); err == nil || !strings.Contains(err.Error(), "additional properties") {
		t.Fatalf("email argument was accepted: %v", err)
	}
	if len(fake.seen) != 0 {
		t.Fatalf("rejected call still read the database: %v", fake.seen)
	}

	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "vynno_whoami"})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("whoami error: %+v", res)
	}
	if len(fake.seen) != 1 || fake.seen[0] != owner {
		t.Fatalf("seen = %v", fake.seen)
	}
}
