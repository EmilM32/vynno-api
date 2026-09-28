package service

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/EmilM32/vynno-api/internal/mail"
	"github.com/EmilM32/vynno-api/internal/store"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func TestDummyPasswordHashUsesDefaultCost(t *testing.T) {
	cost, err := bcrypt.Cost(dummyPasswordHash())
	if err != nil {
		t.Fatal(err)
	}
	if cost != bcrypt.DefaultCost {
		t.Fatalf("cost = %d, want %d", cost, bcrypt.DefaultCost)
	}
}

func TestLoginRateLimitAndReset(t *testing.T) {
	ctx := context.Background()
	mem := store.NewEmptyMemory()
	svc := serviceWithClock(t, mem)
	const email = "ada@example.com"
	const password = "right-password"
	const ip = "203.0.113.10"
	accountHash := addAccount(t, mem, email, password)

	var calls int
	orig := comparePassword
	comparePassword = func(hash, pw []byte) error {
		if bytes.Equal(hash, accountHash) {
			calls++
		}
		return orig(hash, pw)
	}
	t.Cleanup(func() { comparePassword = orig })

	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	svc.Now = func() time.Time { return now }
	for i := 0; i < 10; i++ {
		_, err := svc.Login(ctx, LoginInput{Email: email, Password: "wrong-password"}, ip)
		if codeOf(err) != domain.CodeInvalidCredentials {
			t.Fatalf("failure %d: %v", i, err)
		}
	}
	if calls != 10 {
		t.Fatalf("compares after 10 failures = %d", calls)
	}
	_, err := svc.Login(ctx, LoginInput{Email: email, Password: password}, ip)
	if codeOf(err) != domain.CodeRateLimited {
		t.Fatalf("11th = %v", err)
	}
	de, _ := domain.AsError(err)
	if de.RetryAfter <= 0 {
		t.Fatalf("retry = %s", de.RetryAfter)
	}
	if calls != 10 {
		t.Fatalf("11th compared the password: calls=%d", calls)
	}

	now = now.Add(15 * time.Minute)
	if _, err := svc.Login(ctx, LoginInput{Email: email, Password: password}, ip); err != nil {
		t.Fatalf("after window: %v", err)
	}

	for i := 0; i < 9; i++ {
		if _, err := svc.Login(ctx, LoginInput{Email: email, Password: "wrong-password"}, ip); codeOf(err) != domain.CodeInvalidCredentials {
			t.Fatalf("post-success failure %d: %v", i, err)
		}
	}
	if _, err := svc.Login(ctx, LoginInput{Email: email, Password: password}, ip); err != nil {
		t.Fatalf("success should reset the email counter: %v", err)
	}
	for i := 0; i < 10; i++ {
		if _, err := svc.Login(ctx, LoginInput{Email: email, Password: "wrong-password"}, ip); codeOf(err) != domain.CodeInvalidCredentials {
			t.Fatalf("after reset failure %d: %v", i, err)
		}
	}
	if codeOf(mustLoginErr(svc, ctx, email, password, ip)) != domain.CodeRateLimited {
		t.Fatal("expected rate limit after 10 new failures")
	}
}

func TestLoginUnknownAndHashlessCompareDummy(t *testing.T) {
	ctx := context.Background()
	mem := store.NewEmptyMemory()
	svc := serviceWithClock(t, mem)
	id := uuid.New()
	if err := mem.CreateAccount(ctx, store.Account{ID: id, Email: "empty@example.com", PasswordHash: ""}); err != nil {
		t.Fatal(err)
	}

	orig := comparePassword
	dummy := dummyPasswordHash()
	var dummyCalls int
	comparePassword = func(hash, pw []byte) error {
		if bytes.Equal(hash, dummy) {
			cost, costErr := bcrypt.Cost(hash)
			if costErr != nil || cost != bcrypt.DefaultCost {
				t.Errorf("dummy cost = %d err=%v", cost, costErr)
			}
			dummyCalls++
		}
		return orig(hash, pw)
	}
	t.Cleanup(func() { comparePassword = orig })

	_, err := svc.Login(ctx, LoginInput{Email: "missing@example.com", Password: "whatever-1"}, "198.51.100.4")
	if codeOf(err) != domain.CodeInvalidCredentials {
		t.Fatal(err)
	}
	_, err = svc.Login(ctx, LoginInput{Email: "empty@example.com", Password: "whatever-1"}, "198.51.100.4")
	if codeOf(err) != domain.CodeInvalidCredentials {
		t.Fatal(err)
	}
	if dummyCalls != 2 {
		t.Fatalf("dummy compares = %d", dummyCalls)
	}
}

func TestLoginIPCapIsNotResetBySuccess(t *testing.T) {
	ctx := context.Background()
	mem := store.NewEmptyMemory()
	svc := serviceWithClock(t, mem)
	const ip = "203.0.113.50"
	addAccount(t, mem, "kept@example.com", "right-password")
	for i := 0; i < 30; i++ {
		email := fmt.Sprintf("u%02d@example.com", i)
		addAccount(t, mem, email, "other-password")
		_, err := svc.Login(ctx, LoginInput{Email: email, Password: "wrong-password"}, ip)
		if codeOf(err) != domain.CodeInvalidCredentials {
			t.Fatalf("ip failure %d: %v", i, err)
		}
	}
	_, err := svc.Login(ctx, LoginInput{Email: "kept@example.com", Password: "right-password"}, ip)
	if codeOf(err) != domain.CodeRateLimited {
		t.Fatalf("ip cap = %v", err)
	}
	if _, err := svc.Login(ctx, LoginInput{Email: "kept@example.com", Password: "right-password"}, "203.0.113.51"); err != nil {
		t.Fatalf("other ip: %v", err)
	}
}

func TestRegisterCodeSendIPCapHonorsLimits(t *testing.T) {
	ctx := context.Background()
	const ip = "203.0.113.60"
	send := func(svc *Service, n int) error {
		var err error
		for i := 0; i < n; i++ {
			err = svc.RequestRegisterCode(ctx, fmt.Sprintf("new%02d@example.com", i), ip)
			if err != nil {
				return err
			}
		}
		return nil
	}

	strict := serviceWithClock(t, store.NewEmptyMemory())
	if err := send(strict, 5); err != nil {
		t.Fatalf("default first 5: %v", err)
	}
	if err := strict.RequestRegisterCode(ctx, "sixth@example.com", ip); codeOf(err) != domain.CodeRateLimited {
		t.Fatalf("default 6th = %v", err)
	}

	custom := serviceWithClock(t, store.NewEmptyMemory())
	custom.Limits.SendIPLimit = 7
	if err := send(custom, 7); err != nil {
		t.Fatalf("custom first 7: %v", err)
	}
	if err := custom.RequestRegisterCode(ctx, "eighth@example.com", ip); codeOf(err) != domain.CodeRateLimited {
		t.Fatalf("custom 8th = %v", err)
	}

	relaxed := serviceWithClock(t, store.NewEmptyMemory())
	relaxed.Limits = RelaxedAuthLimits()
	if err := send(relaxed, 50); err != nil {
		t.Fatalf("relaxed 50: %v", err)
	}
}

func serviceWithClock(t *testing.T, mem *store.Memory) *Service {
	t.Helper()
	svc := New(mem, mail.Discard())
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	svc.Now = func() time.Time { return now }
	return svc
}

func addAccount(t *testing.T, mem *store.Memory, email, password string) []byte {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.New()
	ctx := context.Background()
	if err := mem.CreateAccount(ctx, store.Account{ID: id, Email: email, PasswordHash: string(hash)}); err != nil {
		t.Fatal(err)
	}
	if err := mem.CreateProfile(ctx, id, domain.Profile{}); err != nil {
		t.Fatal(err)
	}
	return hash
}

func codeOf(err error) string {
	de, ok := domain.AsError(err)
	if !ok {
		return ""
	}
	return de.Code
}

func mustLoginErr(svc *Service, ctx context.Context, email, password, ip string) error {
	_, err := svc.Login(ctx, LoginInput{Email: email, Password: password}, ip)
	return err
}
