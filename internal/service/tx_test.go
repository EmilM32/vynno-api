package service

import (
	"context"
	"errors"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/EmilM32/vynno-api/internal/mail"
	"github.com/EmilM32/vynno-api/internal/store"
	"github.com/EmilM32/vynno-api/internal/store/storetest"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// onEachStore runs fn against the Memory double and, when TEST_DATABASE_URL is
// set, against Postgres: the transaction and lock rules below are only real there.
func onEachStore(t *testing.T, fn func(t *testing.T, st store.Store)) {
	t.Run("memory", func(t *testing.T) { fn(t, store.NewEmptyMemory()) })
	t.Run("postgres", func(t *testing.T) {
		pg, _ := storetest.NewPostgres(t)
		fn(t, pg)
	})
}

var errInjected = errors.New("injected store failure")

// faultStore fails the named operations, also inside InTx.
type faultStore struct {
	store.Store
	fail map[string]bool
}

func (f faultStore) InTx(ctx context.Context, fn func(tx store.Store) error) error {
	return f.Store.InTx(ctx, func(tx store.Store) error {
		return fn(faultStore{Store: tx, fail: f.fail})
	})
}

func (f faultStore) CreateProject(ctx context.Context, userID uuid.UUID, p domain.Project) (domain.Project, error) {
	if f.fail["CreateProject"] {
		return domain.Project{}, errInjected
	}
	return f.Store.CreateProject(ctx, userID, p)
}

func (f faultStore) DeleteTokensByUser(ctx context.Context, userID uuid.UUID) error {
	if f.fail["DeleteTokensByUser"] {
		return errInjected
	}
	return f.Store.DeleteTokensByUser(ctx, userID)
}

func (f faultStore) DeleteOtherTokens(ctx context.Context, userID uuid.UUID, keepHash string) error {
	if f.fail["DeleteOtherTokens"] {
		return errInjected
	}
	return f.Store.DeleteOtherTokens(ctx, userID, keepHash)
}

var sixDigits = regexp.MustCompile(`\b\d{6}\b`)

func lastCode(t *testing.T, rec *mail.Recorder) string {
	t.Helper()
	if len(rec.Messages) == 0 {
		t.Fatal("no mail was sent")
	}
	code := sixDigits.FindString(rec.Messages[len(rec.Messages)-1].Text)
	if code == "" {
		t.Fatal("no code in the last mail")
	}
	return code
}

func testEmail() string { return uuid.NewString() + "@example.com" }

// txAccount stores an account with a profile and one project, and returns its id.
func txAccount(t *testing.T, st store.Store, email, password string) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.New()
	if err := st.CreateAccount(ctx, store.Account{ID: id, Email: email, PasswordHash: string(hash)}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateProfile(ctx, id, domain.Profile{}); err != nil {
		t.Fatal(err)
	}
	return id
}

func txProject(t *testing.T, svc *Service, name string) domain.Project {
	t.Helper()
	p, err := svc.CreateProject(context.Background(), CreateProjectInput{Name: name, Color: "#3b82f6"})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// A register that fails after the account row was written leaves no account: the
// address is free and a retry with a fresh code works.
func TestRegisterIsAllOrNothing(t *testing.T) {
	onEachStore(t, func(t *testing.T, st store.Store) {
		ctx := context.Background()
		rec := &mail.Recorder{}
		now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
		clock := func() time.Time { return now }
		email := testEmail()

		broken := New(faultStore{Store: st, fail: map[string]bool{"CreateProject": true}}, rec)
		broken.Now = clock
		if err := broken.RequestRegisterCode(ctx, email, "203.0.113.1"); err != nil {
			t.Fatal(err)
		}
		_, err := broken.Register(ctx, RegisterInput{Email: email, Password: "long-enough-pw", Code: lastCode(t, rec)}, "203.0.113.9")
		if !errors.Is(err, errInjected) {
			t.Fatalf("register = %v, want the injected failure", err)
		}
		if taken, err := st.EmailTaken(ctx, email, uuid.Nil); err != nil || taken {
			t.Fatalf("EmailTaken after a failed register = %v, %v; the account must be rolled back", taken, err)
		}

		svc := New(st, rec)
		svc.Now = clock
		now = now.Add(2 * domain.OTPSendCooldown)
		if err := svc.RequestRegisterCode(ctx, email, "203.0.113.1"); err != nil {
			t.Fatalf("new code after the failure: %v", err)
		}
		res, err := svc.Register(ctx, RegisterInput{Email: email, Password: "long-enough-pw", Code: lastCode(t, rec)}, "203.0.113.9")
		if err != nil {
			t.Fatalf("retry: %v", err)
		}
		userID, err := svc.ResolveToken(ctx, res.Token)
		if err != nil {
			t.Fatal(err)
		}
		projects, err := svc.ForUser(userID).ListProjects(ctx, false)
		if err != nil || len(projects) != 1 || projects[0].Name != "Personal" {
			t.Fatalf("projects after register = %+v, %v", projects, err)
		}
	})
}

// A password reset, password change, or email change whose sign-out step fails
// changes nothing: the old credentials still work and no session is lost.
func TestCredentialChangesAreAllOrNothing(t *testing.T) {
	onEachStore(t, func(t *testing.T, st store.Store) {
		ctx := context.Background()
		rec := &mail.Recorder{}
		now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
		const oldPW, newPW, ip = "old-password", "new-password", "203.0.113.2"
		email := testEmail()
		userID := txAccount(t, st, email, oldPW)

		good := New(st, rec)
		broken := New(faultStore{Store: st, fail: map[string]bool{
			"DeleteTokensByUser": true, "DeleteOtherTokens": true,
		}}, rec)
		for _, s := range []*Service{good, broken} {
			s.Now = func() time.Time { return now }
		}
		login, err := good.Login(ctx, LoginInput{Email: email, Password: oldPW}, ip)
		if err != nil {
			t.Fatal(err)
		}
		unchanged := func(step string) {
			t.Helper()
			acc, err := st.GetAccountByID(ctx, userID)
			if err != nil || acc.Email != email ||
				bcrypt.CompareHashAndPassword([]byte(acc.PasswordHash), []byte(oldPW)) != nil {
				t.Fatalf("%s: credentials changed although the request failed (%+v, %v)", step, acc.Email, err)
			}
			if _, err := good.ResolveToken(ctx, login.Token); err != nil {
				t.Fatalf("%s: session lost: %v", step, err)
			}
		}

		if err := broken.RequestPasswordReset(ctx, email, ip); err != nil {
			t.Fatal(err)
		}
		err = broken.ResetPassword(ctx, ResetPasswordInput{Email: email, Code: lastCode(t, rec), Password: newPW}, "203.0.113.9")
		if !errors.Is(err, errInjected) {
			t.Fatalf("reset = %v", err)
		}
		unchanged("reset")

		err = broken.ForUser(userID).ChangePassword(ctx, ChangePasswordInput{CurrentPassword: oldPW, NewPassword: newPW}, login.Token, ip)
		if !errors.Is(err, errInjected) {
			t.Fatalf("change password = %v", err)
		}
		unchanged("change password")

		newEmail := testEmail()
		if err := broken.ForUser(userID).RequestEmailChange(ctx, RequestEmailChangeInput{Email: newEmail, Password: oldPW}, ip); err != nil {
			t.Fatal(err)
		}
		_, err = broken.ForUser(userID).ChangeEmail(ctx, ChangeEmailInput{Email: newEmail, Code: lastCode(t, rec)}, login.Token, ip)
		if !errors.Is(err, errInjected) {
			t.Fatalf("change email = %v", err)
		}
		unchanged("change email")

		// The same reset through a healthy store signs every session out.
		now = now.Add(2 * domain.OTPSendCooldown)
		if err := good.RequestPasswordReset(ctx, email, ip); err != nil {
			t.Fatal(err)
		}
		if err := good.ResetPassword(ctx, ResetPasswordInput{Email: email, Code: lastCode(t, rec), Password: newPW}, "203.0.113.9"); err != nil {
			t.Fatal(err)
		}
		if _, err := good.ResolveToken(ctx, login.Token); codeOf(err) != domain.CodeUnauthorized {
			t.Fatalf("session after reset = %v, want unauthorized", err)
		}
		if _, err := good.Login(ctx, LoginInput{Email: email, Password: newPW}, ip); err != nil {
			t.Fatalf("login with the new password: %v", err)
		}
	})
}

// Two archives of the last two active projects at once: one wins, the other is
// last_active_project. Before the account lock both could count two and proceed.
func TestConcurrentArchiveKeepsOneActiveProject(t *testing.T) {
	onEachStore(t, func(t *testing.T, st store.Store) {
		ctx := context.Background()
		for round := 0; round < 15; round++ {
			svc := New(st, nil).ForUser(txAccount(t, st, testEmail(), "password-1"))
			projects := []domain.Project{txProject(t, svc, "a"), txProject(t, svc, "b")}
			errs := make([]error, len(projects))
			var wg sync.WaitGroup
			for i, p := range projects {
				wg.Add(1)
				go func() {
					defer wg.Done()
					_, errs[i] = svc.ArchiveProject(ctx, uuid.MustParse(p.ID))
				}()
			}
			wg.Wait()

			active, err := svc.ListProjects(ctx, false)
			if err != nil {
				t.Fatal(err)
			}
			if len(active) != 1 {
				t.Fatalf("round %d: %d active projects, want 1 (errors: %v)", round, len(active), errs)
			}
			var refused int
			for _, err := range errs {
				if codeOf(err) == domain.CodeLastActiveProject {
					refused++
				} else if err != nil {
					t.Fatalf("round %d: %v", round, err)
				}
			}
			if refused != 1 {
				t.Fatalf("round %d: %d archives refused, want 1", round, refused)
			}
		}
	})
}

// An edit that races a stop must not write the session back as running. Before
// the account lock the edit could store the row it read before the stop.
func TestConcurrentStopAndEditKeepsSessionStopped(t *testing.T) {
	onEachStore(t, func(t *testing.T, st store.Store) {
		ctx := context.Background()
		svc := New(st, nil).ForUser(txAccount(t, st, testEmail(), "password-1"))
		project := txProject(t, svc, "p")

		for round := 0; round < 25; round++ {
			sess, err := svc.StartSession(ctx, StartSessionInput{ProjectID: project.ID, Note: "before"})
			if err != nil {
				t.Fatalf("round %d start: %v", round, err)
			}
			id := uuid.MustParse(sess.ID)
			note := "after"
			var stopErr, editErr error
			var wg sync.WaitGroup
			wg.Add(2)
			go func() { defer wg.Done(); _, stopErr = svc.StopSession(ctx, id) }()
			go func() { defer wg.Done(); _, editErr = svc.UpdateSession(ctx, id, domain.SessionPatch{Note: &note}) }()
			wg.Wait()
			if stopErr != nil || editErr != nil {
				t.Fatalf("round %d: stop %v, edit %v", round, stopErr, editErr)
			}

			got, err := svc.GetSession(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != domain.StatusStopped || got.EndedAt == nil {
				t.Fatalf("round %d: session is %s with endedAt %v after a stop", round, got.Status, got.EndedAt)
			}
			if got.Note != "after" {
				t.Fatalf("round %d: note = %q, the edit was lost", round, got.Note)
			}
		}
	})
}
