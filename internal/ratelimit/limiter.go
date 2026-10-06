package ratelimit

import (
	"sync"
	"time"
)

// Limiter counts events in memory for this process only.
// A multi-instance deployment would need a shared store; these counters are not shared across processes.
type Limiter struct {
	mu     sync.Mutex
	events map[string][]time.Time
}

func New() *Limiter {
	return &Limiter{events: map[string][]time.Time{}}
}

// Allow reports whether key is still under limit events inside window ending at now.
// It does not record an event. When the key is over the limit, retryAfter is how long
// until the oldest event in the window expires (at least one second).
func (l *Limiter) Allow(key string, now time.Time, limit int, window time.Duration) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now = now.UTC()
	kept := prune(l.events[key], now, window)
	l.events[key] = kept
	if len(kept) >= limit {
		retry := kept[0].Add(window).Sub(now)
		if retry < time.Second {
			retry = time.Second
		}
		return false, retry
	}
	return true, 0
}

// Reserve is Allow and Hit under one lock: when key is under limit it records an
// event at now and reports true. Callers that count attempts before doing slow work
// (bcrypt) use it so concurrent attempts cannot all pass the check before any is
// recorded. Undo an attempt that should not count with Release.
func (l *Limiter) Reserve(key string, now time.Time, limit int, window time.Duration) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now = now.UTC()
	kept := prune(l.events[key], now, window)
	if len(kept) >= limit {
		l.events[key] = kept
		retry := kept[0].Add(window).Sub(now)
		if retry < time.Second {
			retry = time.Second
		}
		return false, retry
	}
	l.events[key] = append(kept, now)
	return true, 0
}

// Release removes one event recorded at exactly at, the newest such event first.
// It undoes a Reserve made with the same time. Nothing happens when none matches.
func (l *Limiter) Release(key string, at time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	at = at.UTC()
	events := l.events[key]
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Equal(at) {
			l.events[key] = append(events[:i], events[i+1:]...)
			return
		}
	}
}

// Hit records an event at now. The caller passes now; there is no hidden clock.
func (l *Limiter) Hit(key string, now time.Time, window time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now = now.UTC()
	kept := prune(l.events[key], now, window)
	l.events[key] = append(kept, now)
}

// Reset clears every event for key.
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.events, key)
}

func prune(events []time.Time, now time.Time, window time.Duration) []time.Time {
	cutoff := now.Add(-window)
	i := 0
	for _, e := range events {
		if e.After(cutoff) {
			events[i] = e
			i++
		}
	}
	return events[:i]
}
