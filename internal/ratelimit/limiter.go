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
