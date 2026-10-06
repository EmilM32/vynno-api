package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/EmilM32/vynno-api/internal/store"
	"github.com/google/uuid"
)

// barrierStore holds every ReserveChallengeGuess caller until all n have arrived,
// so the guesses really run side by side.
type barrierStore struct {
	*store.Memory
	wg *sync.WaitGroup
}

func (b barrierStore) ReserveChallengeGuess(ctx context.Context, email, purpose string, userID uuid.UUID, maxAttempts int) (store.EmailChallenge, error) {
	b.wg.Done()
	b.wg.Wait()
	return b.Memory.ReserveChallengeGuess(ctx, email, purpose, userID, maxAttempts)
}

func TestConcurrentCodeGuessesStayWithinCap(t *testing.T) {
	ctx := context.Background()
	mem := store.NewEmptyMemory()
	const email = "victim@example.com"
	addAccount(t, mem, email, "old-password")
	svc := serviceWithClock(t, mem)
	code, err := svc.issueOTPChallenge(ctx, email, domain.PurposePasswordReset, uuid.Nil)
	if err != nil {
		t.Fatal(err)
	}

	const n = 200
	var arrive sync.WaitGroup
	arrive.Add(n)
	svc.Store = barrierStore{Memory: mem, wg: &arrive}

	var resets atomic.Int32
	var done sync.WaitGroup
	for i := 0; i < n; i++ {
		guess := fmt.Sprintf("%06d", i)
		if guess == code {
			guess = "999999"
		}
		if i == n-1 {
			guess = code // the right code arrives with 199 wrong ones
		}
		done.Add(1)
		go func() {
			defer done.Done()
			err := svc.ResetPassword(ctx, ResetPasswordInput{Email: email, Code: guess, Password: "attacker-pass"})
			if err == nil {
				resets.Add(1)
			}
		}()
	}
	done.Wait()

	ch, err := mem.GetEmailChallenge(ctx, email, domain.PurposePasswordReset)
	if resets.Load() == 1 {
		// The right code won one of the five slots; the challenge is gone.
		if err == nil {
			t.Fatal("challenge left after a successful reset")
		}
		return
	}
	if resets.Load() != 0 {
		t.Fatalf("resets = %d", resets.Load())
	}
	if err != nil {
		t.Fatalf("spent challenge should stay for the send caps: %v", err)
	}
	if ch.AttemptCount != domain.OTPMaxAttempts {
		t.Fatalf("attempts = %d, want %d", ch.AttemptCount, domain.OTPMaxAttempts)
	}
}

func TestSpentCodeIsRejectedAndKeepsSendCooldown(t *testing.T) {
	ctx := context.Background()
	mem := store.NewEmptyMemory()
	const email = "ada@example.com"
	addAccount(t, mem, email, "old-password")
	svc := serviceWithClock(t, mem)
	code, err := svc.issueOTPChallenge(ctx, email, domain.PurposePasswordReset, uuid.Nil)
	if err != nil {
		t.Fatal(err)
	}
	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	for i := 1; i <= domain.OTPMaxAttempts; i++ {
		err := svc.ResetPassword(ctx, ResetPasswordInput{Email: email, Code: wrong, Password: "new-password"})
		want := domain.CodeInvalidCode
		if i == domain.OTPMaxAttempts {
			want = domain.CodeRateLimited
		}
		if codeOf(err) != want {
			t.Fatalf("guess %d = %v, want %s", i, err, want)
		}
	}
	err = svc.ResetPassword(ctx, ResetPasswordInput{Email: email, Code: code, Password: "new-password"})
	if codeOf(err) != domain.CodeInvalidCode {
		t.Fatalf("right code after the cap = %v", err)
	}
	// Spending the guesses must not clear the resend cooldown.
	err = svc.RequestPasswordReset(ctx, email, "203.0.113.10")
	if codeOf(err) != domain.CodeRateLimited {
		t.Fatalf("resend inside cooldown = %v", err)
	}
}

func TestCodeIsSingleUse(t *testing.T) {
	ctx := context.Background()
	mem := store.NewEmptyMemory()
	const email = "ada@example.com"
	addAccount(t, mem, email, "old-password")
	svc := serviceWithClock(t, mem)
	code, err := svc.issueOTPChallenge(ctx, email, domain.PurposePasswordReset, uuid.Nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ResetPassword(ctx, ResetPasswordInput{Email: email, Code: code, Password: "new-password"}); err != nil {
		t.Fatal(err)
	}
	err = svc.ResetPassword(ctx, ResetPasswordInput{Email: email, Code: code, Password: "other-password"})
	if codeOf(err) != domain.CodeInvalidCode {
		t.Fatalf("second use = %v", err)
	}
}

func TestConcurrentLoginFailuresStayWithinEmailCap(t *testing.T) {
	ctx := context.Background()
	mem := store.NewEmptyMemory()
	const email = "ada@example.com"
	accountHash := addAccount(t, mem, email, "right-password")
	svc := serviceWithClock(t, mem)

	var compares atomic.Int32
	orig := comparePassword
	comparePassword = func(hash, pw []byte) error {
		if string(hash) == string(accountHash) {
			compares.Add(1)
			time.Sleep(5 * time.Millisecond) // widen the window bcrypt would leave
		}
		return orig(hash, pw)
	}
	t.Cleanup(func() { comparePassword = orig })

	var done sync.WaitGroup
	for i := 0; i < 100; i++ {
		done.Add(1)
		go func(i int) {
			defer done.Done()
			ip := fmt.Sprintf("203.0.113.%d", i) // spread IPs so only the email cap binds
			_, _ = svc.Login(ctx, LoginInput{Email: email, Password: "wrong-password"}, ip)
		}(i)
	}
	done.Wait()
	if got := int(compares.Load()); got != svc.Limits.LoginEmailFails {
		t.Fatalf("password compares = %d, want %d", got, svc.Limits.LoginEmailFails)
	}
}
