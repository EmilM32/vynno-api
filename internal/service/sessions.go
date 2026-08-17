package service

import (
	"context"
	"time"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/google/uuid"
)

type StartSessionInput struct {
	ProjectID        string
	Note             string
	TicketID         *string
	ActivityType     *string
	Tags             []string
	TargetDurationMs *int64
}

func (s *Service) ListSessions(ctx context.Context, statuses []string, limit int) ([]domain.Session, error) {
	return s.Store.ListSessions(ctx, s.User, statuses, limit)
}

func (s *Service) GetSession(ctx context.Context, id uuid.UUID) (domain.Session, error) {
	return s.Store.GetSession(ctx, s.User, id)
}

func (s *Service) GetActiveSession(ctx context.Context) (domain.Session, error) {
	sess, ok, err := s.Store.GetLiveSession(ctx, s.User)
	if err != nil {
		return domain.Session{}, err
	}
	if !ok {
		return domain.Session{}, domain.ErrSessionNotActive()
	}
	return sess, nil
}

func (s *Service) StartSession(ctx context.Context, in StartSessionInput) (domain.Session, error) {
	projectID, err := uuid.Parse(in.ProjectID)
	if err != nil {
		return domain.Session{}, domain.ErrNotFound()
	}
	activity, err := domain.NormalizeActivityType(in.ActivityType)
	if err != nil {
		return domain.Session{}, err
	}
	target, err := domain.NormalizeTargetDurationMs(in.TargetDurationMs)
	if err != nil {
		return domain.Session{}, err
	}

	project, err := s.Store.GetProject(ctx, s.User, projectID)
	if err != nil {
		return domain.Session{}, err
	}
	if err := domain.CanStartSession(project); err != nil {
		return domain.Session{}, err
	}

	_, live, err := s.Store.GetLiveSession(ctx, s.User)
	if err != nil {
		return domain.Session{}, err
	}
	if live {
		return domain.Session{}, domain.ErrSessionAlreadyActive()
	}

	sess := domain.StartSession(
		s.NewID().String(),
		projectID.String(),
		in.Note,
		in.TicketID,
		activity,
		in.Tags,
		target,
		s.Now(),
	)
	return s.Store.CreateSession(ctx, s.User, sess)
}

func (s *Service) PauseSession(ctx context.Context, id uuid.UUID) (domain.Session, error) {
	return s.applyTransition(ctx, id, domain.Pause)
}

func (s *Service) ResumeSession(ctx context.Context, id uuid.UUID) (domain.Session, error) {
	return s.applyTransition(ctx, id, domain.Resume)
}

func (s *Service) StopSession(ctx context.Context, id uuid.UUID) (domain.Session, error) {
	return s.applyTransition(ctx, id, domain.Stop)
}

func (s *Service) applyTransition(ctx context.Context, id uuid.UUID, fn func(domain.Session, time.Time) (domain.Session, error)) (domain.Session, error) {
	sess, err := s.Store.GetSession(ctx, s.User, id)
	if err != nil {
		return domain.Session{}, err
	}
	next, err := fn(sess, s.Now())
	if err != nil {
		return domain.Session{}, err
	}
	return s.Store.UpdateSession(ctx, s.User, next)
}
