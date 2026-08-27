package domain

import (
	"testing"
	"time"
)

func TestGenerateOTP(t *testing.T) {
	t.Parallel()
	seen := map[string]struct{}{}
	for i := 0; i < 20; i++ {
		got, err := GenerateOTP()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := NormalizeOTP(got); err != nil {
			t.Fatalf("generated %q: %v", got, err)
		}
		seen[got] = struct{}{}
	}
	if len(seen) < 2 {
		t.Fatal("expected some variation across generated codes")
	}
}

func TestNormalizeOTP(t *testing.T) {
	t.Parallel()
	got, err := NormalizeOTP("  012345  ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "012345" {
		t.Fatalf("got %q", got)
	}
	if _, err := NormalizeOTP("12345"); err == nil {
		t.Fatal("expected too short")
	}
	if _, err := NormalizeOTP("12345a"); err == nil {
		t.Fatal("expected non-digit")
	}
	if _, err := NormalizeOTP(""); err == nil {
		t.Fatal("expected empty")
	}
}

func TestOTPWindows(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	if !OTPExpired(now, now) {
		t.Fatal("equal expiry should be expired")
	}
	if OTPExpired(now.Add(time.Second), now) {
		t.Fatal("future expiry should be valid")
	}
	if !OTPSendCooldownActive(now, now.Add(59*time.Second)) {
		t.Fatal("59s should still be cooling down")
	}
	if OTPSendCooldownActive(now, now.Add(60*time.Second)) {
		t.Fatal("60s should allow a resend")
	}

	start, count := AdvanceSendWindow(now, 4, now.Add(59*time.Minute))
	if !start.Equal(now) || count != 4 {
		t.Fatalf("within window: %v %d", start, count)
	}
	start, count = AdvanceSendWindow(now, 4, now.Add(OTPSendWindow))
	if !start.Equal(now.Add(OTPSendWindow)) || count != 0 {
		t.Fatalf("rolled window: %v %d", start, count)
	}
	if !OTPSendLimited(OTPSendsPerHour) || OTPSendLimited(OTPSendsPerHour-1) {
		t.Fatal("send cap")
	}
	if !OTPGuessesSpent(OTPMaxAttempts) || OTPGuessesSpent(OTPMaxAttempts-1) {
		t.Fatal("guess cap")
	}
}
