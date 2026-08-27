package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/EmilM32/vynno-api/internal/mail"
	"github.com/EmilM32/vynno-api/internal/store"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const TokenTTL = 30 * 24 * time.Hour

type RegisterInput struct {
	Email       string
	Password    string
	Code        string
	DisplayName *string
	RememberMe  *bool
}

type LoginInput struct {
	Email      string
	Password   string
	RememberMe *bool
}

type AuthResult struct {
	Token      string
	RememberMe bool
	Profile    domain.Profile
}

func (s *Service) RequestRegisterCode(ctx context.Context, email string) error {
	normalized, err := domain.NormalizeEmail(email)
	if err != nil {
		return err
	}
	taken, err := s.Store.EmailTaken(ctx, normalized, uuid.Nil)
	if err != nil {
		return err
	}
	if taken {
		return domain.ErrEmailInUse()
	}
	code, err := s.issueOTPChallenge(ctx, normalized, domain.PurposeRegister)
	if err != nil {
		return err
	}
	return s.Mailer.Send(ctx, mail.Message{
		To:      normalized,
		Subject: "Your Vynno confirmation code",
		Text: fmt.Sprintf(
			"Your Vynno confirmation code is %s.\n\nIt expires in 15 minutes. If you did not request this, ignore this message.\n",
			code,
		),
	})
}

func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	normalized, err := domain.NormalizeEmail(email)
	if err != nil {
		return err
	}
	code, err := s.issueOTPChallenge(ctx, normalized, domain.PurposePasswordReset)
	if err != nil {
		return err
	}
	if _, err := s.Store.GetAccountByEmail(ctx, normalized); err != nil {
		var de *domain.Error
		if errors.As(err, &de) && de.Code == domain.CodeNotFound {
			return nil
		}
		return err
	}
	return s.Mailer.Send(ctx, mail.Message{
		To:      normalized,
		Subject: "Your Vynno password reset code",
		Text: fmt.Sprintf(
			"Your Vynno password reset code is %s.\n\nIt expires in 15 minutes. If you did not request this, ignore this message.\n",
			code,
		),
	})
}

func (s *Service) issueOTPChallenge(ctx context.Context, email, purpose string) (string, error) {
	now := s.Now()
	ch, err := s.Store.GetEmailChallenge(ctx, email, purpose)
	if err != nil {
		var de *domain.Error
		if !errors.As(err, &de) || de.Code != domain.CodeNotFound {
			return "", err
		}
		ch = store.EmailChallenge{
			Email:           email,
			Purpose:         purpose,
			SendCount:       0,
			SendWindowStart: now,
		}
	} else if domain.OTPSendCooldownActive(ch.SentAt, now) {
		return "", domain.ErrRateLimited()
	}

	windowStart, sendCount := domain.AdvanceSendWindow(ch.SendWindowStart, ch.SendCount, now)
	if domain.OTPSendLimited(sendCount) {
		return "", domain.ErrRateLimited()
	}

	code, err := domain.GenerateOTP()
	if err != nil {
		return "", err
	}
	ch.Email = email
	ch.Purpose = purpose
	ch.CodeHash = hashToken(code)
	ch.ExpiresAt = now.Add(domain.OTPTTL)
	ch.AttemptCount = 0
	ch.SentAt = now
	ch.SendCount = sendCount + 1
	ch.SendWindowStart = windowStart
	if err := s.Store.UpsertEmailChallenge(ctx, ch); err != nil {
		return "", err
	}
	return code, nil
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (AuthResult, error) {
	email, err := domain.NormalizeEmail(in.Email)
	if err != nil {
		return AuthResult{}, err
	}
	password, err := domain.NormalizePassword(in.Password)
	if err != nil {
		return AuthResult{}, err
	}
	code, err := domain.NormalizeOTP(in.Code)
	if err != nil {
		return AuthResult{}, err
	}
	display := ""
	if in.DisplayName != nil {
		d, err := domain.NormalizeDisplayName(*in.DisplayName)
		if err != nil {
			return AuthResult{}, err
		}
		display = d
	}

	taken, err := s.Store.EmailTaken(ctx, email, uuid.Nil)
	if err != nil {
		return AuthResult{}, err
	}
	if taken {
		return AuthResult{}, domain.ErrEmailInUse()
	}

	if err := s.consumeEmailChallenge(ctx, email, domain.PurposeRegister, code); err != nil {
		return AuthResult{}, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return AuthResult{}, err
	}
	userID := s.NewID()
	if err := s.Store.CreateAccount(ctx, store.Account{
		ID: userID, Email: email, PasswordHash: string(hash),
	}); err != nil {
		return AuthResult{}, err
	}
	profile := domain.Profile{
		DisplayName: display,
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

type ResetPasswordInput struct {
	Email    string
	Code     string
	Password string
}

func (s *Service) ResetPassword(ctx context.Context, in ResetPasswordInput) error {
	email, err := domain.NormalizeEmail(in.Email)
	if err != nil {
		return err
	}
	password, err := domain.NormalizePassword(in.Password)
	if err != nil {
		return err
	}
	code, err := domain.NormalizeOTP(in.Code)
	if err != nil {
		return err
	}
	if err := s.consumeEmailChallenge(ctx, email, domain.PurposePasswordReset, code); err != nil {
		return err
	}
	acc, err := s.Store.GetAccountByEmail(ctx, email)
	if err != nil {
		var de *domain.Error
		if errors.As(err, &de) && de.Code == domain.CodeNotFound {
			return domain.ErrInvalidCode()
		}
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.Store.SetAccountCredentials(ctx, acc.ID, acc.Email, string(hash)); err != nil {
		return err
	}
	return s.Store.DeleteTokensByUser(ctx, acc.ID)
}

func (s *Service) consumeEmailChallenge(ctx context.Context, email, purpose, code string) error {
	ch, err := s.Store.GetEmailChallenge(ctx, email, purpose)
	if err != nil {
		var de *domain.Error
		if errors.As(err, &de) && de.Code == domain.CodeNotFound {
			return domain.ErrInvalidCode()
		}
		return err
	}
	if domain.OTPGuessesSpent(ch.AttemptCount) {
		_ = s.Store.DeleteEmailChallenge(ctx, email, purpose)
		return domain.ErrInvalidCode()
	}
	if domain.OTPExpired(ch.ExpiresAt, s.Now()) || ch.CodeHash != hashToken(code) {
		n, incErr := s.Store.IncrementChallengeAttempts(ctx, email, purpose)
		if incErr != nil {
			return incErr
		}
		if domain.OTPGuessesSpent(n) {
			_ = s.Store.DeleteEmailChallenge(ctx, email, purpose)
			return domain.ErrRateLimited()
		}
		return domain.ErrInvalidCode()
	}
	return s.Store.DeleteEmailChallenge(ctx, email, purpose)
}

func (s *Service) Login(ctx context.Context, in LoginInput) (AuthResult, error) {
	email, err := domain.NormalizeEmail(in.Email)
	if err != nil {
		return AuthResult{}, domain.ErrInvalidCredentials()
	}
	if _, err := domain.NormalizePassword(in.Password); err != nil {
		return AuthResult{}, domain.ErrInvalidCredentials()
	}
	acc, err := s.Store.GetAccountByEmail(ctx, email)
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
