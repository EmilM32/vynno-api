package store

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/google/uuid"
)

type memAvatar struct {
	id          uuid.UUID
	userID      uuid.UUID
	contentType string
	bytes       []byte
}

type memAccount struct {
	id            uuid.UUID
	username      string
	passwordHash  string
	profile       domain.Profile
	projects      map[uuid.UUID]domain.Project
	activityTypes map[uuid.UUID]domain.ActivityType
	sessions      map[uuid.UUID]domain.Session
}

// Memory is an in-memory Store for tests. Data is scoped by user.
type Memory struct {
	mu       sync.Mutex
	accounts map[uuid.UUID]*memAccount
	byName   map[string]uuid.UUID
	tokens   map[string]Token
	avatars  map[uuid.UUID]memAvatar
}

// NewEmptyMemory is an in-memory Store with no accounts. Used by operator-tooling tests.
func NewEmptyMemory() *Memory {
	return &Memory{
		accounts: map[uuid.UUID]*memAccount{},
		byName:   map[string]uuid.UUID{},
		tokens:   map[string]Token{},
		avatars:  map[uuid.UUID]memAvatar{},
	}
}

func NewMemory(userID uuid.UUID, profile domain.Profile, project domain.Project) *Memory {
	pid, err := uuid.Parse(project.ID)
	if err != nil {
		pid = uuid.New()
		project.ID = pid.String()
	}
	return &Memory{
		accounts: map[uuid.UUID]*memAccount{
			userID: {
				id:            userID,
				profile:       profile,
				projects:      map[uuid.UUID]domain.Project{pid: project},
				activityTypes: map[uuid.UUID]domain.ActivityType{},
				sessions:      map[uuid.UUID]domain.Session{},
			},
		},
		byName:  map[string]uuid.UUID{},
		tokens:  map[string]Token{},
		avatars: map[uuid.UUID]memAvatar{},
	}
}

func (m *Memory) account(userID uuid.UUID) (*memAccount, bool) {
	a, ok := m.accounts[userID]
	return a, ok
}

func (m *Memory) GetProfile(_ context.Context, userID uuid.UUID) (domain.Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.account(userID)
	if !ok {
		return domain.Profile{}, domain.ErrNotFound()
	}
	return a.profile, nil
}

func (m *Memory) CreateProfile(_ context.Context, userID uuid.UUID, p domain.Profile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.account(userID)
	if !ok {
		return domain.ErrNotFound()
	}
	a.profile = p
	return nil
}

func (m *Memory) UpdateProfileDisplayName(_ context.Context, userID uuid.UUID, displayName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.account(userID)
	if !ok {
		return domain.ErrNotFound()
	}
	a.profile.DisplayName = displayName
	return nil
}

func (m *Memory) ReplaceAvatar(_ context.Context, userID uuid.UUID, av domain.Avatar) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.account(userID)
	if !ok {
		return domain.ErrNotFound()
	}
	id, err := uuid.Parse(av.ID)
	if err != nil {
		return domain.ErrInvalidBody("invalid id")
	}
	for existingID, row := range m.avatars {
		if row.userID == userID {
			delete(m.avatars, existingID)
		}
	}
	data := append([]byte(nil), av.Bytes...)
	m.avatars[id] = memAvatar{id: id, userID: userID, contentType: av.ContentType, bytes: data}
	path := domain.AvatarPath(id.String())
	a.profile.AvatarURL = &path
	return nil
}

func (m *Memory) DeleteAvatarByUser(_ context.Context, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.account(userID)
	if !ok {
		return domain.ErrNotFound()
	}
	for existingID, row := range m.avatars {
		if row.userID == userID {
			delete(m.avatars, existingID)
		}
	}
	a.profile.AvatarURL = nil
	return nil
}

func (m *Memory) GetAvatar(_ context.Context, id uuid.UUID) (domain.Avatar, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.avatars[id]
	if !ok {
		return domain.Avatar{}, domain.ErrNotFound()
	}
	return domain.Avatar{
		ID:          row.id.String(),
		ContentType: row.contentType,
		Bytes:       append([]byte(nil), row.bytes...),
	}, nil
}

func (m *Memory) GetAccountByUsername(_ context.Context, username string) (Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byName[username]
	if !ok {
		return Account{}, domain.ErrNotFound()
	}
	a := m.accounts[id]
	return Account{ID: a.id, Username: a.username, PasswordHash: a.passwordHash}, nil
}

func (m *Memory) GetAccountByID(_ context.Context, id uuid.UUID) (Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.account(id)
	if !ok {
		return Account{}, domain.ErrNotFound()
	}
	return Account{ID: a.id, Username: a.username, PasswordHash: a.passwordHash}, nil
}

func (m *Memory) CreateAccount(_ context.Context, acc Account) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if acc.Username != "" {
		if _, taken := m.byName[acc.Username]; taken {
			return domain.ErrUsernameInUse()
		}
	}
	if _, exists := m.accounts[acc.ID]; exists {
		return domain.ErrUsernameInUse()
	}
	m.accounts[acc.ID] = &memAccount{
		id:            acc.ID,
		username:      acc.Username,
		passwordHash:  acc.PasswordHash,
		projects:      map[uuid.UUID]domain.Project{},
		activityTypes: map[uuid.UUID]domain.ActivityType{},
		sessions:      map[uuid.UUID]domain.Session{},
	}
	if acc.Username != "" {
		m.byName[acc.Username] = acc.ID
	}
	return nil
}

func (m *Memory) SetAccountCredentials(_ context.Context, id uuid.UUID, username, passwordHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.account(id)
	if !ok {
		return domain.ErrNotFound()
	}
	if other, taken := m.byName[username]; taken && other != id {
		return domain.ErrUsernameInUse()
	}
	if a.username != "" && a.username != username {
		delete(m.byName, a.username)
	}
	a.username = username
	a.passwordHash = passwordHash
	m.byName[username] = id
	return nil
}

func (m *Memory) UsernameTaken(_ context.Context, username string, excludeID uuid.UUID) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byName[username]
	return ok && id != excludeID, nil
}

func (m *Memory) CreateToken(_ context.Context, tok Token) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens[tok.TokenHash] = tok
	return nil
}

func (m *Memory) GetTokenByHash(_ context.Context, hash string) (Token, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tok, ok := m.tokens[hash]
	if !ok {
		return Token{}, domain.ErrNotFound()
	}
	return tok, nil
}

func (m *Memory) DeleteTokenByHash(_ context.Context, hash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tokens, hash)
	return nil
}

func (m *Memory) ListProjects(_ context.Context, userID uuid.UUID, includeArchived bool) ([]domain.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.account(userID)
	if !ok {
		return nil, domain.ErrNotFound()
	}
	out := make([]domain.Project, 0, len(a.projects))
	for _, p := range a.projects {
		if p.Archived && !includeArchived {
			continue
		}
		out = append(out, cloneProject(p))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (m *Memory) GetProject(_ context.Context, userID, id uuid.UUID) (domain.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.account(userID)
	if !ok {
		return domain.Project{}, domain.ErrNotFound()
	}
	p, ok := a.projects[id]
	if !ok {
		return domain.Project{}, domain.ErrNotFound()
	}
	return cloneProject(p), nil
}

func (m *Memory) CreateProject(_ context.Context, userID uuid.UUID, p domain.Project) (domain.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.account(userID)
	if !ok {
		return domain.Project{}, domain.ErrNotFound()
	}
	id, err := uuid.Parse(p.ID)
	if err != nil {
		return domain.Project{}, domain.ErrInvalidBody("invalid id")
	}
	if p.Code != nil && m.codeTaken(a, *p.Code, uuid.Nil) {
		return domain.Project{}, domain.ErrCodeInUse()
	}
	a.projects[id] = cloneProject(p)
	return cloneProject(p), nil
}

func (m *Memory) UpdateProject(_ context.Context, userID uuid.UUID, p domain.Project) (domain.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.account(userID)
	if !ok {
		return domain.Project{}, domain.ErrNotFound()
	}
	id, err := uuid.Parse(p.ID)
	if err != nil {
		return domain.Project{}, domain.ErrNotFound()
	}
	if _, ok := a.projects[id]; !ok {
		return domain.Project{}, domain.ErrNotFound()
	}
	if p.Code != nil && m.codeTaken(a, *p.Code, id) {
		return domain.Project{}, domain.ErrCodeInUse()
	}
	a.projects[id] = cloneProject(p)
	return cloneProject(p), nil
}

func (m *Memory) DeleteProject(_ context.Context, userID, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.account(userID)
	if !ok {
		return domain.ErrNotFound()
	}
	if _, ok := a.projects[id]; !ok {
		return domain.ErrNotFound()
	}
	delete(a.projects, id)
	return nil
}

func (m *Memory) CountActiveProjects(_ context.Context, userID uuid.UUID) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.account(userID)
	if !ok {
		return 0, domain.ErrNotFound()
	}
	n := 0
	for _, p := range a.projects {
		if !p.Archived {
			n++
		}
	}
	return n, nil
}

func (m *Memory) CountProjectSessions(_ context.Context, userID, projectID uuid.UUID) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.account(userID)
	if !ok {
		return 0, domain.ErrNotFound()
	}
	n := 0
	want := projectID.String()
	for _, s := range a.sessions {
		if s.ProjectID == want {
			n++
		}
	}
	return n, nil
}

func (m *Memory) CodeInUse(_ context.Context, userID uuid.UUID, code string, excludeID uuid.UUID) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.account(userID)
	if !ok {
		return false, domain.ErrNotFound()
	}
	return m.codeTaken(a, code, excludeID), nil
}

func (m *Memory) codeTaken(a *memAccount, code string, excludeID uuid.UUID) bool {
	want := strings.ToUpper(code)
	for id, p := range a.projects {
		if id == excludeID || p.Code == nil {
			continue
		}
		if strings.ToUpper(*p.Code) == want {
			return true
		}
	}
	return false
}

func (m *Memory) ListActivityTypes(_ context.Context, userID uuid.UUID) ([]domain.ActivityType, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.account(userID)
	if !ok {
		return nil, domain.ErrNotFound()
	}
	out := make([]domain.ActivityType, 0, len(a.activityTypes))
	for _, t := range a.activityTypes {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out, nil
}

func (m *Memory) GetActivityType(_ context.Context, userID, id uuid.UUID) (domain.ActivityType, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.account(userID)
	if !ok {
		return domain.ActivityType{}, domain.ErrNotFound()
	}
	t, ok := a.activityTypes[id]
	if !ok {
		return domain.ActivityType{}, domain.ErrNotFound()
	}
	return t, nil
}

func (m *Memory) CreateActivityType(_ context.Context, userID uuid.UUID, t domain.ActivityType) (domain.ActivityType, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.account(userID)
	if !ok {
		return domain.ActivityType{}, domain.ErrNotFound()
	}
	id, err := uuid.Parse(t.ID)
	if err != nil {
		return domain.ActivityType{}, domain.ErrInvalidBody("invalid id")
	}
	if m.activityNameTaken(a, t.Name, uuid.Nil) {
		return domain.ActivityType{}, domain.ErrNameInUse()
	}
	if a.activityTypes == nil {
		a.activityTypes = map[uuid.UUID]domain.ActivityType{}
	}
	a.activityTypes[id] = t
	return t, nil
}

func (m *Memory) UpdateActivityType(_ context.Context, userID uuid.UUID, t domain.ActivityType) (domain.ActivityType, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.account(userID)
	if !ok {
		return domain.ActivityType{}, domain.ErrNotFound()
	}
	id, err := uuid.Parse(t.ID)
	if err != nil {
		return domain.ActivityType{}, domain.ErrNotFound()
	}
	if _, ok := a.activityTypes[id]; !ok {
		return domain.ActivityType{}, domain.ErrNotFound()
	}
	if m.activityNameTaken(a, t.Name, id) {
		return domain.ActivityType{}, domain.ErrNameInUse()
	}
	a.activityTypes[id] = t
	return t, nil
}

func (m *Memory) DeleteActivityType(_ context.Context, userID, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.account(userID)
	if !ok {
		return domain.ErrNotFound()
	}
	if _, ok := a.activityTypes[id]; !ok {
		return domain.ErrNotFound()
	}
	delete(a.activityTypes, id)
	return nil
}

func (m *Memory) CountActivityTypeSessions(_ context.Context, userID, activityTypeID uuid.UUID) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.account(userID)
	if !ok {
		return 0, domain.ErrNotFound()
	}
	want := activityTypeID.String()
	n := 0
	for _, s := range a.sessions {
		if s.ActivityTypeID != nil && *s.ActivityTypeID == want {
			n++
		}
	}
	return n, nil
}

func (m *Memory) ActivityTypeNameInUse(_ context.Context, userID uuid.UUID, name string, excludeID uuid.UUID) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.account(userID)
	if !ok {
		return false, domain.ErrNotFound()
	}
	return m.activityNameTaken(a, name, excludeID), nil
}

func (m *Memory) activityNameTaken(a *memAccount, name string, excludeID uuid.UUID) bool {
	want := strings.ToLower(name)
	for id, t := range a.activityTypes {
		if id == excludeID {
			continue
		}
		if strings.ToLower(t.Name) == want {
			return true
		}
	}
	return false
}

func (m *Memory) ListSessions(_ context.Context, userID uuid.UUID, statuses []string, limit int, cursor string) (SessionPage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.account(userID)
	if !ok {
		return SessionPage{}, domain.ErrNotFound()
	}
	allow := map[string]bool{}
	for _, s := range statuses {
		allow[s] = true
	}
	out := make([]domain.Session, 0, len(a.sessions))
	for _, s := range a.sessions {
		if len(allow) > 0 && !allow[s.Status] {
			continue
		}
		out = append(out, cloneSession(s))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].StartedAt.Equal(out[j].StartedAt) {
			return out[i].ID > out[j].ID
		}
		return out[i].StartedAt.After(out[j].StartedAt)
	})
	return paginateSessions(out, limit, cursor)
}

func (m *Memory) GetSession(_ context.Context, userID, id uuid.UUID) (domain.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.account(userID)
	if !ok {
		return domain.Session{}, domain.ErrNotFound()
	}
	s, ok := a.sessions[id]
	if !ok {
		return domain.Session{}, domain.ErrNotFound()
	}
	return cloneSession(s), nil
}

func (m *Memory) GetLiveSession(_ context.Context, userID uuid.UUID) (domain.Session, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.account(userID)
	if !ok {
		return domain.Session{}, false, domain.ErrNotFound()
	}
	for _, s := range a.sessions {
		if domain.IsLiveStatus(s.Status) {
			return cloneSession(s), true, nil
		}
	}
	return domain.Session{}, false, nil
}

func (m *Memory) CreateSession(_ context.Context, userID uuid.UUID, s domain.Session) (domain.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.account(userID)
	if !ok {
		return domain.Session{}, domain.ErrNotFound()
	}
	if domain.IsLiveStatus(s.Status) {
		for _, existing := range a.sessions {
			if domain.IsLiveStatus(existing.Status) {
				return domain.Session{}, domain.ErrSessionAlreadyActive()
			}
		}
	}
	id, err := uuid.Parse(s.ID)
	if err != nil {
		return domain.Session{}, domain.ErrInvalidBody("invalid id")
	}
	a.sessions[id] = cloneSession(s)
	return cloneSession(s), nil
}

func (m *Memory) UpdateSession(_ context.Context, userID uuid.UUID, s domain.Session) (domain.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.account(userID)
	if !ok {
		return domain.Session{}, domain.ErrNotFound()
	}
	id, err := uuid.Parse(s.ID)
	if err != nil {
		return domain.Session{}, domain.ErrNotFound()
	}
	if _, ok := a.sessions[id]; !ok {
		return domain.Session{}, domain.ErrNotFound()
	}
	a.sessions[id] = cloneSession(s)
	return cloneSession(s), nil
}

func (m *Memory) DeleteSession(_ context.Context, userID, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.account(userID)
	if !ok {
		return domain.ErrNotFound()
	}
	if _, ok := a.sessions[id]; !ok {
		return domain.ErrNotFound()
	}
	delete(a.sessions, id)
	return nil
}

func (m *Memory) FirstUserID(_ context.Context) (uuid.UUID, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id := range m.accounts {
		return id, true, nil
	}
	return uuid.Nil, false, nil
}

func (m *Memory) Bootstrap(_ context.Context, userID uuid.UUID, username, passwordHash string, profile domain.Profile, project domain.Project) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if a, ok := m.accounts[userID]; ok {
		if a.passwordHash != "" {
			return nil
		}
		if username != "" {
			if other, taken := m.byName[username]; taken && other != userID {
				return domain.ErrUsernameInUse()
			}
			a.username = username
			a.passwordHash = passwordHash
			m.byName[username] = userID
		}
		return nil
	}
	if len(m.accounts) > 0 {
		return nil
	}
	pid, err := uuid.Parse(project.ID)
	if err != nil {
		pid = uuid.New()
		project.ID = pid.String()
	}
	m.accounts[userID] = &memAccount{
		id:            userID,
		username:      username,
		passwordHash:  passwordHash,
		profile:       profile,
		projects:      map[uuid.UUID]domain.Project{pid: project},
		activityTypes: map[uuid.UUID]domain.ActivityType{},
		sessions:      map[uuid.UUID]domain.Session{},
	}
	if username != "" {
		m.byName[username] = userID
	}
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
	if s.ActivityTypeID != nil {
		v := *s.ActivityTypeID
		out.ActivityTypeID = &v
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
