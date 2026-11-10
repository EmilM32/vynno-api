package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/EmilM32/vynno-api/internal/mail"
	"github.com/EmilM32/vynno-api/internal/store"
	"github.com/google/uuid"
)

// hardeningService is a service on Memory with a recorder and a clock the test moves.
func hardeningService(t *testing.T) (*Service, *store.Memory, *mail.Recorder, *time.Time) {
	t.Helper()
	mem := store.NewEmptyMemory()
	rec := &mail.Recorder{}
	svc := New(mem, rec)
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	svc.Now = func() time.Time { return now }
	return svc, mem, rec, &now
}

func wrongCode(right string) string {
	if right == "000000" {
		return "000001"
	}
	return "000000"
}

func retryAfter(t *testing.T, err error) time.Duration {
	t.Helper()
	de, ok := domain.AsError(err)
	if !ok || de.Code != domain.CodeRateLimited {
		t.Fatalf("error = %v, want rate_limited", err)
	}
	return de.RetryAfter
}

// A resend gives a new challenge with five fresh guesses. Wrong codes for one
// email are still capped across resends, and the cap also refuses the right code.
func TestCodeGuessesAreCappedPerEmailAcrossResends(t *testing.T) {
	ctx := context.Background()
	svc, mem, rec, now := hardeningService(t)
	const email = "victim@example.com"
	addAccount(t, mem, email, "old-password")

	failures := 0
	for failures < svc.Limits.GuessEmailFails {
		// Each round comes from a new address, so only the per-email cap is in play.
		ip := fmt.Sprintf("198.51.100.%d", failures)
		if err := svc.RequestPasswordReset(ctx, email, ip); err != nil {
			t.Fatalf("send after %d failures: %v", failures, err)
		}
		wrong := wrongCode(lastCode(t, rec))
		for i := 0; i < domain.OTPMaxAttempts-1 && failures < svc.Limits.GuessEmailFails; i++ {
			err := svc.ResetPassword(ctx, ResetPasswordInput{Email: email, Code: wrong, Password: "attacker-pw"}, ip)
			if codeOf(err) != domain.CodeInvalidCode {
				t.Fatalf("failure %d = %v, want invalid_code", failures, err)
			}
			failures++
		}
		*now = now.Add(domain.OTPSendCooldown + time.Second)
	}

	if err := svc.RequestPasswordReset(ctx, email, "198.51.100.200"); err != nil {
		t.Fatal(err)
	}
	right := lastCode(t, rec)
	err := svc.ResetPassword(ctx, ResetPasswordInput{Email: email, Code: right, Password: "attacker-pw"}, "198.51.100.201")
	if wait := retryAfter(t, err); wait <= 0 || wait > svc.Limits.GuessEmailWindow {
		t.Fatalf("Retry-After = %v, want within the %v window", wait, svc.Limits.GuessEmailWindow)
	}
	// The refused request did not use up a guess on the challenge itself.
	ch, err := mem.GetEmailChallenge(ctx, email, domain.PurposePasswordReset)
	if err != nil || ch.AttemptCount != 0 {
		t.Fatalf("challenge after a limited guess = %+v, %v", ch, err)
	}
	// Another email is not affected.
	other := "other@example.com"
	addAccount(t, mem, other, "old-password")
	if err := svc.RequestPasswordReset(ctx, other, "198.51.100.202"); err != nil {
		t.Fatal(err)
	}
	if err := svc.ResetPassword(ctx, ResetPasswordInput{Email: other, Code: lastCode(t, rec), Password: "new-password"}, "198.51.100.202"); err != nil {
		t.Fatalf("reset of another account: %v", err)
	}

	// Once the window has passed, a fresh code works.
	*now = now.Add(svc.Limits.GuessEmailWindow + time.Minute)
	if err := svc.RequestPasswordReset(ctx, email, "198.51.100.203"); err != nil {
		t.Fatal(err)
	}
	if err := svc.ResetPassword(ctx, ResetPasswordInput{Email: email, Code: lastCode(t, rec), Password: "owner-new-pw"}, "198.51.100.203"); err != nil {
		t.Fatalf("reset after the window: %v", err)
	}
}

// One client guessing codes for many emails is stopped; a right code does not
// count against it, and another client is not affected.
func TestCodeGuessesAreCappedPerClient(t *testing.T) {
	ctx := context.Background()
	svc, _, _, _ := hardeningService(t)
	const attacker, bystander = "203.0.113.50", "203.0.113.51"
	relaxed := RelaxedAuthLimits()
	if relaxed.GuessIPFails <= svc.Limits.GuessIPFails || relaxed.GuessEmailFails != svc.Limits.GuessEmailFails {
		t.Fatalf("relaxed limits = %+v: only the per-client cap is lifted", relaxed)
	}
	// A right code hashes a password; a small cap keeps the test fast.
	svc.Limits.GuessIPFails = 4

	guess := func(ip string, n int, code func(right string) string) error {
		email := fmt.Sprintf("user%d-%s@example.com", n, strings.ReplaceAll(ip, ".", "-"))
		right, err := svc.issueOTPChallenge(ctx, email, domain.PurposeRegister, uuid.Nil)
		if err != nil {
			t.Fatal(err)
		}
		_, err = svc.Register(ctx, RegisterInput{Email: email, Password: "long-enough-pw", Code: code(right)}, ip)
		return err
	}
	same := func(right string) string { return right }

	// Successes never fill the window.
	for i := 0; i < svc.Limits.GuessIPFails+2; i++ {
		if err := guess(attacker, 1000+i, same); err != nil {
			t.Fatalf("right code %d: %v", i, err)
		}
	}
	for i := 0; i < svc.Limits.GuessIPFails; i++ {
		if err := guess(attacker, i, wrongCode); codeOf(err) != domain.CodeInvalidCode {
			t.Fatalf("wrong code %d = %v", i, err)
		}
	}
	retryAfter(t, guess(attacker, 5000, same))
	if err := guess(bystander, 1, same); err != nil {
		t.Fatalf("another client: %v", err)
	}
}

type failingMailer struct{}

func (failingMailer) Send(context.Context, mail.Message) error { return errors.New("smtp down") }

// Forgot must not tell a real account from an unknown address: not by the send
// cap, not by a mail failure, and not by waiting for the mail.
func TestPasswordForgotIsUniform(t *testing.T) {
	ctx := context.Background()
	svc, mem, rec, _ := hardeningService(t)
	const known, ip = "known@example.com", "203.0.113.60"
	addAccount(t, mem, known, "old-password")

	// Unknown addresses use up the per-client send cap like real ones.
	for i := 0; i < svc.Limits.SendIPLimit; i++ {
		if err := svc.RequestPasswordReset(ctx, fmt.Sprintf("nobody%d@example.com", i), ip); err != nil {
			t.Fatalf("unknown %d: %v", i, err)
		}
	}
	if len(rec.Messages) != 0 {
		t.Fatalf("mail sent to unknown addresses: %+v", rec.Messages)
	}
	retryAfter(t, svc.RequestPasswordReset(ctx, known, ip))
	retryAfter(t, svc.RequestPasswordReset(ctx, "nobody-else@example.com", ip))

	// A failing SMTP server is not reported for a real account.
	svc.Mailer = failingMailer{}
	if err := svc.RequestPasswordReset(ctx, known, "203.0.113.61"); err != nil {
		t.Fatalf("forgot with SMTP down = %v, want nil", err)
	}

	// With Async set the request returns before the mail is sent.
	svc.Mailer = rec
	var queued []func(context.Context)
	svc.Async = func(fn func(context.Context)) { queued = append(queued, fn) }
	const second = "second@example.com"
	addAccount(t, mem, second, "old-password")
	if err := svc.RequestPasswordReset(ctx, second, "203.0.113.62"); err != nil {
		t.Fatal(err)
	}
	if len(rec.Messages) != 0 || len(queued) != 1 {
		t.Fatalf("mail on the request path: %d sent, %d queued", len(rec.Messages), len(queued))
	}
	queued[0](ctx)
	if len(rec.Messages) != 1 || rec.Messages[0].To != second {
		t.Fatalf("queued mail = %+v", rec.Messages)
	}
}

func TestPasswordResetMailsNotice(t *testing.T) {
	ctx := context.Background()
	svc, mem, rec, _ := hardeningService(t)
	const email, ip = "owner@example.com", "203.0.113.70"
	addAccount(t, mem, email, "old-password")
	if err := svc.RequestPasswordReset(ctx, email, ip); err != nil {
		t.Fatal(err)
	}
	code := lastCode(t, rec)
	if err := svc.ResetPassword(ctx, ResetPasswordInput{Email: email, Code: code, Password: "new-password"}, ip); err != nil {
		t.Fatal(err)
	}
	last := rec.Messages[len(rec.Messages)-1]
	if last.To != email || last.Subject != "Your Vynno password was reset" || strings.Contains(last.Text, code) {
		t.Fatalf("notice = %+v", last)
	}

	// The reset has happened; a failing notice does not undo or fail it.
	svc.Mailer = failingMailer{}
	code2, err := svc.issueOTPChallenge(ctx, "second@example.com", domain.PurposePasswordReset, uuid.Nil)
	if err != nil {
		t.Fatal(err)
	}
	addAccount(t, mem, "second@example.com", "old-password")
	if err := svc.ResetPassword(ctx, ResetPasswordInput{Email: "second@example.com", Code: code2, Password: "new-password"}, ip); err != nil {
		t.Fatalf("reset with a failing notice = %v", err)
	}
}

type brokenAccounts struct{ store.Store }

func (brokenAccounts) GetAccountByEmail(context.Context, string) (store.Account, error) {
	return store.Account{}, errors.New("db down")
}

// A store failure during login is a server error, not "wrong password", and does
// not count as a failed attempt.
func TestLoginStoreFailureIsNotInvalidCredentials(t *testing.T) {
	ctx := context.Background()
	svc, mem, _, _ := hardeningService(t)
	const email, ip = "ada@example.com", "203.0.113.80"
	addAccount(t, mem, email, "right-password")

	svc.Store = brokenAccounts{mem}
	for i := 0; i < svc.Limits.LoginEmailFails+2; i++ {
		_, err := svc.Login(ctx, LoginInput{Email: email, Password: "right-password"}, ip)
		if err == nil || codeOf(err) != "" {
			t.Fatalf("attempt %d = %v, want a non-domain error", i, err)
		}
	}
	svc.Store = mem
	if _, err := svc.Login(ctx, LoginInput{Email: email, Password: "right-password"}, ip); err != nil {
		t.Fatalf("login after the outage: %v", err)
	}
}

// Auth events name what happened, the client, and the account id when known.
// They never contain an email, a password, or a code.
func TestAuthEventsAreLoggedWithoutSecrets(t *testing.T) {
	ctx := context.Background()
	svc, mem, rec, _ := hardeningService(t)
	const email, password, ip = "logged@example.com", "right-password", "203.0.113.90"
	addAccount(t, mem, email, password)
	acc, err := mem.GetAccountByEmail(ctx, email)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	_, _ = svc.Login(ctx, LoginInput{Email: email, Password: "wrong-password"}, ip)
	_, _ = svc.Login(ctx, LoginInput{Email: "nobody@example.com", Password: "wrong-password"}, ip)
	if _, err := svc.Login(ctx, LoginInput{Email: email, Password: password}, ip); err != nil {
		t.Fatal(err)
	}
	if err := svc.RequestPasswordReset(ctx, email, ip); err != nil {
		t.Fatal(err)
	}
	code := lastCode(t, rec)
	_ = svc.ResetPassword(ctx, ResetPasswordInput{Email: email, Code: wrongCode(code), Password: "new-password"}, ip)
	if err := svc.ResetPassword(ctx, ResetPasswordInput{Email: email, Code: code, Password: "new-password"}, ip); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	for _, want := range []string{
		"event=login_failed", "event=login ", "event=password_reset_requested ",
		"event=code_rejected", "event=password_reset ", "client_ip=" + ip, "user_id=" + acc.ID.String(),
	} {
		if !strings.Contains(out, want) {
			t.Errorf("log is missing %q:\n%s", want, out)
		}
	}
	for _, secret := range []string{email, "nobody@example.com", password, "wrong-password", "new-password", code} {
		if strings.Contains(out, secret) {
			t.Errorf("log contains %q:\n%s", secret, out)
		}
	}
}

func TestSweepDropsIdleLimiterKeys(t *testing.T) {
	ctx := context.Background()
	svc, mem, _, now := hardeningService(t)
	addAccount(t, mem, "ada@example.com", "right-password")
	for i := 0; i < 5; i++ {
		_, _ = svc.Login(ctx, LoginInput{Email: fmt.Sprintf("guess%d@example.com", i), Password: "wrong-password"}, "203.0.113.95")
	}
	if svc.Limiter.Len() == 0 {
		t.Fatal("no limiter keys after failed logins")
	}
	if _, err := svc.SweepExpired(ctx); err != nil {
		t.Fatal(err)
	}
	if svc.Limiter.Len() == 0 {
		t.Fatal("sweep dropped keys that are still inside a window")
	}
	*now = now.Add(svc.longestLimitWindow() + time.Minute)
	if _, err := svc.SweepExpired(ctx); err != nil {
		t.Fatal(err)
	}
	if n := svc.Limiter.Len(); n != 0 {
		t.Fatalf("%d limiter keys left after every window passed", n)
	}
}

func TestBackgroundWaitsForJobs(t *testing.T) {
	var bg Background
	release, ran := make(chan struct{}), make(chan struct{})
	bg.Go(func(ctx context.Context) {
		if _, ok := ctx.Deadline(); !ok {
			t.Error("background job has no deadline")
		}
		<-release
		close(ran)
	})
	bg.Go(func(context.Context) { panic("job failed") })

	short, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if bg.Wait(short) {
		t.Fatal("Wait returned before the job finished")
	}
	close(release)
	if !bg.Wait(context.Background()) {
		t.Fatal("Wait did not see the jobs finish")
	}
	<-ran
}

// The token `cmd/mcp token` prints is for the MCP only: the HTTP API's check
// refuses it, so leaking it gives read access to the history and nothing else.
func TestMCPTokenIsRefusedByTheAPI(t *testing.T) {
	ctx := context.Background()
	svc, mem, _, _ := hardeningService(t)
	const email, password, ip = "agent@example.com", "right-password", "127.0.0.1"
	addAccount(t, mem, email, password)
	acc, err := mem.GetAccountByEmail(ctx, email)
	if err != nil {
		t.Fatal(err)
	}

	mcp, err := svc.IssueMCPToken(ctx, LoginInput{Email: email, Password: password}, ip)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ResolveToken(ctx, mcp.Token); codeOf(err) != domain.CodeUnauthorized {
		t.Fatalf("API check of an MCP token = %v, want unauthorized", err)
	}
	if id, err := svc.ResolveMCPToken(ctx, mcp.Token); err != nil || id != acc.ID {
		t.Fatalf("MCP check of an MCP token = %v, %v", id, err)
	}

	// A session token still works for both, so a token issued before scopes
	// existed keeps the MCP running.
	session, err := svc.Login(ctx, LoginInput{Email: email, Password: password}, ip)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ResolveToken(ctx, session.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ResolveMCPToken(ctx, session.Token); err != nil {
		t.Fatal(err)
	}

	// The wrong password issues nothing, and a password change revokes the MCP token.
	if _, err := svc.IssueMCPToken(ctx, LoginInput{Email: email, Password: "wrong-password"}, ip); codeOf(err) != domain.CodeInvalidCredentials {
		t.Fatalf("wrong password = %v", err)
	}
	err = svc.ForUser(acc.ID).ChangePassword(ctx, ChangePasswordInput{CurrentPassword: password, NewPassword: "another-password"}, session.Token, ip)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ResolveMCPToken(ctx, mcp.Token); codeOf(err) != domain.CodeUnauthorized {
		t.Fatalf("MCP token after a password change = %v, want unauthorized", err)
	}
}

// "This address has an account" costs a slot of the per-client send cap, so one
// client cannot ask about addresses in bulk; past the cap it learns nothing.
func TestRegisterCodeEmailInUseIsCappedPerClient(t *testing.T) {
	ctx := context.Background()
	svc, mem, rec, _ := hardeningService(t)
	const ip = "203.0.113.100"
	for i := 0; i < svc.Limits.SendIPLimit+1; i++ {
		addAccount(t, mem, fmt.Sprintf("taken%d@example.com", i), "some-password")
	}
	for i := 0; i < svc.Limits.SendIPLimit; i++ {
		err := svc.RequestRegisterCode(ctx, fmt.Sprintf("taken%d@example.com", i), ip)
		if codeOf(err) != domain.CodeEmailInUse {
			t.Fatalf("taken %d = %v, want email_in_use", i, err)
		}
	}
	retryAfter(t, svc.RequestRegisterCode(ctx, fmt.Sprintf("taken%d@example.com", svc.Limits.SendIPLimit), ip))
	retryAfter(t, svc.RequestRegisterCode(ctx, "free@example.com", ip))
	if len(rec.Messages) != 0 {
		t.Fatalf("mail sent: %+v", rec.Messages)
	}
	// Another client is not affected.
	if err := svc.RequestRegisterCode(ctx, "free@example.com", "203.0.113.101"); err != nil {
		t.Fatal(err)
	}
}
