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
	email := opts.BootstrapEmail
	if email == "" {
		email = "alexdev@vynno.local"
	}
	return Dataset{Accounts: []Account{{
		ID:       store.DefaultUserID(),
		Email:    email,
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
	bootstrapEmail := opts.BootstrapEmail
	if bootstrapEmail == "" {
		bootstrapEmail = "alexdev@vynno.local"
	}

	accounts := make([]Account, 0, 3)
	for i, p := range seedPersonas() {
		email := p.email
		password := seedPass
		if i == 0 {
			email = bootstrapEmail
			password = opts.BootstrapPassword
		}
		// Isolated stream per persona so tuning one does not reshape the others.
		rng := rand.New(rand.NewPCG(randSeed, uint64(i+1)))
		acc := buildAccount(rng, now, p, email, password)
		accounts = append(accounts, acc)
	}
	return Dataset{Accounts: accounts}
}

func buildAccount(rng *rand.Rand, now time.Time, p persona, email, password string) Account {
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
		Email:    email,
		Password: password,
		Blurb:    p.blurb,
		Profile: domain.Profile{
			DisplayName: p.displayName,
		},
		Projects:      projects,
		ActivityTypes: types,
		Sessions:      sessions,
	}
}

func seedActivityTypes() []domain.ActivityType {
	return []domain.ActivityType{
		{ID: uuid.New().String(), Name: "Deep work", Color: "primary"},
		{ID: uuid.New().String(), Name: "Meeting", Color: "tertiary"},
		{ID: uuid.New().String(), Name: "Maintenance", Color: "primary"},
		{ID: uuid.New().String(), Name: "Coding", Color: "secondary"},
		{ID: uuid.New().String(), Name: "Debugging", Color: "error"},
		{ID: uuid.New().String(), Name: "Docs", Color: "on-surface-variant"},
		{ID: uuid.New().String(), Name: "Research", Color: "primary"},
		{ID: uuid.New().String(), Name: "Other", Color: "outline"},
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
		var lockSpec *projectSpec
		var lockProj *domain.Project
		var lockTicket *string
		lockLeft := 0
		for i := 0; i < n; i++ {
			if !cursor.Before(cutoff) {
				break
			}
			var spec projectSpec
			var proj domain.Project
			useLock := lockLeft > 0 && lockSpec != nil && lockProj != nil
			if useLock {
				spec, proj = *lockSpec, *lockProj
				lockLeft--
			} else {
				var ok bool
				spec, proj, ok = pickProject(rng, now, day, p.projects, projects)
				if !ok {
					break
				}
			}
			dur := pickDuration(rng)
			start := cursor
			end := start.Add(dur)
			if !end.Before(cutoff) && !end.Equal(cutoff) {
				break
			}
			var forced *string
			if useLock {
				forced = lockTicket
			}
			sess, err := buildStopped(rng, spec, proj, start, end, activityIDs, forced)
			if err != nil {
				break
			}
			if !useLock && spec.ticketPrefix != "" && sess.TicketID != nil && i+2 < n && rng.Float64() < 0.55 {
				sCopy, pCopy := spec, proj
				tCopy := *sess.TicketID
				lockSpec, lockProj, lockTicket = &sCopy, &pCopy, &tCopy
				lockLeft = 1 + rng.IntN(2)
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

func buildStopped(rng *rand.Rand, spec projectSpec, proj domain.Project, start, end time.Time, activityIDs map[string]string, forcedTicket *string) (domain.Session, error) {
	note, ticket, activity, target := sessionFields(rng, spec, activityIDs, forcedTicket)
	s := domain.StartSession(uuid.New().String(), proj.ID, note, ticket, activity, target, start)
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
	idx := -1
	for i, spec := range specs {
		if spec.archived {
			continue
		}
		if spec.fixedID != "" {
			idx = i
			break
		}
		if idx < 0 {
			idx = i
		}
	}
	if idx < 0 {
		return domain.Session{}, false
	}
	spec := specs[idx]
	proj := projects[idx]
	note, ticket, activity, target := sessionFields(rng, spec, activityIDs, nil)
	s := domain.StartSession(uuid.New().String(), proj.ID, note, ticket, activity, target, start)
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

func sessionFields(rng *rand.Rand, spec projectSpec, activityIDs map[string]string, forcedTicket *string) (note string, ticket, activity *string, target *int64) {
	if len(spec.notes) > 0 && rng.Float64() >= 0.04 {
		note = spec.notes[rng.IntN(len(spec.notes))]
	}
	if forcedTicket != nil && *forcedTicket != "" {
		t := *forcedTicket
		ticket = &t
	} else if spec.ticketPrefix != "" && rng.Float64() < 0.42 {
		t := fmt.Sprintf("%s-%d", spec.ticketPrefix, 40+rng.IntN(800))
		ticket = &t
	}
	if len(spec.activities) > 0 && rng.Float64() >= 0.16 {
		slug := spec.activities[rng.IntN(len(spec.activities))]
		if id, ok := activityIDs[slug]; ok {
			activity = &id
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
	return note, ticket, activity, target
}
