package service

import (
	"context"

	"github.com/EmilM32/vynno-api/internal/domain"
)

// DayTotals is tracked time per local date, project, and activity type for stopped
// sessions in the range. The live session is left to the client, which ticks it.
func (s *Service) DayTotals(ctx context.Context, r domain.DayRange) ([]domain.DayTotal, error) {
	from, to := r.QueryWindow()
	sessions, err := s.Store.ListStoppedSessionsStartedBetween(ctx, s.User, from, to)
	if err != nil {
		return nil, err
	}
	return domain.BucketDayTotals(sessions, r), nil
}
