package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/EmilM32/vynno-api/internal/mail"
	"github.com/EmilM32/vynno-api/internal/store"
	"golang.org/x/crypto/bcrypt"
)

// Credential changes for a signed-in user. Each one re-checks the password (or a
// code sent to the new address), keeps the caller's session, and signs out every
// other session of the account.

type ChangePasswordInput struct {
	CurrentPassword string
	NewPassword     string
}

type RequestEmailChangeInput struct {
	Email    string
	Password string
}

type ChangeEmailInput struct {
	Email string
	Code  string
}

// verifyAccountPassword re-checks a signed-in user's password. Failures count
// against the same per-email and per-client caps as login, so a stolen session
// cookie cannot be used to guess the password.
func (s *Service) verifyAccountPassword(acc store.Account, password, clientIP string, now time.Time) error {
	if retry, limited := s.loginLimited(acc.Email, clientIP, now); limited {
		return domain.ErrRateLimitedAfter(retry)
	}
	if _, err := domain.NormalizePassword(password); err != nil || acc.PasswordHash == "" {
		s.recordLoginFailure(acc.Email, clientIP, now)
		return domain.ErrInvalidCredentials()
	}
	if err := comparePassword([]byte(acc.PasswordHash), []byte(password)); err != nil {
		s.recordLoginFailure(acc.Email, clientIP, now)
		return domain.ErrInvalidCredentials()
	}
	s.Limiter.Reset(loginEmailKey(acc.Email))
	return nil
}

// ChangePassword sets a new password after checking the current one.
func (s *Service) ChangePassword(ctx context.Context, in ChangePasswordInput, rawToken, clientIP string) error {
	newPassword, err := domain.NormalizePassword(in.NewPassword)
	if err != nil {
		return err
	}
	acc, err := s.Store.GetAccountByID(ctx, s.User)
	if err != nil {
		return err
	}
	if err := s.verifyAccountPassword(acc, in.CurrentPassword, clientIP, s.Now()); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.Store.SetAccountCredentials(ctx, acc.ID, acc.Email, string(hash)); err != nil {
		return err
	}
	if err := s.Store.DeleteOtherTokens(ctx, acc.ID, hashToken(rawToken)); err != nil {
		return err
	}
	s.sendNotice(ctx, mail.Message{
		To:      acc.Email,
		Subject: "Your Vynno password was changed",
		Text: "The password for your Vynno account was just changed. Other devices were signed out.\n\n" +
			"If you did not do this, reset your password from the Vynno sign-in page.\n",
	})
	return nil
}

// RequestEmailChange mails a code to the new address after checking the password.
func (s *Service) RequestEmailChange(ctx context.Context, in RequestEmailChangeInput, clientIP string) error {
	newEmail, err := domain.NormalizeEmail(in.Email)
	if err != nil {
		return err
	}
	acc, err := s.Store.GetAccountByID(ctx, s.User)
	if err != nil {
		return err
	}
	if newEmail == acc.Email {
		return domain.ErrInvalidBody("That is already your email.")
	}
	now := s.Now()
	if err := s.verifyAccountPassword(acc, in.Password, clientIP, now); err != nil {
		return err
	}
	taken, err := s.Store.EmailTaken(ctx, newEmail, acc.ID)
	if err != nil {
		return err
	}
	if taken {
		return domain.ErrEmailInUse()
	}
	if err := s.emailChallengeSendable(ctx, newEmail, domain.PurposeChangeEmail, now); err != nil {
		return err
	}
	if err := s.allowSend(clientIP, now); err != nil {
		return err
	}
	code, err := s.issueOTPChallenge(ctx, newEmail, domain.PurposeChangeEmail, acc.ID)
	if err != nil {
		return err
	}
	s.Limiter.Hit(sendIPKey(clientIP), now, s.Limits.SendWindow)
	return s.Mailer.Send(ctx, mail.Message{
		To:      newEmail,
		Subject: "Your Vynno email change code",
		Text: fmt.Sprintf(
			"Your code to use this address for Vynno is %s.\n\nIt expires in 15 minutes. If you did not request this, ignore this message.\n",
			code,
		),
	})
}

// ChangeEmail switches the sign-in email once the code sent to it is confirmed.
func (s *Service) ChangeEmail(ctx context.Context, in ChangeEmailInput, rawToken string) (domain.Profile, error) {
	newEmail, err := domain.NormalizeEmail(in.Email)
	if err != nil {
		return domain.Profile{}, err
	}
	code, err := domain.NormalizeOTP(in.Code)
	if err != nil {
		return domain.Profile{}, err
	}
	acc, err := s.Store.GetAccountByID(ctx, s.User)
	if err != nil {
		return domain.Profile{}, err
	}
	if err := s.consumeEmailChallenge(ctx, newEmail, domain.PurposeChangeEmail, code, acc.ID); err != nil {
		return domain.Profile{}, err
	}
	// Someone may have registered the address since the code was sent. The unique
	// index is the final word; SetAccountCredentials maps it to email_in_use too.
	taken, err := s.Store.EmailTaken(ctx, newEmail, acc.ID)
	if err != nil {
		return domain.Profile{}, err
	}
	if taken {
		return domain.Profile{}, domain.ErrEmailInUse()
	}
	if err := s.Store.SetAccountCredentials(ctx, acc.ID, newEmail, acc.PasswordHash); err != nil {
		return domain.Profile{}, err
	}
	if err := s.Store.DeleteOtherTokens(ctx, acc.ID, hashToken(rawToken)); err != nil {
		return domain.Profile{}, err
	}
	s.sendNotice(ctx, mail.Message{
		To:      acc.Email,
		Subject: "Your Vynno sign-in email was changed",
		Text: fmt.Sprintf(
			"Your Vynno account now signs in with %s instead of this address. Other devices were signed out.\n\n"+
				"If you did not do this, contact whoever runs your Vynno server.\n",
			newEmail,
		),
	})
	return s.Store.GetProfile(ctx, acc.ID)
}

// sendNotice mails a security notice. The change it reports has already happened,
// so a mail failure is logged instead of failing the request.
func (s *Service) sendNotice(ctx context.Context, msg mail.Message) {
	if err := s.Mailer.Send(ctx, msg); err != nil {
		slog.WarnContext(ctx, "security notice mail failed", "subject", msg.Subject, "err", err)
	}
}
