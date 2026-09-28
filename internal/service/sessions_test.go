package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/EmilM32/vynno-api/internal/mail"
	"github.com/EmilM32/vynno-api/internal/store"
	"github.com/google/uuid"
)

func TestConcurrentStopHasOneWinner(t *testing.T) {
	ctx := context.Background()
	user := uuid.New()
	project := domain.Project{ID: uuid.NewString(), Name: "Identity", Color: "#3b82f6"}
	mem := store.NewMemory(user, domain.Profile{}, project)
	svc := New(mem, mail.Discard()).ForUser(user)
	var clock sync.Mutex
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	svc.Now = func() time.Time {
		clock.Lock()
		defer clock.Unlock()
		now = now.Add(time.Millisecond)
		return now
	}

	for trial := 0; trial < 20; trial++ {
		sess, err := svc.StartSession(ctx, StartSessionInput{ProjectID: project.ID, Note: "race"})
		if err != nil {
			t.Fatalf("trial %d start: %v", trial, err)
		}
		id := uuid.MustParse(sess.ID)

		const n = 10
		var wg sync.WaitGroup
		results := make([]domain.Session, n)
		errs := make([]error, n)
		start := make(chan struct{})
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				results[i], errs[i] = svc.StopSession(ctx, id)
			}(i)
		}
		close(start)
		wg.Wait()

		var winner *domain.Session
		for i, err := range errs {
			switch {
			case err == nil:
				if winner != nil {
					t.Fatalf("trial %d: two stops succeeded (%v, %v)", trial, winner.EndedAt, results[i].EndedAt)
				}
				winner = &results[i]
			case codeOf(err) != domain.CodeInvalidTransition:
				t.Fatalf("trial %d: loser error = %v", trial, err)
			}
		}
		if winner == nil {
			t.Fatalf("trial %d: no stop succeeded", trial)
		}
		stored, err := svc.GetSession(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if !stored.EndedAt.Equal(*winner.EndedAt) {
			t.Fatalf("trial %d: stored endedAt %v, returned %v", trial, stored.EndedAt, winner.EndedAt)
		}
	}
}

func TestTransitionSessionRejectsStaleStatus(t *testing.T) {
	ctx := context.Background()
	user := uuid.New()
	project := domain.Project{ID: uuid.NewString(), Name: "Identity", Color: "#3b82f6"}
	mem := store.NewMemory(user, domain.Profile{}, project)
	svc := New(mem, mail.Discard()).ForUser(user)

	sess, err := svc.StartSession(ctx, StartSessionInput{ProjectID: project.ID, Note: "stale"})
	if err != nil {
		t.Fatal(err)
	}
	stale := sess
	if _, err := svc.StopSession(ctx, uuid.MustParse(sess.ID)); err != nil {
		t.Fatal(err)
	}
	next, err := domain.Stop(stale, svc.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mem.TransitionSession(ctx, user, next, domain.StatusActive); codeOf(err) != domain.CodeInvalidTransition {
		t.Fatalf("stale transition = %v", err)
	}
	next.ID = uuid.NewString()
	if _, err := mem.TransitionSession(ctx, user, next, domain.StatusActive); codeOf(err) != domain.CodeNotFound {
		t.Fatalf("missing session = %v", err)
	}
}
