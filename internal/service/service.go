package service

import (
	"context"
	"time"

	"github.com/EmilM32/vynno-api/internal/mail"
	"github.com/EmilM32/vynno-api/internal/ratelimit"
	"github.com/EmilM32/vynno-api/internal/store"
	"github.com/google/uuid"
)

type Service struct {
	Store   store.Store
	Mailer  mail.Mailer
	User    uuid.UUID
	Now     func() time.Time
	NewID   func() uuid.UUID
	Limiter *ratelimit.Limiter
	Limits  AuthLimits
	// Async runs work that must not hold up or influence the response (the password
	// reset mail). nil runs it inline, which is what tests want. cmd/api sets
	// Background.Go.
	Async func(fn func(ctx context.Context))
}

func New(st store.Store, m mail.Mailer) *Service {
	if m == nil {
		m = mail.Discard()
	}
	return &Service{
		Store:   st,
		Mailer:  m,
		Now:     func() time.Time { return time.Now().UTC() },
		NewID:   uuid.New,
		Limiter: ratelimit.New(),
		Limits:  DefaultAuthLimits(),
	}
}

// runAsync hands fn to Async, or runs it inline when Async is nil. fn gets a
// context that outlives the request.
func (s *Service) runAsync(ctx context.Context, fn func(ctx context.Context)) {
	if s.Async == nil {
		fn(context.WithoutCancel(ctx))
		return
	}
	s.Async(fn)
}

// inTx runs fn with a copy of the service whose Store is one transaction: the
// writes commit together, or none does when fn returns an error.
func (s *Service) inTx(ctx context.Context, fn func(tx *Service) error) error {
	return s.Store.InTx(ctx, func(st store.Store) error {
		cp := *s
		cp.Store = st
		return fn(&cp)
	})
}

// inAccountTx is inTx with the signed-in account locked first. Requests that read
// a row, decide, and write it back use this so two of them for the same account
// run one after the other and neither decides on a stale read.
func (s *Service) inAccountTx(ctx context.Context, fn func(tx *Service) error) error {
	return s.inTx(ctx, func(tx *Service) error {
		if err := tx.Store.LockAccount(ctx, tx.User); err != nil {
			return err
		}
		return fn(tx)
	})
}

// lockedResult is inAccountTx for a function that also returns a value.
func lockedResult[T any](ctx context.Context, s *Service, fn func(tx *Service) (T, error)) (T, error) {
	var out T
	err := s.inAccountTx(ctx, func(tx *Service) error {
		v, err := fn(tx)
		if err != nil {
			return err
		}
		out = v
		return nil
	})
	return out, err
}

func (s *Service) ForUser(user uuid.UUID) *Service {
	cp := *s
	cp.User = user
	return &cp
}
