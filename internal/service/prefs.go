package service

import (
	"context"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/google/uuid"
)

// UpdatePrefsInput follows the present-vs-absent PATCH rule: *Set is true when the
// field was in the body, and a nil value with *Set clears it.
type UpdatePrefsInput struct {
	DailyTargetMs     *int64
	DailyTargetSet    bool
	DefaultProjectID  *string
	DefaultProjectSet bool
}

func (s *Service) GetPrefs(ctx context.Context) (domain.Prefs, error) {
	return s.Store.GetPrefs(ctx, s.User)
}

func (s *Service) UpdatePrefs(ctx context.Context, in UpdatePrefsInput) (domain.Prefs, error) {
	prefs, err := s.Store.GetPrefs(ctx, s.User)
	if err != nil {
		return domain.Prefs{}, err
	}
	if in.DailyTargetSet {
		if in.DailyTargetMs != nil {
			if err := domain.ValidateDailyTargetMs(*in.DailyTargetMs); err != nil {
				return domain.Prefs{}, err
			}
		}
		prefs.DailyTargetMs = in.DailyTargetMs
	}
	if in.DefaultProjectSet {
		prefs.DefaultProjectID = nil
		if in.DefaultProjectID != nil {
			id, err := uuid.Parse(*in.DefaultProjectID)
			if err != nil {
				return domain.Prefs{}, domain.ErrNotFound()
			}
			// Archived projects are allowed: archive does not rewrite prefs, and the
			// SPA falls back to an active project when the default is archived.
			if _, err := s.Store.GetProject(ctx, s.User, id); err != nil {
				return domain.Prefs{}, err
			}
			canonical := id.String()
			prefs.DefaultProjectID = &canonical
		}
	}
	if err := s.Store.SavePrefs(ctx, s.User, prefs); err != nil {
		return domain.Prefs{}, err
	}
	return prefs, nil
}
