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
	ActivityTypeID   *string
	Tags             []string
	TargetDurationMs *int64
}

type CreateManualSessionInput struct {
	ProjectID        string
	Note             string
	TicketID         *string
	ActivityTypeID   *string
	Tags             []string
	TargetDurationMs *int64
	StartedAt        time.Time
	EndedAt          time.Time
	PausedMs         *int64
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
	activityID, err := s.resolveActivityTypeID(ctx, in.ActivityTypeID)
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
		activityID,
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

func (s *Service) UpdateSession(ctx context.Context, id uuid.UUID, patch domain.SessionPatch) (domain.Session, error) {
	sess, err := s.Store.GetSession(ctx, s.User, id)
	if err != nil {
		return domain.Session{}, err
	}
	if patch.ProjectID != nil {
		pid, err := s.resolveProjectID(ctx, *patch.ProjectID)
		if err != nil {
			return domain.Session{}, err
		}
		patch.ProjectID = &pid
	}
	if patch.ActivityTypeSet {
		activityID, err := s.resolveActivityTypeID(ctx, patch.ActivityTypeID)
		if err != nil {
			return domain.Session{}, err
		}
		patch.ActivityTypeID = activityID
	}
	next, err := domain.ApplySessionPatch(sess, patch, s.Now())
	if err != nil {
		return domain.Session{}, err
	}
	return s.Store.UpdateSession(ctx, s.User, next)
}

func (s *Service) DeleteSession(ctx context.Context, id uuid.UUID) error {
	if _, err := s.Store.GetSession(ctx, s.User, id); err != nil {
		return err
	}
	return s.Store.DeleteSession(ctx, s.User, id)
}

func (s *Service) CreateManualSession(ctx context.Context, in CreateManualSessionInput) (domain.Session, error) {
	projectID, err := s.resolveProjectID(ctx, in.ProjectID)
	if err != nil {
		return domain.Session{}, err
	}
	activityID, err := s.resolveActivityTypeID(ctx, in.ActivityTypeID)
	if err != nil {
		return domain.Session{}, err
	}
	target, err := domain.NormalizeTargetDurationMs(in.TargetDurationMs)
	if err != nil {
		return domain.Session{}, err
	}
	paused := int64(0)
	if in.PausedMs != nil {
		paused = *in.PausedMs
	}
	sess, err := domain.ManualSession(
		s.NewID().String(),
		projectID,
		in.Note,
		in.TicketID,
		activityID,
		in.Tags,
		target,
		in.StartedAt,
		in.EndedAt,
		paused,
	)
	if err != nil {
		return domain.Session{}, err
	}
	return s.Store.CreateSession(ctx, s.User, sess)
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

func (s *Service) resolveProjectID(ctx context.Context, raw string) (string, error) {
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return "", domain.ErrNotFound()
	}
	if _, err := s.Store.GetProject(ctx, s.User, parsed); err != nil {
		return "", err
	}
	return parsed.String(), nil
}

func (s *Service) resolveActivityTypeID(ctx context.Context, raw *string) (*string, error) {
	id := domain.NormalizeOptionalString(raw)
	if id == nil {
		return nil, nil
	}
	parsed, err := uuid.Parse(*id)
	if err != nil {
		return nil, domain.ErrNotFound()
	}
	if _, err := s.Store.GetActivityType(ctx, s.User, parsed); err != nil {
		return nil, err
	}
	v := parsed.String()
	return &v, nil
}
