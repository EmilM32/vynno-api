package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/EmilM32/vynno-api/internal/mail"
	"github.com/EmilM32/vynno-api/internal/store"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const TokenTTL = 30 * 24 * time.Hour

// AuthLimits are the in-memory login and one-time-code send caps.
type AuthLimits struct {
	LoginWindow     time.Duration
	LoginEmailFails int
	LoginIPFails    int
	SendWindow      time.Duration
	SendIPLimit     int
}

// DefaultAuthLimits are the production caps.
func DefaultAuthLimits() AuthLimits {
	return AuthLimits{
		LoginWindow:     15 * time.Minute,
		LoginEmailFails: 10,
		LoginIPFails:    30,
		SendWindow:      10 * time.Minute,
		SendIPLimit:     5,
	}
}

// RelaxedAuthLimits lift the per-IP caps so a playground e2e run can register an
// account per test from one address. Per-email caps and the OTP cooldown stay.
func RelaxedAuthLimits() AuthLimits {
	l := DefaultAuthLimits()
	l.LoginIPFails = 10000
	l.SendIPLimit = 10000
	return l
}

// comparePassword is bcrypt.CompareHashAndPassword so tests can count calls.
var comparePassword = bcrypt.CompareHashAndPassword

var (
	dummyHashOnce sync.Once
	dummyHash     []byte
)

func dummyPasswordHash() []byte {
	dummyHashOnce.Do(func() {
		h, err := bcrypt.GenerateFromPassword([]byte("vynno-dummy-password"), bcrypt.DefaultCost)
		if err != nil {
			panic("dummy bcrypt: " + err.Error())
		}
		dummyHash = h
	})
	return dummyHash
}

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

func (s *Service) RequestRegisterCode(ctx context.Context, email, clientIP string) error {
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
	now := s.Now()
	if err := s.emailChallengeSendable(ctx, normalized, domain.PurposeRegister, now); err != nil {
		return err
	}
	if err := s.allowSend(clientIP, now); err != nil {
		return err
	}
	code, err := s.issueOTPChallenge(ctx, normalized, domain.PurposeRegister, uuid.Nil)
	if err != nil {
		return err
	}
	s.Limiter.Hit(sendIPKey(clientIP), now, s.Limits.SendWindow)
	return s.Mailer.Send(ctx, mail.Message{
		To:      normalized,
		Subject: "Your Vynno confirmation code",
		Text: fmt.Sprintf(
			"Your Vynno confirmation code is %s.\n\nIt expires in 15 minutes. If you did not request this, ignore this message.\n",
			code,
		),
	})
}

func (s *Service) RequestPasswordReset(ctx context.Context, email, clientIP string) error {
	normalized, err := domain.NormalizeEmail(email)
	if err != nil {
		return err
	}
	_, err = s.Store.GetAccountByEmail(ctx, normalized)
	if err != nil {
		var de *domain.Error
		if !errors.As(err, &de) || de.Code != domain.CodeNotFound {
			return err
		}
		// Unknown addresses do not send mail and do not consume the client-IP cap.
		// The per-email challenge is still issued so cooldown matches a real account.
		_, err = s.issueOTPChallenge(ctx, normalized, domain.PurposePasswordReset, uuid.Nil)
		return err
	}
	now := s.Now()
	if err := s.emailChallengeSendable(ctx, normalized, domain.PurposePasswordReset, now); err != nil {
		return err
	}
	if err := s.allowSend(clientIP, now); err != nil {
		return err
	}
	code, err := s.issueOTPChallenge(ctx, normalized, domain.PurposePasswordReset, uuid.Nil)
	if err != nil {
		return err
	}
	s.Limiter.Hit(sendIPKey(clientIP), now, s.Limits.SendWindow)
	return s.Mailer.Send(ctx, mail.Message{
		To:      normalized,
		Subject: "Your Vynno password reset code",
		Text: fmt.Sprintf(
			"Your Vynno password reset code is %s.\n\nIt expires in 15 minutes. If you did not request this, ignore this message.\n",
			code,
		),
	})
}

func (s *Service) emailChallengeSendable(ctx context.Context, email, purpose string, now time.Time) error {
	ch, err := s.Store.GetEmailChallenge(ctx, email, purpose)
	if err != nil {
		var de *domain.Error
		if errors.As(err, &de) && de.Code == domain.CodeNotFound {
			return nil
		}
		return err
	}
	if domain.OTPSendCooldownActive(ch.SentAt, now) {
		return domain.ErrRateLimitedAfter(domain.OTPSendCooldownRemaining(ch.SentAt, now))
	}
	windowStart, sendCount := domain.AdvanceSendWindow(ch.SendWindowStart, ch.SendCount, now)
	if domain.OTPSendLimited(sendCount) {
		return domain.ErrRateLimitedAfter(domain.OTPSendWindowRemaining(windowStart, now))
	}
	return nil
}

func (s *Service) allowSend(clientIP string, now time.Time) error {
	ok, retry := s.Limiter.Allow(sendIPKey(clientIP), now, s.Limits.SendIPLimit, s.Limits.SendWindow)
	if !ok {
		return domain.ErrRateLimitedAfter(retry)
	}
	return nil
}

func loginEmailKey(email string) string { return "login-email:" + email }
func loginIPKey(ip string) string       { return "login-ip:" + ip }
func sendIPKey(ip string) string        { return "send-ip:" + ip }

// reserveLoginAttempt counts this attempt against the per-email and per-client caps
// before the password is compared. Reserving first means concurrent guesses cannot
// all pass the check while bcrypt runs. A success gives the slots back with
// loginSucceeded; a failure keeps them.
func (s *Service) reserveLoginAttempt(email, clientIP string, now time.Time) (time.Duration, bool) {
	if ok, retry := s.Limiter.Reserve(loginEmailKey(email), now, s.Limits.LoginEmailFails, s.Limits.LoginWindow); !ok {
		return retry, true
	}
	if ok, retry := s.Limiter.Reserve(loginIPKey(clientIP), now, s.Limits.LoginIPFails, s.Limits.LoginWindow); !ok {
		s.Limiter.Release(loginEmailKey(email), now)
		return retry, true
	}
	return 0, false
}

// releaseLoginAttempt returns both slots when the attempt ended without a verdict
// on the password (a store error).
func (s *Service) releaseLoginAttempt(email, clientIP string, now time.Time) {
	s.Limiter.Release(loginEmailKey(email), now)
	s.Limiter.Release(loginIPKey(clientIP), now)
}

// loginSucceeded clears the per-email failures and returns this attempt's client slot.
func (s *Service) loginSucceeded(email, clientIP string, now time.Time) {
	s.Limiter.Reset(loginEmailKey(email))
	s.Limiter.Release(loginIPKey(clientIP), now)
}

// issueOTPChallenge stores a fresh code for email+purpose. userID binds a change_email
// challenge to the account that asked; register and reset pass uuid.Nil.
func (s *Service) issueOTPChallenge(ctx context.Context, email, purpose string, userID uuid.UUID) (string, error) {
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
		return "", domain.ErrRateLimitedAfter(domain.OTPSendCooldownRemaining(ch.SentAt, now))
	}

	windowStart, sendCount := domain.AdvanceSendWindow(ch.SendWindowStart, ch.SendCount, now)
	if domain.OTPSendLimited(sendCount) {
		return "", domain.ErrRateLimitedAfter(domain.OTPSendWindowRemaining(windowStart, now))
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
	ch.UserID = userID
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

	if err := s.consumeEmailChallenge(ctx, email, domain.PurposeRegister, code, uuid.Nil); err != nil {
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
	if err := s.consumeEmailChallenge(ctx, email, domain.PurposePasswordReset, code, uuid.Nil); err != nil {
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

// consumeEmailChallenge spends a matching code. The guess is reserved in the store
// before the code is compared, so concurrent requests cannot make more than
// OTPMaxAttempts comparisons against one challenge. A challenge bound to another
// account, or one already spent, reads as no challenge and does not count a guess.
// A spent challenge stays in place so its send cooldown and hourly cap still hold.
func (s *Service) consumeEmailChallenge(ctx context.Context, email, purpose, code string, userID uuid.UUID) error {
	ch, err := s.Store.ReserveChallengeGuess(ctx, email, purpose, userID, domain.OTPMaxAttempts)
	if err != nil {
		var de *domain.Error
		if errors.As(err, &de) && de.Code == domain.CodeNotFound {
			return domain.ErrInvalidCode()
		}
		return err
	}
	if domain.OTPExpired(ch.ExpiresAt, s.Now()) ||
		subtle.ConstantTimeCompare([]byte(ch.CodeHash), []byte(hashToken(code))) != 1 {
		if domain.OTPGuessesSpent(ch.AttemptCount) {
			return domain.ErrRateLimited()
		}
		return domain.ErrInvalidCode()
	}
	// Delete only the code we compared: a concurrent request that already used it,
	// or a resend that replaced it, leaves nothing to delete.
	consumed, err := s.Store.ConsumeEmailChallenge(ctx, email, purpose, ch.CodeHash)
	if err != nil {
		return err
	}
	if !consumed {
		return domain.ErrInvalidCode()
	}
	return nil
}

func (s *Service) Login(ctx context.Context, in LoginInput, clientIP string) (AuthResult, error) {
	email, err := domain.NormalizeEmail(in.Email)
	if err != nil {
		return AuthResult{}, domain.ErrInvalidCredentials()
	}
	now := s.Now()
	if retry, limited := s.reserveLoginAttempt(email, clientIP, now); limited {
		return AuthResult{}, domain.ErrRateLimitedAfter(retry)
	}
	if _, err := domain.NormalizePassword(in.Password); err != nil {
		return AuthResult{}, domain.ErrInvalidCredentials()
	}
	acc, err := s.Store.GetAccountByEmail(ctx, email)
	if err != nil {
		var de *domain.Error
		if errors.As(err, &de) && de.Code == domain.CodeNotFound {
			_ = comparePassword(dummyPasswordHash(), []byte(in.Password))
			return AuthResult{}, domain.ErrInvalidCredentials()
		}
		s.releaseLoginAttempt(email, clientIP, now)
		return AuthResult{}, domain.ErrInvalidCredentials()
	}
	if acc.PasswordHash == "" {
		_ = comparePassword(dummyPasswordHash(), []byte(in.Password))
		return AuthResult{}, domain.ErrInvalidCredentials()
	}
	if err := comparePassword([]byte(acc.PasswordHash), []byte(in.Password)); err != nil {
		return AuthResult{}, domain.ErrInvalidCredentials()
	}
	s.loginSucceeded(email, clientIP, now)
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
