package ratelimit

import (
	"sync"
	"sync/atomic"
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

func TestReserveIsAtomicUnderConcurrency(t *testing.T) {
	l := New()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	const limit = 10
	var granted atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if ok, _ := l.Reserve("email", now, limit, time.Minute); ok {
				granted.Add(1)
			}
		}()
	}
	wg.Wait()
	if got := granted.Load(); got != limit {
		t.Fatalf("granted = %d, want %d", got, limit)
	}
}

func TestReleaseUndoesReserve(t *testing.T) {
	l := New()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if ok, _ := l.Reserve("ip", now, 1, time.Minute); !ok {
		t.Fatal("first reserve blocked")
	}
	if ok, _ := l.Reserve("ip", now, 1, time.Minute); ok {
		t.Fatal("second reserve should be over the limit")
	}
	l.Release("ip", now)
	if ok, _ := l.Reserve("ip", now, 1, time.Minute); !ok {
		t.Fatal("reserve after release blocked")
	}
	l.Release("ip", now.Add(time.Second)) // no matching event: no-op
	if ok, _ := l.Reserve("ip", now, 1, time.Minute); ok {
		t.Fatal("unmatched release must not free a slot")
	}
}
