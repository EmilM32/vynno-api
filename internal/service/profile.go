package service

import (
	"context"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/google/uuid"
)

type UpdateProfileInput struct {
	DisplayName *string
}

func (s *Service) GetProfile(ctx context.Context) (domain.Profile, error) {
	return s.Store.GetProfile(ctx, s.User)
}

func (s *Service) UpdateProfile(ctx context.Context, in UpdateProfileInput) (domain.Profile, error) {
	if in.DisplayName != nil {
		name, err := domain.NormalizeDisplayName(*in.DisplayName)
		if err != nil {
			return domain.Profile{}, err
		}
		if err := s.Store.UpdateProfileDisplayName(ctx, s.User, name); err != nil {
			return domain.Profile{}, err
		}
	}
	return s.Store.GetProfile(ctx, s.User)
}

func (s *Service) ReplaceAvatar(ctx context.Context, data []byte) (domain.Profile, error) {
	ct, err := domain.DetectAvatarContentType(data)
	if err != nil {
		return domain.Profile{}, err
	}
	// The avatar URL is public. A photo that says where it was taken is refused
	// rather than published or silently rewritten.
	if domain.AvatarHasLocation(data, ct) {
		return domain.Profile{}, domain.ErrInvalidBody("Avatar contains GPS location data. Remove the location from the photo and upload it again.")
	}
	id := s.NewID()
	if err := s.Store.ReplaceAvatar(ctx, s.User, domain.Avatar{
		ID:          id.String(),
		ContentType: ct,
		Bytes:       data,
	}); err != nil {
		return domain.Profile{}, err
	}
	return s.Store.GetProfile(ctx, s.User)
}

func (s *Service) DeleteAvatar(ctx context.Context) (domain.Profile, error) {
	if err := s.Store.DeleteAvatarByUser(ctx, s.User); err != nil {
		return domain.Profile{}, err
	}
	return s.Store.GetProfile(ctx, s.User)
}

func (s *Service) GetAvatar(ctx context.Context, id uuid.UUID) (domain.Avatar, error) {
	return s.Store.GetAvatar(ctx, id)
}
