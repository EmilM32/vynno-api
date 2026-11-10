package service

import (
	"context"
	"time"

	"github.com/EmilM32/vynno-api/internal/domain"
)

// SweepInterval is how often the API process removes expired auth rows.
const SweepInterval = time.Hour

// challengeRetention keeps an expired challenge for one send window past its
// expiry. The row carries the send cooldown and the hourly send count, and the
// window can end up to OTPSendWindow after the last send, so deleting it sooner
// would reset the send cap.
const challengeRetention = domain.OTPSendWindow

// SweepResult counts the rows one sweep removed.
type SweepResult struct {
	Tokens     int64
	Challenges int64
}

// SweepExpired deletes expired session tokens and stale one-time code challenges.
// Expired rows are already refused on use; this only keeps the tables from growing.
func (s *Service) SweepExpired(ctx context.Context) (SweepResult, error) {
	now := s.Now()
	s.Limiter.Sweep(now, s.longestLimitWindow())
	tokens, err := s.Store.DeleteExpiredTokens(ctx, now)
	if err != nil {
		return SweepResult{}, err
	}
	challenges, err := s.Store.DeleteEmailChallengesExpiredBefore(ctx, now.Add(-challengeRetention))
	if err != nil {
		return SweepResult{Tokens: tokens}, err
	}
	return SweepResult{Tokens: tokens, Challenges: challenges}, nil
}

// longestLimitWindow is the longest window any limiter key is counted over. A key
// with no event that recent is empty for every cap and can be dropped.
func (s *Service) longestLimitWindow() time.Duration {
	return max(s.Limits.LoginWindow, s.Limits.SendWindow, s.Limits.GuessIPWindow, s.Limits.GuessEmailWindow)
}
