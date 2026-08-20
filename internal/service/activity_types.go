package service

import (
	"context"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/google/uuid"
)

type CreateActivityTypeInput struct {
	Name  string
	Color string
}

type UpdateActivityTypeInput struct {
	Name  *string
	Color *string
}

func (s *Service) ListActivityTypes(ctx context.Context) ([]domain.ActivityType, error) {
	return s.Store.ListActivityTypes(ctx, s.User)
}

func (s *Service) GetActivityType(ctx context.Context, id uuid.UUID) (domain.ActivityType, error) {
	return s.Store.GetActivityType(ctx, s.User, id)
}

func (s *Service) CreateActivityType(ctx context.Context, in CreateActivityTypeInput) (domain.ActivityType, error) {
	name, err := domain.NormalizeActivityTypeName(in.Name)
	if err != nil {
		return domain.ActivityType{}, err
	}
	color, err := domain.NormalizeActivityColor(in.Color)
	if err != nil {
		return domain.ActivityType{}, err
	}
	used, err := s.Store.ActivityTypeNameInUse(ctx, s.User, name, uuid.Nil)
	if err != nil {
		return domain.ActivityType{}, err
	}
	if used {
		return domain.ActivityType{}, domain.ErrNameInUse()
	}
	a := domain.ActivityType{
		ID:    s.NewID().String(),
		Name:  name,
		Color: color,
	}
	return s.Store.CreateActivityType(ctx, s.User, a)
}

func (s *Service) UpdateActivityType(ctx context.Context, id uuid.UUID, in UpdateActivityTypeInput) (domain.ActivityType, error) {
	a, err := s.Store.GetActivityType(ctx, s.User, id)
	if err != nil {
		return domain.ActivityType{}, err
	}
	if in.Name != nil {
		name, err := domain.NormalizeActivityTypeName(*in.Name)
		if err != nil {
			return domain.ActivityType{}, err
		}
		a.Name = name
	}
	if in.Color != nil {
		color, err := domain.NormalizeActivityColor(*in.Color)
		if err != nil {
			return domain.ActivityType{}, err
		}
		a.Color = color
	}
	used, err := s.Store.ActivityTypeNameInUse(ctx, s.User, a.Name, id)
	if err != nil {
		return domain.ActivityType{}, err
	}
	if used {
		return domain.ActivityType{}, domain.ErrNameInUse()
	}
	return s.Store.UpdateActivityType(ctx, s.User, a)
}

func (s *Service) DeleteActivityType(ctx context.Context, id uuid.UUID) error {
	if _, err := s.Store.GetActivityType(ctx, s.User, id); err != nil {
		return err
	}
	n, err := s.Store.CountActivityTypeSessions(ctx, s.User, id)
	if err != nil {
		return err
	}
	if err := domain.CanDeleteActivityType(n); err != nil {
		return err
	}
	return s.Store.DeleteActivityType(ctx, s.User, id)
}

func (s *Service) ActivityTypeSessionCount(ctx context.Context, id uuid.UUID) (int, error) {
	if _, err := s.Store.GetActivityType(ctx, s.User, id); err != nil {
		return 0, err
	}
	return s.Store.CountActivityTypeSessions(ctx, s.User, id)
}
