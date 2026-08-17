package service

import (
	"time"

	"github.com/EmilM32/vynno-api/internal/store"
	"github.com/google/uuid"
)

type Service struct {
	Store store.Store
	User  uuid.UUID
	Now   func() time.Time
	NewID func() uuid.UUID
}

func New(st store.Store) *Service {
	return &Service{
		Store: st,
		Now:   func() time.Time { return time.Now().UTC() },
		NewID: uuid.New,
	}
}

func (s *Service) ForUser(user uuid.UUID) *Service {
	cp := *s
	cp.User = user
	return &cp
}
