package service

import (
	"context"

	"github.com/EmilM32/vynno-api/internal/domain"
)

func (s *Service) GetProfile(ctx context.Context) (domain.Profile, error) {
	return s.Store.GetProfile(ctx, s.User)
}
