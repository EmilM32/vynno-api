package ratelimit

import (
	"testing"
	"time"
)

func TestWindowAndReset(t *testing.T) {
	l := New()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	const window = 15 * time.Minute
	const limit = 10

	for i := 0; i < limit; i++ {
		ok, retry := l.Allow("email", now, limit, window)
		if !ok || retry != 0 {
			t.Fatalf("event %d blocked retry=%s", i, retry)
		}
		l.Hit("email", now, window)
	}
	ok, retry := l.Allow("email", now, limit, window)
	if ok {
		t.Fatal("expected the next event to be limited")
	}
	if retry != window {
		t.Fatalf("retry = %s, want %s", retry, window)
	}

	later := now.Add(window)
	ok, retry = l.Allow("email", later, limit, window)
	if !ok || retry != 0 {
		t.Fatalf("window passed: ok=%v retry=%s", ok, retry)
	}

	l.Hit("email", later, window)
	l.Reset("email")
	ok, _ = l.Allow("email", later, limit, window)
	if !ok {
		t.Fatal("reset should clear the counter")
	}

	for i := 0; i < 30; i++ {
		l.Hit("ip", now, window)
	}
	if ok, _ := l.Allow("ip", now, 30, window); ok {
		t.Fatal("expected ip cap")
	}
	l.Reset("email")
	if ok, _ := l.Allow("ip", now, 30, window); ok {
		t.Fatal("resetting another key must not clear ip")
	}
	if ok, _ := l.Allow("ip", now.Add(window), 30, window); !ok {
		t.Fatal("ip window should pass")
	}
}
