package store

import (
	"context"
	"testing"
	"time"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/google/uuid"
)

// runStoreContract is the behavior every Store must share. It runs against
// Memory (the double the service and HTTP tests use) and against Postgres, so
// the double cannot drift from the system of record unnoticed.
//
// Every subtest makes its own accounts and never assumes an empty store, so the
// Postgres run needs no truncate between subtests.
//
// Not covered on purpose, because the two stores differ and callers must not
// rely on either answer:
//   - deleting a row that is not there (Memory: not_found, Postgres: nil);
//   - ListProjects order across letter case (Memory: byte order, Postgres: the
//     database collation);
//   - references to another account's rows (Postgres refuses them, see
//     TestPostgresOwnerScopedForeignKeys; Memory does not check).
func runStoreContract(t *testing.T, st Store) {
	t.Run("accounts", func(t *testing.T) { contractAccounts(t, st) })
	t.Run("profile and avatar", func(t *testing.T) { contractProfile(t, st) })
	t.Run("prefs", func(t *testing.T) { contractPrefs(t, st) })
	t.Run("tokens", func(t *testing.T) { contractTokens(t, st) })
	t.Run("email challenges", func(t *testing.T) { contractChallenges(t, st) })
	t.Run("projects", func(t *testing.T) { contractProjects(t, st) })
	t.Run("activity types", func(t *testing.T) { contractActivityTypes(t, st) })
	t.Run("sessions", func(t *testing.T) { contractSessions(t, st) })
	t.Run("session list", func(t *testing.T) { contractSessionList(t, st) })
}

func TestMemoryContract(t *testing.T) {
	runStoreContract(t, NewEmptyMemory())
}

// contractT0 is a fixed instant with microsecond precision, the finest Postgres keeps.
var contractT0 = time.Date(2026, 3, 1, 9, 0, 0, 123456000, time.UTC)

func wantCode(t *testing.T, err error, code string) {
	t.Helper()
	de, ok := domain.AsError(err)
	if !ok || de.Code != code {
		t.Fatalf("error = %v, want domain code %s", err, code)
	}
}

func contractEmail() string {
	return uuid.NewString() + "@example.com"
}

// contractUser creates an account with a profile, as register does.
func contractUser(t *testing.T, st Store) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	id := uuid.New()
	if err := st.CreateAccount(ctx, Account{ID: id, Email: contractEmail(), PasswordHash: "hash"}); err != nil {
		t.Fatalf("create account: %v", err)
	}
	if err := st.CreateProfile(ctx, id, domain.Profile{DisplayName: "Test"}); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	return id
}

func contractProject(t *testing.T, st Store, user uuid.UUID, name string, code *string) domain.Project {
	t.Helper()
	p, err := st.CreateProject(context.Background(), user, domain.Project{
		ID: uuid.NewString(), Name: name, Color: "#3b82f6", Code: code,
	})
	if err != nil {
		t.Fatalf("create project %q: %v", name, err)
	}
	return p
}

func contractStopped(projectID string, start time.Time, d time.Duration) domain.Session {
	end := start.Add(d)
	return domain.Session{
		ID: uuid.NewString(), ProjectID: projectID, Note: "note",
		Status: domain.StatusStopped, StartedAt: start, EndedAt: &end,
	}
}

func contractLive(projectID string, start time.Time) domain.Session {
	return domain.Session{
		ID: uuid.NewString(), ProjectID: projectID, Note: "live",
		Status: domain.StatusActive, StartedAt: start,
	}
}

func strPtr(s string) *string { return &s }

func contractAccounts(t *testing.T, st Store) {
	ctx := context.Background()
	id := uuid.New()
	email := contractEmail()
	if err := st.CreateAccount(ctx, Account{ID: id, Email: email, PasswordHash: "hash-1"}); err != nil {
		t.Fatal(err)
	}

	byEmail, err := st.GetAccountByEmail(ctx, email)
	if err != nil || byEmail.ID != id || byEmail.Email != email || byEmail.PasswordHash != "hash-1" {
		t.Fatalf("by email = %+v, %v", byEmail, err)
	}
	byID, err := st.GetAccountByID(ctx, id)
	if err != nil || byID != byEmail {
		t.Fatalf("by id = %+v, %v", byID, err)
	}
	_, err = st.GetAccountByEmail(ctx, contractEmail())
	wantCode(t, err, domain.CodeNotFound)
	_, err = st.GetAccountByID(ctx, uuid.New())
	wantCode(t, err, domain.CodeNotFound)

	err = st.CreateAccount(ctx, Account{ID: uuid.New(), Email: email, PasswordHash: "x"})
	wantCode(t, err, domain.CodeEmailInUse)

	if taken, err := st.EmailTaken(ctx, email, uuid.Nil); err != nil || !taken {
		t.Fatalf("EmailTaken = %v, %v", taken, err)
	}
	if taken, err := st.EmailTaken(ctx, email, id); err != nil || taken {
		t.Fatalf("EmailTaken excluding the owner = %v, %v", taken, err)
	}
	if taken, err := st.EmailTaken(ctx, contractEmail(), uuid.Nil); err != nil || taken {
		t.Fatalf("EmailTaken for a free address = %v, %v", taken, err)
	}

	newEmail := contractEmail()
	if err := st.SetAccountCredentials(ctx, id, newEmail, "hash-2"); err != nil {
		t.Fatal(err)
	}
	moved, err := st.GetAccountByEmail(ctx, newEmail)
	if err != nil || moved.ID != id || moved.PasswordHash != "hash-2" {
		t.Fatalf("after credential change = %+v, %v", moved, err)
	}
	_, err = st.GetAccountByEmail(ctx, email)
	wantCode(t, err, domain.CodeNotFound)

	// The old address is free again; another account's address is not.
	other := uuid.New()
	if err := st.CreateAccount(ctx, Account{ID: other, Email: email, PasswordHash: "x"}); err != nil {
		t.Fatalf("reuse of a released address: %v", err)
	}
	err = st.SetAccountCredentials(ctx, other, newEmail, "x")
	wantCode(t, err, domain.CodeEmailInUse)
}

func contractProfile(t *testing.T, st Store) {
	ctx := context.Background()
	id := uuid.New()
	email := contractEmail()
	if err := st.CreateAccount(ctx, Account{ID: id, Email: email, PasswordHash: "hash"}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateProfile(ctx, id, domain.Profile{DisplayName: "Zoë"}); err != nil {
		t.Fatal(err)
	}
	p, err := st.GetProfile(ctx, id)
	if err != nil || p.DisplayName != "Zoë" || p.Email != email || p.AvatarURL != nil {
		t.Fatalf("profile = %+v, %v", p, err)
	}
	_, err = st.GetProfile(ctx, uuid.New())
	wantCode(t, err, domain.CodeNotFound)

	if err := st.UpdateProfileDisplayName(ctx, id, ""); err != nil {
		t.Fatal(err)
	}
	if p, _ := st.GetProfile(ctx, id); p.DisplayName != "" {
		t.Fatalf("display name = %q, want empty", p.DisplayName)
	}
	err = st.UpdateProfileDisplayName(ctx, uuid.New(), "x")
	wantCode(t, err, domain.CodeNotFound)

	first := domain.Avatar{ID: uuid.NewString(), ContentType: "image/png", Bytes: []byte{0x89, 'P', 'N', 'G', 0}}
	if err := st.ReplaceAvatar(ctx, id, first); err != nil {
		t.Fatal(err)
	}
	p, _ = st.GetProfile(ctx, id)
	if p.AvatarURL == nil || *p.AvatarURL != domain.AvatarPath(first.ID) {
		t.Fatalf("avatar url = %v", p.AvatarURL)
	}
	got, err := st.GetAvatar(ctx, uuid.MustParse(first.ID))
	if err != nil || got.ContentType != "image/png" || string(got.Bytes) != string(first.Bytes) {
		t.Fatalf("avatar = %+v, %v", got, err)
	}

	second := domain.Avatar{ID: uuid.NewString(), ContentType: "image/webp", Bytes: []byte("RIFF")}
	if err := st.ReplaceAvatar(ctx, id, second); err != nil {
		t.Fatal(err)
	}
	_, err = st.GetAvatar(ctx, uuid.MustParse(first.ID))
	wantCode(t, err, domain.CodeNotFound)
	p, _ = st.GetProfile(ctx, id)
	if p.AvatarURL == nil || *p.AvatarURL != domain.AvatarPath(second.ID) {
		t.Fatalf("avatar url after replace = %v", p.AvatarURL)
	}

	if err := st.DeleteAvatarByUser(ctx, id); err != nil {
		t.Fatal(err)
	}
	_, err = st.GetAvatar(ctx, uuid.MustParse(second.ID))
	wantCode(t, err, domain.CodeNotFound)
	if p, _ := st.GetProfile(ctx, id); p.AvatarURL != nil {
		t.Fatalf("avatar url after delete = %v", *p.AvatarURL)
	}
	// Deleting when there is no avatar is not an error.
	if err := st.DeleteAvatarByUser(ctx, id); err != nil {
		t.Fatal(err)
	}
}

func contractPrefs(t *testing.T, st Store) {
	ctx := context.Background()
	user := contractUser(t, st)

	p, err := st.GetPrefs(ctx, user)
	if err != nil || p.DailyTargetMs != nil || p.DefaultProjectID != nil {
		t.Fatalf("unset prefs = %+v, %v", p, err)
	}

	project := contractProject(t, st, user, "Default", nil)
	target := int64(4 * 60 * 60 * 1000)
	if err := st.SavePrefs(ctx, user, domain.Prefs{DailyTargetMs: &target, DefaultProjectID: &project.ID}); err != nil {
		t.Fatal(err)
	}
	p, err = st.GetPrefs(ctx, user)
	if err != nil || p.DailyTargetMs == nil || *p.DailyTargetMs != target ||
		p.DefaultProjectID == nil || *p.DefaultProjectID != project.ID {
		t.Fatalf("saved prefs = %+v, %v", p, err)
	}

	err = st.SavePrefs(ctx, user, domain.Prefs{DefaultProjectID: strPtr(uuid.NewString())})
	wantCode(t, err, domain.CodeNotFound)

	// Hard-deleting the default project clears only that pref.
	if err := st.DeleteProject(ctx, user, uuid.MustParse(project.ID)); err != nil {
		t.Fatal(err)
	}
	p, err = st.GetPrefs(ctx, user)
	if err != nil || p.DefaultProjectID != nil || p.DailyTargetMs == nil || *p.DailyTargetMs != target {
		t.Fatalf("prefs after project delete = %+v, %v", p, err)
	}

	// Saving replaces every pref: nil clears.
	if err := st.SavePrefs(ctx, user, domain.Prefs{}); err != nil {
		t.Fatal(err)
	}
	if p, _ := st.GetPrefs(ctx, user); p.DailyTargetMs != nil || p.DefaultProjectID != nil {
		t.Fatalf("cleared prefs = %+v", p)
	}
}

func contractTokens(t *testing.T, st Store) {
	ctx := context.Background()
	user := contractUser(t, st)
	other := contractUser(t, st)
	now := contractT0

	mk := func(owner uuid.UUID, expires time.Time) Token {
		tok := Token{ID: uuid.New(), UserID: owner, TokenHash: uuid.NewString(), ExpiresAt: expires}
		if err := st.CreateToken(ctx, tok); err != nil {
			t.Fatal(err)
		}
		return tok
	}
	gone := func(tok Token) bool {
		_, err := st.GetTokenByHash(ctx, tok.TokenHash)
		if err == nil {
			return false
		}
		wantCode(t, err, domain.CodeNotFound)
		return true
	}

	a := mk(user, now.Add(time.Hour))
	got, err := st.GetTokenByHash(ctx, a.TokenHash)
	if err != nil || got.ID != a.ID || got.UserID != user || !got.ExpiresAt.Equal(a.ExpiresAt) {
		t.Fatalf("token = %+v, %v", got, err)
	}
	if err := st.DeleteTokenByHash(ctx, a.TokenHash); err != nil {
		t.Fatal(err)
	}
	if !gone(a) {
		t.Fatal("token survived DeleteTokenByHash")
	}

	keep, drop, theirs := mk(user, now.Add(time.Hour)), mk(user, now.Add(time.Hour)), mk(other, now.Add(time.Hour))
	if err := st.DeleteOtherTokens(ctx, user, keep.TokenHash); err != nil {
		t.Fatal(err)
	}
	if gone(keep) || !gone(drop) || gone(theirs) {
		t.Fatal("DeleteOtherTokens must drop only the user's other sessions")
	}
	if err := st.DeleteTokensByUser(ctx, user); err != nil {
		t.Fatal(err)
	}
	if !gone(keep) || gone(theirs) {
		t.Fatal("DeleteTokensByUser must drop every session of that user only")
	}

	// Expiry at exactly now counts as expired.
	past, edge, future := mk(user, now.Add(-time.Second)), mk(user, now), mk(user, now.Add(time.Second))
	if _, err := st.DeleteExpiredTokens(ctx, now); err != nil {
		t.Fatal(err)
	}
	if !gone(past) || !gone(edge) || gone(future) {
		t.Fatal("DeleteExpiredTokens must drop tokens with expires_at <= now")
	}
}

func contractChallenges(t *testing.T, st Store) {
	ctx := context.Background()
	now := contractT0
	email := contractEmail()

	_, err := st.GetEmailChallenge(ctx, email, domain.PurposeRegister)
	wantCode(t, err, domain.CodeNotFound)

	ch := EmailChallenge{
		Email: email, Purpose: domain.PurposeRegister, CodeHash: "hash-1",
		ExpiresAt: now.Add(domain.OTPTTL), SentAt: now, SendCount: 1, SendWindowStart: now,
	}
	if err := st.UpsertEmailChallenge(ctx, ch); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetEmailChallenge(ctx, email, domain.PurposeRegister)
	if err != nil || got.CodeHash != "hash-1" || got.AttemptCount != 0 || got.SendCount != 1 ||
		got.UserID != uuid.Nil || !got.ExpiresAt.Equal(ch.ExpiresAt) || !got.SentAt.Equal(now) ||
		!got.SendWindowStart.Equal(now) {
		t.Fatalf("challenge = %+v, %v", got, err)
	}
	// Same email, other purpose: a separate challenge.
	_, err = st.GetEmailChallenge(ctx, email, domain.PurposePasswordReset)
	wantCode(t, err, domain.CodeNotFound)

	// Guesses are counted one at a time and stop at the cap.
	for want := 1; want <= 3; want++ {
		got, err := st.ReserveChallengeGuess(ctx, email, domain.PurposeRegister, uuid.Nil, 3)
		if err != nil || got.AttemptCount != want || got.CodeHash != "hash-1" {
			t.Fatalf("guess %d = %+v, %v", want, got, err)
		}
	}
	_, err = st.ReserveChallengeGuess(ctx, email, domain.PurposeRegister, uuid.Nil, 3)
	wantCode(t, err, domain.CodeNotFound)

	// A resend replaces the code and resets the guesses.
	ch.CodeHash, ch.SendCount = "hash-2", 2
	if err := st.UpsertEmailChallenge(ctx, ch); err != nil {
		t.Fatal(err)
	}
	got, err = st.GetEmailChallenge(ctx, email, domain.PurposeRegister)
	if err != nil || got.CodeHash != "hash-2" || got.AttemptCount != 0 || got.SendCount != 2 {
		t.Fatalf("after resend = %+v, %v", got, err)
	}

	// Only the code that was compared can be consumed, and only once.
	if ok, err := st.ConsumeEmailChallenge(ctx, email, domain.PurposeRegister, "hash-1"); err != nil || ok {
		t.Fatalf("consume with a replaced code = %v, %v", ok, err)
	}
	if ok, err := st.ConsumeEmailChallenge(ctx, email, domain.PurposeRegister, "hash-2"); err != nil || !ok {
		t.Fatalf("consume = %v, %v", ok, err)
	}
	if ok, err := st.ConsumeEmailChallenge(ctx, email, domain.PurposeRegister, "hash-2"); err != nil || ok {
		t.Fatalf("second consume = %v, %v", ok, err)
	}
	_, err = st.GetEmailChallenge(ctx, email, domain.PurposeRegister)
	wantCode(t, err, domain.CodeNotFound)

	// A change_email challenge is bound to the account that asked.
	owner, stranger := contractUser(t, st), contractUser(t, st)
	bound := EmailChallenge{
		Email: contractEmail(), Purpose: domain.PurposeChangeEmail, CodeHash: "hash-3",
		ExpiresAt: now.Add(domain.OTPTTL), SentAt: now, SendCount: 1, SendWindowStart: now, UserID: owner,
	}
	if err := st.UpsertEmailChallenge(ctx, bound); err != nil {
		t.Fatal(err)
	}
	_, err = st.ReserveChallengeGuess(ctx, bound.Email, bound.Purpose, stranger, 5)
	wantCode(t, err, domain.CodeNotFound)
	_, err = st.ReserveChallengeGuess(ctx, bound.Email, bound.Purpose, uuid.Nil, 5)
	wantCode(t, err, domain.CodeNotFound)
	got, err = st.ReserveChallengeGuess(ctx, bound.Email, bound.Purpose, owner, 5)
	if err != nil || got.AttemptCount != 1 || got.UserID != owner {
		t.Fatalf("owner guess = %+v, %v (the refused guesses must not count)", got, err)
	}

	// The sweep cutoff is exclusive: a challenge that expired exactly at it stays.
	old := EmailChallenge{
		Email: contractEmail(), Purpose: domain.PurposePasswordReset, CodeHash: "old",
		ExpiresAt: now.Add(-time.Second), SentAt: now.Add(-time.Hour), SendCount: 1, SendWindowStart: now.Add(-time.Hour),
	}
	edge := old
	edge.Email, edge.ExpiresAt = contractEmail(), now
	for _, c := range []EmailChallenge{old, edge} {
		if err := st.UpsertEmailChallenge(ctx, c); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := st.DeleteEmailChallengesExpiredBefore(ctx, now); err != nil {
		t.Fatal(err)
	}
	_, err = st.GetEmailChallenge(ctx, old.Email, old.Purpose)
	wantCode(t, err, domain.CodeNotFound)
	if _, err := st.GetEmailChallenge(ctx, edge.Email, edge.Purpose); err != nil {
		t.Fatalf("challenge expiring at the cutoff was swept: %v", err)
	}
}

func contractProjects(t *testing.T, st Store) {
	ctx := context.Background()
	user := contractUser(t, st)
	other := contractUser(t, st)

	progress := 40
	created, err := st.CreateProject(ctx, user, domain.Project{
		ID: uuid.NewString(), Name: "bravo", Color: "#3b82f6", Code: strPtr("AUTH"), ProgressPercent: &progress,
	})
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.MustParse(created.ID)
	got, err := st.GetProject(ctx, user, id)
	if err != nil || got.Name != "bravo" || got.Color != "#3b82f6" || got.Code == nil || *got.Code != "AUTH" ||
		got.ProgressPercent == nil || *got.ProgressPercent != 40 || got.Archived {
		t.Fatalf("project = %+v, %v", got, err)
	}

	// Rows are scoped by account.
	_, err = st.GetProject(ctx, other, id)
	wantCode(t, err, domain.CodeNotFound)
	_, err = st.UpdateProject(ctx, other, got)
	wantCode(t, err, domain.CodeNotFound)
	_, err = st.GetProject(ctx, user, uuid.New())
	wantCode(t, err, domain.CodeNotFound)

	// A code is unique per account, ignoring case; other accounts may reuse it.
	_, err = st.CreateProject(ctx, user, domain.Project{ID: uuid.NewString(), Name: "dup", Color: "#000000", Code: strPtr("auth")})
	wantCode(t, err, domain.CodeCodeInUse)
	contractProject(t, st, other, "theirs", strPtr("AUTH"))
	if used, err := st.CodeInUse(ctx, user, "Auth", uuid.Nil); err != nil || !used {
		t.Fatalf("CodeInUse = %v, %v", used, err)
	}
	if used, err := st.CodeInUse(ctx, user, "AUTH", id); err != nil || used {
		t.Fatalf("CodeInUse excluding the owner = %v, %v", used, err)
	}
	if used, err := st.CodeInUse(ctx, user, "FREE", uuid.Nil); err != nil || used {
		t.Fatalf("CodeInUse for a free code = %v, %v", used, err)
	}

	alpha := contractProject(t, st, user, "alpha", nil)
	second := contractProject(t, st, user, "charlie", strPtr("OPS"))
	second.Code = strPtr("auth")
	_, err = st.UpdateProject(ctx, user, second)
	wantCode(t, err, domain.CodeCodeInUse)

	// Update writes every field; nil clears code and progress.
	got.Name, got.Color, got.Code, got.ProgressPercent, got.Archived = "bravo 2", "#ffffff", nil, nil, true
	updated, err := st.UpdateProject(ctx, user, got)
	if err != nil || updated.Name != "bravo 2" || updated.Color != "#ffffff" || updated.Code != nil ||
		updated.ProgressPercent != nil || !updated.Archived {
		t.Fatalf("updated = %+v, %v", updated, err)
	}
	// Several projects may have no code.
	contractProject(t, st, user, "delta", nil)

	active, err := st.ListProjects(ctx, user, false)
	if err != nil || len(active) != 3 || active[0].ID != alpha.ID || active[1].Name != "charlie" || active[2].Name != "delta" {
		t.Fatalf("active projects by name = %+v, %v", active, err)
	}
	all, err := st.ListProjects(ctx, user, true)
	if err != nil || len(all) != 4 || all[1].Name != "bravo 2" {
		t.Fatalf("all projects by name = %+v, %v", all, err)
	}
	if n, err := st.CountActiveProjects(ctx, user); err != nil || n != 3 {
		t.Fatalf("CountActiveProjects = %d, %v", n, err)
	}

	if n, err := st.CountProjectSessions(ctx, user, id); err != nil || n != 0 {
		t.Fatalf("CountProjectSessions = %d, %v", n, err)
	}
	for i := 0; i < 2; i++ {
		if _, err := st.CreateSession(ctx, user, contractStopped(alpha.ID, contractT0.Add(time.Duration(i)*time.Hour), time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	if n, err := st.CountProjectSessions(ctx, user, uuid.MustParse(alpha.ID)); err != nil || n != 2 {
		t.Fatalf("CountProjectSessions = %d, %v", n, err)
	}

	if err := st.DeleteProject(ctx, user, id); err != nil {
		t.Fatal(err)
	}
	_, err = st.GetProject(ctx, user, id)
	wantCode(t, err, domain.CodeNotFound)
}

func contractActivityTypes(t *testing.T, st Store) {
	ctx := context.Background()
	user := contractUser(t, st)
	other := contractUser(t, st)

	mk := func(owner uuid.UUID, name string) domain.ActivityType {
		a, err := st.CreateActivityType(ctx, owner, domain.ActivityType{ID: uuid.NewString(), Name: name, Color: "#112233"})
		if err != nil {
			t.Fatalf("create activity type %q: %v", name, err)
		}
		return a
	}
	review := mk(user, "Review")
	mk(user, "coding")
	mk(user, "Admin")
	mk(other, "Review")
	id := uuid.MustParse(review.ID)

	got, err := st.GetActivityType(ctx, user, id)
	if err != nil || got != review {
		t.Fatalf("activity type = %+v, %v", got, err)
	}
	_, err = st.GetActivityType(ctx, other, id)
	wantCode(t, err, domain.CodeNotFound)

	// Listed by name, ignoring case.
	list, err := st.ListActivityTypes(ctx, user)
	if err != nil || len(list) != 3 || list[0].Name != "Admin" || list[1].Name != "coding" || list[2].Name != "Review" {
		t.Fatalf("list = %+v, %v", list, err)
	}

	// A name is unique per account, ignoring case.
	_, err = st.CreateActivityType(ctx, user, domain.ActivityType{ID: uuid.NewString(), Name: "REVIEW", Color: "#000000"})
	wantCode(t, err, domain.CodeNameInUse)
	if used, err := st.ActivityTypeNameInUse(ctx, user, "review", uuid.Nil); err != nil || !used {
		t.Fatalf("NameInUse = %v, %v", used, err)
	}
	if used, err := st.ActivityTypeNameInUse(ctx, user, "review", id); err != nil || used {
		t.Fatalf("NameInUse excluding the owner = %v, %v", used, err)
	}
	review.Name = "CODING"
	_, err = st.UpdateActivityType(ctx, user, review)
	wantCode(t, err, domain.CodeNameInUse)

	review.Name, review.Color = "Code review", "#abcdef"
	updated, err := st.UpdateActivityType(ctx, user, review)
	if err != nil || updated != review {
		t.Fatalf("updated = %+v, %v", updated, err)
	}
	_, err = st.UpdateActivityType(ctx, other, review)
	wantCode(t, err, domain.CodeNotFound)

	project := contractProject(t, st, user, "P", nil)
	with := contractStopped(project.ID, contractT0, time.Hour)
	with.ActivityTypeID = &review.ID
	without := contractStopped(project.ID, contractT0.Add(2*time.Hour), time.Hour)
	for _, s := range []domain.Session{with, without} {
		if _, err := st.CreateSession(ctx, user, s); err != nil {
			t.Fatal(err)
		}
	}
	if n, err := st.CountActivityTypeSessions(ctx, user, id); err != nil || n != 1 {
		t.Fatalf("CountActivityTypeSessions = %d, %v", n, err)
	}

	if err := st.DeleteSession(ctx, user, uuid.MustParse(with.ID)); err != nil {
		t.Fatal(err)
	}
	if err := st.DeleteActivityType(ctx, user, id); err != nil {
		t.Fatal(err)
	}
	_, err = st.GetActivityType(ctx, user, id)
	wantCode(t, err, domain.CodeNotFound)
}

func contractSessions(t *testing.T, st Store) {
	ctx := context.Background()
	user := contractUser(t, st)
	other := contractUser(t, st)
	project := contractProject(t, st, user, "P", nil)
	second := contractProject(t, st, user, "Q", nil)
	otherProject := contractProject(t, st, other, "P", nil)

	if _, live, err := st.GetLiveSession(ctx, user); err != nil || live {
		t.Fatalf("live session on a new account = %v, %v", live, err)
	}

	live := contractLive(project.ID, contractT0)
	created, err := st.CreateSession(ctx, user, live)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != live.ID || created.ProjectID != project.ID || created.Note != "live" ||
		created.TicketID != nil || created.ActivityTypeID != nil || created.Status != domain.StatusActive ||
		!created.StartedAt.Equal(contractT0) || created.EndedAt != nil {
		t.Fatalf("created = %+v", created)
	}
	id := uuid.MustParse(live.ID)

	// One live session per account; stopped ones and other accounts are unaffected.
	_, err = st.CreateSession(ctx, user, contractLive(project.ID, contractT0.Add(time.Minute)))
	wantCode(t, err, domain.CodeSessionAlreadyActive)
	if _, err := st.CreateSession(ctx, user, contractStopped(project.ID, contractT0.Add(-2*time.Hour), time.Hour)); err != nil {
		t.Fatalf("manual entry beside a live session: %v", err)
	}
	if _, err := st.CreateSession(ctx, other, contractLive(otherProject.ID, contractT0)); err != nil {
		t.Fatalf("live session of another account: %v", err)
	}

	got, ok, err := st.GetLiveSession(ctx, user)
	if err != nil || !ok || got.ID != live.ID {
		t.Fatalf("live = %+v, %v, %v", got, ok, err)
	}
	_, err = st.GetSession(ctx, other, id)
	wantCode(t, err, domain.CodeNotFound)
	_, err = st.GetSession(ctx, user, uuid.New())
	wantCode(t, err, domain.CodeNotFound)

	// Update writes every field, with microsecond instants kept exactly.
	typ, err := st.CreateActivityType(ctx, user, domain.ActivityType{ID: uuid.NewString(), Name: "Deep", Color: "#112233"})
	if err != nil {
		t.Fatal(err)
	}
	edit := got
	edit.ProjectID, edit.Note, edit.TicketID, edit.ActivityTypeID = second.ID, "edited ✓", strPtr("EMI-1"), &typ.ID
	edit.StartedAt = contractT0.Add(-time.Microsecond)
	updated, err := st.UpdateSession(ctx, user, edit)
	if err != nil || updated.ProjectID != second.ID || updated.Note != "edited ✓" ||
		updated.TicketID == nil || *updated.TicketID != "EMI-1" ||
		updated.ActivityTypeID == nil || *updated.ActivityTypeID != typ.ID ||
		!updated.StartedAt.Equal(edit.StartedAt) {
		t.Fatalf("updated = %+v, %v", updated, err)
	}
	_, err = st.UpdateSession(ctx, other, edit)
	wantCode(t, err, domain.CodeNotFound)
	edit.TicketID, edit.ActivityTypeID = nil, nil
	if cleared, err := st.UpdateSession(ctx, user, edit); err != nil || cleared.TicketID != nil || cleared.ActivityTypeID != nil {
		t.Fatalf("cleared = %+v, %v", cleared, err)
	}

	// A transition is written only while the stored status is the one that was read.
	stopped, err := domain.Stop(edit, contractT0.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	done, err := st.TransitionSession(ctx, user, stopped, domain.StatusActive)
	if err != nil || done.Status != domain.StatusStopped || done.EndedAt == nil || !done.EndedAt.Equal(contractT0.Add(time.Hour)) {
		t.Fatalf("transition = %+v, %v", done, err)
	}
	_, err = st.TransitionSession(ctx, user, stopped, domain.StatusActive)
	wantCode(t, err, domain.CodeInvalidTransition)
	missing := stopped
	missing.ID = uuid.NewString()
	_, err = st.TransitionSession(ctx, user, missing, domain.StatusActive)
	wantCode(t, err, domain.CodeNotFound)
	_, err = st.TransitionSession(ctx, other, stopped, domain.StatusStopped)
	wantCode(t, err, domain.CodeNotFound)

	// With the timer stopped a new one can start.
	if _, live, err := st.GetLiveSession(ctx, user); err != nil || live {
		t.Fatalf("live after stop = %v, %v", live, err)
	}
	if _, err := st.CreateSession(ctx, user, contractLive(project.ID, contractT0.Add(2*time.Hour))); err != nil {
		t.Fatalf("start after stop: %v", err)
	}

	if err := st.DeleteSession(ctx, user, id); err != nil {
		t.Fatal(err)
	}
	_, err = st.GetSession(ctx, user, id)
	wantCode(t, err, domain.CodeNotFound)
}

func contractSessionList(t *testing.T, st Store) {
	ctx := context.Background()
	user := contractUser(t, st)
	project := contractProject(t, st, user, "P", nil)
	other := contractUser(t, st)
	otherProject := contractProject(t, st, other, "P", nil)
	if _, err := st.CreateSession(ctx, other, contractStopped(otherProject.ID, contractT0, time.Hour)); err != nil {
		t.Fatal(err)
	}

	empty, err := st.ListSessions(ctx, user, nil, SessionWindow{}, 20, "")
	if err != nil || empty.Items == nil || len(empty.Items) != 0 || empty.NextCursor != nil {
		t.Fatalf("empty page = %+v, %v (items must be an empty list, not nil)", empty, err)
	}

	// Hours 0..4 hold one stopped one-hour session each; hour 2 holds three that
	// start at the same instant, so paging has to break the tie by id.
	var want []string
	at := func(h int) time.Time { return contractT0.Add(time.Duration(h) * time.Hour) }
	add := func(s domain.Session) {
		if _, err := st.CreateSession(ctx, user, s); err != nil {
			t.Fatal(err)
		}
	}
	for h := 0; h <= 4; h++ {
		add(contractStopped(project.ID, at(h), time.Hour))
	}
	add(contractStopped(project.ID, at(2), time.Hour))
	add(contractStopped(project.ID, at(2), time.Hour))
	live := contractLive(project.ID, at(6))
	add(live)

	all, err := st.ListSessions(ctx, user, nil, SessionWindow{}, 100, "")
	if err != nil || len(all.Items) != 8 || all.NextCursor != nil {
		t.Fatalf("one page = %d items, cursor %v, %v", len(all.Items), all.NextCursor, err)
	}
	for i, s := range all.Items {
		want = append(want, s.ID)
		if i == 0 {
			continue
		}
		prev := all.Items[i-1]
		if s.StartedAt.After(prev.StartedAt) || (s.StartedAt.Equal(prev.StartedAt) && s.ID >= prev.ID) {
			t.Fatalf("order at %d: %s %s after %s %s; want started_at DESC, id DESC",
				i, s.StartedAt, s.ID, prev.StartedAt, prev.ID)
		}
	}
	if want[0] != live.ID {
		t.Fatalf("newest = %s, want the live session", want[0])
	}

	// Pages of 3 walk the same order with no row repeated or skipped.
	var paged []string
	cursor := ""
	for page := 0; ; page++ {
		if page > 5 {
			t.Fatal("paging did not end")
		}
		res, err := st.ListSessions(ctx, user, nil, SessionWindow{}, 3, cursor)
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range res.Items {
			paged = append(paged, s.ID)
		}
		if res.NextCursor == nil {
			break
		}
		if len(res.Items) != 3 {
			t.Fatalf("page %d has %d items and a next cursor", page, len(res.Items))
		}
		cursor = *res.NextCursor
	}
	if len(paged) != len(want) {
		t.Fatalf("paged %d sessions, want %d", len(paged), len(want))
	}
	for i := range want {
		if paged[i] != want[i] {
			t.Fatalf("paged[%d] = %s, want %s", i, paged[i], want[i])
		}
	}

	_, err = st.ListSessions(ctx, user, nil, SessionWindow{}, 3, "not-a-cursor")
	wantCode(t, err, domain.CodeInvalidQuery)

	count := func(statuses []string, w SessionWindow) int {
		t.Helper()
		res, err := st.ListSessions(ctx, user, statuses, w, 100, "")
		if err != nil {
			t.Fatal(err)
		}
		return len(res.Items)
	}
	if n := count([]string{domain.StatusActive}, SessionWindow{}); n != 1 {
		t.Fatalf("active = %d, want 1", n)
	}
	if n := count([]string{domain.StatusStopped}, SessionWindow{}); n != 7 {
		t.Fatalf("stopped = %d, want 7", n)
	}
	if n := count([]string{domain.StatusActive, domain.StatusStopped}, SessionWindow{}); n != 8 {
		t.Fatalf("both statuses = %d, want 8", n)
	}

	// The window is [from, to) by overlap: endedAt > from (or live), startedAt < to.
	from, to := at(1), at(3)
	if n := count(nil, SessionWindow{From: &from, To: &to}); n != 4 {
		// Hour 0 ends exactly at from (out); hours 1 and 2 (x3) are in; hour 3 starts at to (out).
		t.Fatalf("window [1h, 3h) = %d, want 4", n)
	}
	if n := count(nil, SessionWindow{From: &from}); n != 7 {
		t.Fatalf("from 1h = %d, want 7 (the live session overlaps every from)", n)
	}
	if n := count(nil, SessionWindow{To: &to}); n != 5 {
		t.Fatalf("to 3h = %d, want 5", n)
	}
	late := at(100)
	if n := count(nil, SessionWindow{From: &late}); n != 1 {
		t.Fatalf("from far future = %d, want only the live session", n)
	}

	// Day totals read stopped sessions with from <= startedAt < to, oldest first.
	between, err := st.ListStoppedSessionsStartedBetween(ctx, user, at(2), at(4))
	if err != nil || len(between) != 4 {
		t.Fatalf("started in [2h, 4h) = %d, %v; want 4", len(between), err)
	}
	for i := 1; i < len(between); i++ {
		a, b := between[i-1], between[i]
		if b.StartedAt.Before(a.StartedAt) || (b.StartedAt.Equal(a.StartedAt) && b.ID <= a.ID) {
			t.Fatalf("order at %d; want started_at ASC, id ASC", i)
		}
	}
	none, err := st.ListStoppedSessionsStartedBetween(ctx, user, at(6), at(7))
	if err != nil || len(none) != 0 {
		t.Fatalf("the live session was returned as stopped: %+v, %v", none, err)
	}
}
