package service

import (
	"time"

	"github.com/EmilM32/vynno-api/internal/mail"
	"github.com/EmilM32/vynno-api/internal/store"
	"github.com/google/uuid"
)

type Service struct {
	Store  store.Store
	Mailer mail.Mailer
	User   uuid.UUID
	Now    func() time.Time
	NewID  func() uuid.UUID
}

func New(st store.Store, m mail.Mailer) *Service {
	if m == nil {
		m = mail.Discard()
	}
	return &Service{
		Store:  st,
		Mailer: m,
		Now:    func() time.Time { return time.Now().UTC() },
		NewID:  uuid.New,
	}
}

func (s *Service) ForUser(user uuid.UUID) *Service {
	cp := *s
	cp.User = user
	return &cp
}
