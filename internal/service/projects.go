package service

import (
	"context"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/google/uuid"
)

type CreateProjectInput struct {
	Name  string
	Color string
	Code  *string
}

type UpdateProjectInput struct {
	Name    *string
	Color   *string
	Code    *string
	CodeSet bool
}

func (s *Service) ListProjects(ctx context.Context, includeArchived bool) ([]domain.Project, error) {
	return s.Store.ListProjects(ctx, s.User, includeArchived)
}

func (s *Service) GetProject(ctx context.Context, id uuid.UUID) (domain.Project, error) {
	return s.Store.GetProject(ctx, s.User, id)
}

func (s *Service) CreateProject(ctx context.Context, in CreateProjectInput) (domain.Project, error) {
	name, err := domain.NormalizeName(in.Name)
	if err != nil {
		return domain.Project{}, err
	}
	color, err := domain.NormalizeColor(in.Color)
	if err != nil {
		return domain.Project{}, err
	}
	code, err := domain.NormalizeCode(in.Code)
	if err != nil {
		return domain.Project{}, err
	}
	if code != nil {
		used, err := s.Store.CodeInUse(ctx, s.User, *code, uuid.Nil)
		if err != nil {
			return domain.Project{}, err
		}
		if used {
			return domain.Project{}, domain.ErrCodeInUse()
		}
	}
	p := domain.Project{
		ID:       s.NewID().String(),
		Name:     name,
		Color:    color,
		Code:     code,
		Archived: false,
	}
	return s.Store.CreateProject(ctx, s.User, p)
}

func (s *Service) UpdateProject(ctx context.Context, id uuid.UUID, in UpdateProjectInput) (domain.Project, error) {
	p, err := s.Store.GetProject(ctx, s.User, id)
	if err != nil {
		return domain.Project{}, err
	}
	if in.Name != nil {
		name, err := domain.NormalizeName(*in.Name)
		if err != nil {
			return domain.Project{}, err
		}
		p.Name = name
	}
	if in.Color != nil {
		color, err := domain.NormalizeColor(*in.Color)
		if err != nil {
			return domain.Project{}, err
		}
		p.Color = color
	}
	if in.CodeSet {
		code, err := domain.NormalizeCode(in.Code)
		if err != nil {
			return domain.Project{}, err
		}
		p.Code = code
		if code != nil {
			used, err := s.Store.CodeInUse(ctx, s.User, *code, id)
			if err != nil {
				return domain.Project{}, err
			}
			if used {
				return domain.Project{}, domain.ErrCodeInUse()
			}
		}
	}
	return s.Store.UpdateProject(ctx, s.User, p)
}

func (s *Service) ArchiveProject(ctx context.Context, id uuid.UUID) (domain.Project, error) {
	p, err := s.Store.GetProject(ctx, s.User, id)
	if err != nil {
		return domain.Project{}, err
	}
	active, err := s.Store.CountActiveProjects(ctx, s.User)
	if err != nil {
		return domain.Project{}, err
	}
	if err := domain.CanArchive(p, active); err != nil {
		return domain.Project{}, err
	}
	p.Archived = true
	return s.Store.UpdateProject(ctx, s.User, p)
}

func (s *Service) RestoreProject(ctx context.Context, id uuid.UUID) (domain.Project, error) {
	p, err := s.Store.GetProject(ctx, s.User, id)
	if err != nil {
		return domain.Project{}, err
	}
	if err := domain.CanRestore(p); err != nil {
		return domain.Project{}, err
	}
	p.Archived = false
	return s.Store.UpdateProject(ctx, s.User, p)
}

func (s *Service) DeleteProject(ctx context.Context, id uuid.UUID) error {
	p, err := s.Store.GetProject(ctx, s.User, id)
	if err != nil {
		return err
	}
	active, err := s.Store.CountActiveProjects(ctx, s.User)
	if err != nil {
		return err
	}
	sessions, err := s.Store.CountProjectSessions(ctx, s.User, id)
	if err != nil {
		return err
	}
	if err := domain.CanHardDelete(p, active, sessions); err != nil {
		return err
	}
	return s.Store.DeleteProject(ctx, s.User, id)
}

func (s *Service) ProjectSessionCount(ctx context.Context, id uuid.UUID) (int, error) {
	if _, err := s.Store.GetProject(ctx, s.User, id); err != nil {
		return 0, err
	}
	return s.Store.CountProjectSessions(ctx, s.User, id)
}
