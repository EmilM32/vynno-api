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

func New(st store.Store, user uuid.UUID) *Service {
	return &Service{
		Store: st,
		User:  user,
		Now:   func() time.Time { return time.Now().UTC() },
		NewID: uuid.New,
	}
}
