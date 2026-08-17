package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/EmilM32/vynno-api/internal/store"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const TokenTTL = 30 * 24 * time.Hour

type RegisterInput struct {
	Username    string
	Password    string
	DisplayName *string
	RememberMe  *bool
}

type LoginInput struct {
	Username   string
	Password   string
	RememberMe *bool
}

type AuthResult struct {
	Token      string
	RememberMe bool
	Profile    domain.Profile
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (AuthResult, error) {
	username, err := domain.NormalizeUsername(in.Username)
	if err != nil {
		return AuthResult{}, err
	}
	password, err := domain.NormalizePassword(in.Password)
	if err != nil {
		return AuthResult{}, err
	}
	display := username
	if in.DisplayName != nil {
		d, err := domain.NormalizeDisplayName(*in.DisplayName)
		if err != nil {
			return AuthResult{}, err
		}
		if d != "" {
			display = d
		}
	}

	taken, err := s.Store.UsernameTaken(ctx, username, uuid.Nil)
	if err != nil {
		return AuthResult{}, err
	}
	if taken {
		return AuthResult{}, domain.ErrUsernameInUse()
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return AuthResult{}, err
	}
	userID := s.NewID()
	if err := s.Store.CreateAccount(ctx, store.Account{
		ID: userID, Username: username, PasswordHash: string(hash),
	}); err != nil {
		return AuthResult{}, err
	}
	profile := domain.Profile{
		DisplayName: display,
		Handle:      domain.HandleFromUsername(username),
	}
	if err := s.Store.CreateProfile(ctx, userID, profile); err != nil {
		return AuthResult{}, err
	}
	if _, err := s.ForUser(userID).CreateProject(ctx, CreateProjectInput{
		Name: "Personal", Color: "#3b82f6",
	}); err != nil {
		return AuthResult{}, err
	}
	return s.issueToken(ctx, userID, domain.RememberMe(in.RememberMe))
}

func (s *Service) Login(ctx context.Context, in LoginInput) (AuthResult, error) {
	username, err := domain.NormalizeUsername(in.Username)
	if err != nil {
		return AuthResult{}, domain.ErrInvalidCredentials()
	}
	if _, err := domain.NormalizePassword(in.Password); err != nil {
		return AuthResult{}, domain.ErrInvalidCredentials()
	}
	acc, err := s.Store.GetAccountByUsername(ctx, username)
	if err != nil {
		return AuthResult{}, domain.ErrInvalidCredentials()
	}
	if acc.PasswordHash == "" {
		return AuthResult{}, domain.ErrInvalidCredentials()
	}
	if err := bcrypt.CompareHashAndPassword([]byte(acc.PasswordHash), []byte(in.Password)); err != nil {
		return AuthResult{}, domain.ErrInvalidCredentials()
	}
	return s.issueToken(ctx, acc.ID, domain.RememberMe(in.RememberMe))
}

func (s *Service) Logout(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return domain.ErrUnauthorized()
	}
	return s.Store.DeleteTokenByHash(ctx, hashToken(rawToken))
}

func (s *Service) ResolveToken(ctx context.Context, rawToken string) (uuid.UUID, error) {
	if rawToken == "" {
		return uuid.Nil, domain.ErrUnauthorized()
	}
	tok, err := s.Store.GetTokenByHash(ctx, hashToken(rawToken))
	if err != nil {
		var de *domain.Error
		if errors.As(err, &de) && de.Code == domain.CodeNotFound {
			return uuid.Nil, domain.ErrUnauthorized()
		}
		return uuid.Nil, err
	}
	if !tok.ExpiresAt.After(s.Now()) {
		_ = s.Store.DeleteTokenByHash(ctx, tok.TokenHash)
		return uuid.Nil, domain.ErrUnauthorized()
	}
	return tok.UserID, nil
}

func (s *Service) issueToken(ctx context.Context, userID uuid.UUID, remember bool) (AuthResult, error) {
	raw, err := randomToken()
	if err != nil {
		return AuthResult{}, err
	}
	if err := s.Store.CreateToken(ctx, store.Token{
		ID:        s.NewID(),
		UserID:    userID,
		TokenHash: hashToken(raw),
		ExpiresAt: s.Now().Add(TokenTTL),
	}); err != nil {
		return AuthResult{}, err
	}
	profile, err := s.Store.GetProfile(ctx, userID)
	if err != nil {
		return AuthResult{}, err
	}
	return AuthResult{Token: raw, RememberMe: remember, Profile: profile}, nil
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
