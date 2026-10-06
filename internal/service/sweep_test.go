package service

import (
	"context"
	"testing"
	"time"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/EmilM32/vynno-api/internal/store"
	"github.com/google/uuid"
)

func TestSweepExpiredKeepsLiveRowsAndSendCaps(t *testing.T) {
	ctx := context.Background()
	mem := store.NewEmptyMemory()
	svc := serviceWithClock(t, mem)
	now := svc.Now()
	user := uuid.New()

	for _, tok := range []store.Token{
		{ID: uuid.New(), UserID: user, TokenHash: "expired", ExpiresAt: now.Add(-time.Second)},
		{ID: uuid.New(), UserID: user, TokenHash: "at-now", ExpiresAt: now},
		{ID: uuid.New(), UserID: user, TokenHash: "live", ExpiresAt: now.Add(time.Hour)},
	} {
		if err := mem.CreateToken(ctx, tok); err != nil {
			t.Fatal(err)
		}
	}
	for email, expires := range map[string]time.Time{
		// Expired, but its send window may still be running: keep.
		"recent@example.com": now.Add(-30 * time.Minute),
		// Expired more than a send window ago: remove.
		"stale@example.com": now.Add(-domain.OTPSendWindow - time.Minute),
		"live@example.com":  now.Add(10 * time.Minute),
	} {
		if err := mem.UpsertEmailChallenge(ctx, store.EmailChallenge{
			Email: email, Purpose: domain.PurposeRegister, CodeHash: "h",
			ExpiresAt: expires, SentAt: expires.Add(-domain.OTPTTL), SendCount: 1,
			SendWindowStart: expires.Add(-domain.OTPTTL),
		}); err != nil {
			t.Fatal(err)
		}
	}

	res, err := svc.SweepExpired(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res.Tokens != 2 || res.Challenges != 1 {
		t.Fatalf("swept %+v, want 2 tokens and 1 challenge", res)
	}
	if _, err := mem.GetTokenByHash(ctx, "live"); err != nil {
		t.Fatalf("live token removed: %v", err)
	}
	for _, email := range []string{"recent@example.com", "live@example.com"} {
		if _, err := mem.GetEmailChallenge(ctx, email, domain.PurposeRegister); err != nil {
			t.Fatalf("%s removed: %v", email, err)
		}
	}
	if _, err := mem.GetEmailChallenge(ctx, "stale@example.com", domain.PurposeRegister); err == nil {
		t.Fatal("stale challenge kept")
	}
}
