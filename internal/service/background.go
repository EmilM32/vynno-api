package service

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// backgroundTimeout bounds one background job: an SMTP dial plus send.
const backgroundTimeout = 45 * time.Second

// Background runs jobs off the request path and lets shutdown wait for them.
// The zero value is ready to use.
type Background struct {
	wg sync.WaitGroup
}

// Go starts fn in its own goroutine with a context limited to backgroundTimeout.
// A panic in fn is logged and does not take the process down.
func (b *Background) Go(fn func(ctx context.Context)) {
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				slog.Error("background job panicked", "panic", r)
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), backgroundTimeout)
		defer cancel()
		fn(ctx)
	}()
}

// Wait returns when every started job has finished or ctx is done, and reports
// whether all of them finished.
func (b *Background) Wait(ctx context.Context) bool {
	done := make(chan struct{})
	go func() {
		b.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-ctx.Done():
		return false
	}
}
