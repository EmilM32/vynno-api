package devdata

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/EmilM32/vynno-api/internal/store"
	"github.com/google/uuid"
)

const randSeed = 42

// BuildReset is the empty-db bootstrap: one user, one active project, no sessions.
func BuildReset(opts Options) Dataset {
	username := opts.BootstrapUsername
	if username == "" {
		username = "alexdev"
	}
	return Dataset{Accounts: []Account{{
		ID:       store.DefaultUserID(),
		Username: username,
		Password: opts.BootstrapPassword,
		Blurb:    "bootstrap, Identity only",
		Profile:  store.DefaultProfile(),
		Projects: []domain.Project{store.DefaultProject()},
	}}}
}

// BuildSeed wipes conceptually (caller wipes the DB) and returns three personas
// with production-like history relative to opts.Now.
func BuildSeed(opts Options) Dataset {
	now := opts.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	seedPass := opts.SeedPassword
	if seedPass == "" {
		seedPass = DefaultSeedPassword
	}
	bootstrapUser := opts.BootstrapUsername
	if bootstrapUser == "" {
		bootstrapUser = "alexdev"
	}

	accounts := make([]Account, 0, 3)
	for i, p := range seedPersonas() {
		username := p.username
		password := seedPass
		if i == 0 {
			username = bootstrapUser
			password = opts.BootstrapPassword
		}
		// Isolated stream per persona so tuning one does not reshape the others.
		rng := rand.New(rand.NewPCG(randSeed, uint64(i+1)))
		acc := buildAccount(rng, now, p, username, password)
		accounts = append(accounts, acc)
	}
	return Dataset{Accounts: accounts}
}

func buildAccount(rng *rand.Rand, now time.Time, p persona, username, password string) Account {
	projects := make([]domain.Project, 0, len(p.projects))
	for _, spec := range p.projects {
		projects = append(projects, projectFromSpec(spec))
	}
	types := seedActivityTypes()
	byName := map[string]string{}
	for _, a := range types {
		byName[a.Name] = a.ID
	}
	sessions := generateSessions(rng, now, p, projects, byName)
	return Account{
		ID:       p.id,
		Username: username,
		Password: password,
		Blurb:    p.blurb,
		Profile: domain.Profile{
			DisplayName: p.displayName,
			Handle:      domain.HandleFromUsername(username),
		},
		Projects:      projects,
		ActivityTypes: types,
		Sessions:      sessions,
	}
}

func seedActivityTypes() []domain.ActivityType {
	return []domain.ActivityType{
		{ID: uuid.New().String(), Name: "deep_work", Color: "primary"},
		{ID: uuid.New().String(), Name: "meeting", Color: "tertiary"},
		{ID: uuid.New().String(), Name: "maintenance", Color: "primary"},
		{ID: uuid.New().String(), Name: "coding", Color: "secondary"},
		{ID: uuid.New().String(), Name: "debugging", Color: "error"},
		{ID: uuid.New().String(), Name: "docs", Color: "on-surface-variant"},
		{ID: uuid.New().String(), Name: "research", Color: "primary"},
		{ID: uuid.New().String(), Name: "other", Color: "outline"},
	}
}

func projectFromSpec(spec projectSpec) domain.Project {
	id := spec.fixedID
	if id == "" {
		id = uuid.New().String()
	}
	var code *string
	if spec.code != "" {
		c := spec.code
		code = &c
	}
	return domain.Project{
		ID:              id,
		Name:            spec.name,
		Color:           spec.color,
		Code:            code,
		ProgressPercent: spec.progress,
		Archived:        spec.archived,
	}
}

func generateSessions(rng *rand.Rand, now time.Time, p persona, projects []domain.Project, activityIDs map[string]string) []domain.Session {
	cutoff := now.Add(-45 * time.Minute)
	out := make([]domain.Session, 0, p.daysBack*p.maxPerDay)
	for _, day := range calendarDays(now, p.daysBack) {
		n := sessionsForDay(rng, day, p)
		if n == 0 {
			continue
		}
		cursor := day.Add(time.Duration(8+rng.IntN(2))*time.Hour + time.Duration(rng.IntN(50))*time.Minute)
		for i := 0; i < n; i++ {
			if !cursor.Before(cutoff) {
				break
			}
			spec, proj, ok := pickProject(rng, now, day, p.projects, projects)
			if !ok {
				break
			}
			dur := pickDuration(rng)
			start := cursor
			end := start.Add(dur)
			paused := rng.Float64() < 0.16
			var pauseLen time.Duration
			if paused {
				pauseLen = time.Duration(5+rng.IntN(16)) * time.Minute
				end = end.Add(pauseLen)
			}
			if !end.Before(cutoff) && !end.Equal(cutoff) {
				break
			}
			sess, err := buildStopped(rng, spec, proj, start, end, pauseLen, activityIDs)
			if err != nil {
				break
			}
			out = append(out, sess)
			cursor = end.Add(time.Duration(12+rng.IntN(40)) * time.Minute)
			if cursor.Hour() >= 18 {
				break
			}
		}
	}
	if p.live {
		if live, ok := buildLive(rng, now, p.projects, projects, out, activityIDs); ok {
			out = append(out, live)
		}
	}
	return out
}

func calendarDays(now time.Time, n int) []time.Time {
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	out := make([]time.Time, n)
	for i := 0; i < n; i++ {
		out[i] = start.AddDate(0, 0, -(n - 1 - i))
	}
	return out
}

func sessionsForDay(rng *rand.Rand, day time.Time, p persona) int {
	wd := day.Weekday()
	if wd == time.Saturday || wd == time.Sunday {
		if rng.Float64() >= p.weekendProb {
			return 0
		}
		return 1
	}
	if rng.Float64() < p.skipWeekday {
		return 0
	}
	if p.maxPerDay <= p.minPerDay {
		return p.minPerDay
	}
	return p.minPerDay + rng.IntN(p.maxPerDay-p.minPerDay+1)
}

func pickProject(rng *rand.Rand, now, day time.Time, specs []projectSpec, projects []domain.Project) (projectSpec, domain.Project, bool) {
	type pair struct {
		spec projectSpec
		proj domain.Project
	}
	var eligible []pair
	for i, spec := range specs {
		if spec.onlyBeforeDays > 0 && !day.Before(now.AddDate(0, 0, -spec.onlyBeforeDays)) {
			continue
		}
		if spec.archived && spec.onlyBeforeDays == 0 {
			continue
		}
		eligible = append(eligible, pair{spec: spec, proj: projects[i]})
	}
	if len(eligible) == 0 {
		return projectSpec{}, domain.Project{}, false
	}
	choice := eligible[rng.IntN(len(eligible))]
	return choice.spec, choice.proj, true
}

func pickDuration(rng *rand.Rand) time.Duration {
	n := rng.IntN(100)
	switch {
	case n < 8:
		return time.Duration(8+rng.IntN(8)) * time.Minute
	case n < 18:
		return 3*time.Hour + time.Duration(rng.IntN(25))*time.Minute
	case n < 48:
		return time.Duration(25+rng.IntN(10)) * time.Minute
	default:
		return time.Duration(45+rng.IntN(50)) * time.Minute
	}
}

func buildStopped(rng *rand.Rand, spec projectSpec, proj domain.Project, start, end time.Time, pauseLen time.Duration, activityIDs map[string]string) (domain.Session, error) {
	note, ticket, activity, tags, target := sessionFields(rng, spec, activityIDs)
	s := domain.StartSession(uuid.New().String(), proj.ID, note, ticket, activity, tags, target, start)
	if pauseLen > 0 {
		pauseAt := start.Add((end.Sub(start) - pauseLen) / 3)
		if !pauseAt.After(start) {
			pauseAt = start.Add(time.Minute)
		}
		resumeAt := pauseAt.Add(pauseLen)
		if !resumeAt.Before(end) {
			resumeAt = end.Add(-time.Minute)
			if !resumeAt.After(pauseAt) {
				return domain.Stop(s, end)
			}
		}
		var err error
		s, err = domain.Pause(s, pauseAt)
		if err != nil {
			return domain.Session{}, err
		}
		s, err = domain.Resume(s, resumeAt)
		if err != nil {
			return domain.Session{}, err
		}
	}
	return domain.Stop(s, end)
}

func buildLive(rng *rand.Rand, now time.Time, specs []projectSpec, projects []domain.Project, existing []domain.Session, activityIDs map[string]string) (domain.Session, bool) {
	start := now.Add(-40 * time.Minute)
	if lastEnd := latestEnd(existing); lastEnd != nil && !lastEnd.Before(start) {
		start = lastEnd.Add(5 * time.Minute)
	}
	if !start.Before(now) {
		return domain.Session{}, false
	}
	var active []int
	for i, spec := range specs {
		if !spec.archived {
			active = append(active, i)
		}
	}
	if len(active) == 0 {
		return domain.Session{}, false
	}
	idx := active[rng.IntN(len(active))]
	spec := specs[idx]
	proj := projects[idx]
	note, ticket, activity, tags, target := sessionFields(rng, spec, activityIDs)
	s := domain.StartSession(uuid.New().String(), proj.ID, note, ticket, activity, tags, target, start)
	return s, true
}

func latestEnd(sessions []domain.Session) *time.Time {
	var latest *time.Time
	for i := range sessions {
		end := sessions[i].EndedAt
		if end == nil {
			end = &sessions[i].StartedAt
		}
		if latest == nil || end.After(*latest) {
			latest = end
		}
	}
	return latest
}

func sessionFields(rng *rand.Rand, spec projectSpec, activityIDs map[string]string) (note string, ticket, activity *string, tags []string, target *int64) {
	if len(spec.notes) > 0 && rng.Float64() >= 0.04 {
		note = spec.notes[rng.IntN(len(spec.notes))]
	}
	if spec.ticketPrefix != "" && rng.Float64() < 0.42 {
		t := fmt.Sprintf("%s-%d", spec.ticketPrefix, 40+rng.IntN(800))
		ticket = &t
	}
	if len(spec.activities) > 0 && rng.Float64() >= 0.16 {
		slug := spec.activities[rng.IntN(len(spec.activities))]
		if id, ok := activityIDs[slug]; ok {
			activity = &id
		}
	}
	if rng.Float64() < 0.32 {
		n := 1
		if rng.Float64() < 0.25 {
			n = 2
		}
		seen := map[string]bool{}
		for len(tags) < n {
			t := tagBank[rng.IntN(len(tagBank))]
			if seen[t] {
				continue
			}
			seen[t] = true
			tags = append(tags, t)
		}
	}
	switch {
	case rng.Float64() < 0.10:
		v := int64(25 * 60 * 1000)
		target = &v
	case rng.Float64() < 0.05:
		v := int64(90 * 60 * 1000)
		target = &v
	}
	return note, ticket, activity, tags, target
}
