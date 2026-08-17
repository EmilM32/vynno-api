package store

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/google/uuid"
)

// Memory is an in-memory Store for tests.
type Memory struct {
	mu       sync.Mutex
	profile  domain.Profile
	userID   uuid.UUID
	projects map[uuid.UUID]domain.Project
	sessions map[uuid.UUID]domain.Session
}

func NewMemory(userID uuid.UUID, profile domain.Profile, project domain.Project) *Memory {
	pid, err := uuid.Parse(project.ID)
	if err != nil {
		pid = uuid.New()
		project.ID = pid.String()
	}
	return &Memory{
		userID:   userID,
		profile:  profile,
		projects: map[uuid.UUID]domain.Project{pid: project},
		sessions: map[uuid.UUID]domain.Session{},
	}
}

func (m *Memory) GetProfile(_ context.Context, _ uuid.UUID) (domain.Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.profile, nil
}

func (m *Memory) ListProjects(_ context.Context, _ uuid.UUID, includeArchived bool) ([]domain.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.Project, 0, len(m.projects))
	for _, p := range m.projects {
		if p.Archived && !includeArchived {
			continue
		}
		out = append(out, cloneProject(p))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (m *Memory) GetProject(_ context.Context, _, id uuid.UUID) (domain.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.projects[id]
	if !ok {
		return domain.Project{}, domain.ErrNotFound()
	}
	return cloneProject(p), nil
}

func (m *Memory) CreateProject(_ context.Context, _ uuid.UUID, p domain.Project) (domain.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, err := uuid.Parse(p.ID)
	if err != nil {
		return domain.Project{}, domain.ErrInvalidBody("invalid id")
	}
	if p.Code != nil && m.codeTaken(*p.Code, uuid.Nil) {
		return domain.Project{}, domain.ErrCodeInUse()
	}
	m.projects[id] = cloneProject(p)
	return cloneProject(p), nil
}

func (m *Memory) UpdateProject(_ context.Context, _ uuid.UUID, p domain.Project) (domain.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, err := uuid.Parse(p.ID)
	if err != nil {
		return domain.Project{}, domain.ErrNotFound()
	}
	if _, ok := m.projects[id]; !ok {
		return domain.Project{}, domain.ErrNotFound()
	}
	if p.Code != nil && m.codeTaken(*p.Code, id) {
		return domain.Project{}, domain.ErrCodeInUse()
	}
	m.projects[id] = cloneProject(p)
	return cloneProject(p), nil
}

func (m *Memory) DeleteProject(_ context.Context, _, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.projects[id]; !ok {
		return domain.ErrNotFound()
	}
	delete(m.projects, id)
	return nil
}

func (m *Memory) CountActiveProjects(_ context.Context, _ uuid.UUID) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, p := range m.projects {
		if !p.Archived {
			n++
		}
	}
	return n, nil
}

func (m *Memory) CountProjectSessions(_ context.Context, _, projectID uuid.UUID) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	want := projectID.String()
	for _, s := range m.sessions {
		if s.ProjectID == want {
			n++
		}
	}
	return n, nil
}

func (m *Memory) CodeInUse(_ context.Context, _ uuid.UUID, code string, excludeID uuid.UUID) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.codeTaken(code, excludeID), nil
}

func (m *Memory) codeTaken(code string, excludeID uuid.UUID) bool {
	want := strings.ToUpper(code)
	for id, p := range m.projects {
		if id == excludeID || p.Code == nil {
			continue
		}
		if strings.ToUpper(*p.Code) == want {
			return true
		}
	}
	return false
}

func (m *Memory) ListSessions(_ context.Context, _ uuid.UUID, statuses []string, limit int) ([]domain.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	allow := map[string]bool{}
	for _, s := range statuses {
		allow[s] = true
	}
	out := make([]domain.Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		if len(allow) > 0 && !allow[s.Status] {
			continue
		}
		out = append(out, cloneSession(s))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt.After(out[j].StartedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *Memory) GetSession(_ context.Context, _, id uuid.UUID) (domain.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	if !ok {
		return domain.Session{}, domain.ErrNotFound()
	}
	return cloneSession(s), nil
}

func (m *Memory) GetLiveSession(_ context.Context, _ uuid.UUID) (domain.Session, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.sessions {
		if domain.IsLiveStatus(s.Status) {
			return cloneSession(s), true, nil
		}
	}
	return domain.Session{}, false, nil
}

func (m *Memory) CreateSession(_ context.Context, _ uuid.UUID, s domain.Session) (domain.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if domain.IsLiveStatus(s.Status) {
		for _, existing := range m.sessions {
			if domain.IsLiveStatus(existing.Status) {
				return domain.Session{}, domain.ErrSessionAlreadyActive()
			}
		}
	}
	id, err := uuid.Parse(s.ID)
	if err != nil {
		return domain.Session{}, domain.ErrInvalidBody("invalid id")
	}
	m.sessions[id] = cloneSession(s)
	return cloneSession(s), nil
}

func (m *Memory) UpdateSession(_ context.Context, _ uuid.UUID, s domain.Session) (domain.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, err := uuid.Parse(s.ID)
	if err != nil {
		return domain.Session{}, domain.ErrNotFound()
	}
	if _, ok := m.sessions[id]; !ok {
		return domain.Session{}, domain.ErrNotFound()
	}
	m.sessions[id] = cloneSession(s)
	return cloneSession(s), nil
}

func (m *Memory) FirstUserID(_ context.Context) (uuid.UUID, bool, error) {
	return m.userID, true, nil
}

func (m *Memory) SeedEmpty(_ context.Context, _ uuid.UUID, _ domain.Profile, _ domain.Project) error {
	return nil
}

func cloneProject(p domain.Project) domain.Project {
	out := p
	if p.Code != nil {
		c := *p.Code
		out.Code = &c
	}
	if p.ProgressPercent != nil {
		n := *p.ProgressPercent
		out.ProgressPercent = &n
	}
	return out
}

func cloneSession(s domain.Session) domain.Session {
	out := s
	if s.TicketID != nil {
		v := *s.TicketID
		out.TicketID = &v
	}
	if s.ActivityType != nil {
		v := *s.ActivityType
		out.ActivityType = &v
	}
	if s.EndedAt != nil {
		v := *s.EndedAt
		out.EndedAt = &v
	}
	if s.PausedAt != nil {
		v := *s.PausedAt
		out.PausedAt = &v
	}
	if s.TargetDurationMs != nil {
		v := *s.TargetDurationMs
		out.TargetDurationMs = &v
	}
	out.Tags = append([]string{}, s.Tags...)
	if out.Tags == nil {
		out.Tags = []string{}
	}
	return out
}
