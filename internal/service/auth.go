package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
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
	// Wrong one-time codes. The per-challenge cap (domain.OTPMaxAttempts) resets on
	// every resend; these two do not, so resending does not buy more guesses.
	GuessIPWindow    time.Duration
	GuessIPFails     int
	GuessEmailWindow time.Duration
	GuessEmailFails  int
}

// DefaultAuthLimits are the production caps.
func DefaultAuthLimits() AuthLimits {
	return AuthLimits{
		LoginWindow:     15 * time.Minute,
		LoginEmailFails: 10,
		LoginIPFails:    30,
		SendWindow:      10 * time.Minute,
		SendIPLimit:     5,

		GuessIPWindow:    15 * time.Minute,
		GuessIPFails:     20,
		GuessEmailWindow: 24 * time.Hour,
		GuessEmailFails:  10,
	}
}

// RelaxedAuthLimits lift the per-IP caps so a playground e2e run can register an
// account per test from one address. Per-email caps and the OTP cooldown stay.
func RelaxedAuthLimits() AuthLimits {
	l := DefaultAuthLimits()
	l.LoginIPFails = 10000
	l.SendIPLimit = 10000
	l.GuessIPFails = 10000
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

// RequestPasswordReset answers the same way, after the same work, whether or not
// the address has an account: the per-email cooldown, the per-client send cap, and
// the stored challenge are identical. Only a real account gets mail, and that mail
// is sent off the request path, so neither the response time nor an SMTP failure
// tells the caller which case it was.
func (s *Service) RequestPasswordReset(ctx context.Context, email, clientIP string) error {
	normalized, err := domain.NormalizeEmail(email)
	if err != nil {
		return err
	}
	now := s.Now()
	if err := s.emailChallengeSendable(ctx, normalized, domain.PurposePasswordReset, now); err != nil {
		s.authEvent(ctx, slog.LevelWarn, "password_reset_request_limited", clientIP, uuid.Nil)
		return err
	}
	if err := s.allowSend(clientIP, now); err != nil {
		s.authEvent(ctx, slog.LevelWarn, "password_reset_request_limited", clientIP, uuid.Nil)
		return err
	}
	acc, err := s.Store.GetAccountByEmail(ctx, normalized)
	known := err == nil
	if err != nil {
		var de *domain.Error
		if !errors.As(err, &de) || de.Code != domain.CodeNotFound {
			return err
		}
	}
	code, err := s.issueOTPChallenge(ctx, normalized, domain.PurposePasswordReset, uuid.Nil)
	if err != nil {
		return err
	}
	s.Limiter.Hit(sendIPKey(clientIP), now, s.Limits.SendWindow)
	if !known {
		s.authEvent(ctx, slog.LevelInfo, "password_reset_requested_unknown_email", clientIP, uuid.Nil)
		return nil
	}
	s.authEvent(ctx, slog.LevelInfo, "password_reset_requested", clientIP, acc.ID)
	s.runAsync(ctx, func(ctx context.Context) {
		s.sendNotice(ctx, mail.Message{
			To:      normalized,
			Subject: "Your Vynno password reset code",
			Text: fmt.Sprintf(
				"Your Vynno password reset code is %s.\n\nIt expires in 15 minutes. If you did not request this, ignore this message.\n",
				code,
			),
		})
	})
	return nil
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

func guessIPKey(ip string) string { return "guess-ip:" + ip }
func guessEmailKey(purpose, email string) string {
	return "guess-email:" + purpose + ":" + email
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

func (s *Service) Register(ctx context.Context, in RegisterInput, clientIP string) (AuthResult, error) {
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

	if err := s.consumeEmailChallenge(ctx, email, domain.PurposeRegister, code, uuid.Nil, clientIP); err != nil {
		return AuthResult{}, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return AuthResult{}, err
	}
	userID := s.NewID()
	// Account, profile, and first project are one unit: a failure part-way must not
	// leave an address that is taken but cannot sign in.
	err = s.inTx(ctx, func(tx *Service) error {
		if err := tx.Store.CreateAccount(ctx, store.Account{
			ID: userID, Email: email, PasswordHash: string(hash),
		}); err != nil {
			return err
		}
		if err := tx.Store.CreateProfile(ctx, userID, domain.Profile{DisplayName: display}); err != nil {
			return err
		}
		_, err := tx.ForUser(userID).CreateProject(ctx, CreateProjectInput{
			Name: "Personal", Color: "#3b82f6",
		})
		return err
	})
	if err != nil {
		return AuthResult{}, err
	}
	s.authEvent(ctx, slog.LevelInfo, "register", clientIP, userID)
	return s.issueToken(ctx, userID, domain.RememberMe(in.RememberMe), store.ScopeSession)
}

type ResetPasswordInput struct {
	Email    string
	Code     string
	Password string
}

func (s *Service) ResetPassword(ctx context.Context, in ResetPasswordInput, clientIP string) error {
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
	if err := s.consumeEmailChallenge(ctx, email, domain.PurposePasswordReset, code, uuid.Nil, clientIP); err != nil {
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
	// The new password and the sign-out are one unit: a reset that leaves the old
	// sessions alive would not lock out whoever had them.
	err = s.inTx(ctx, func(tx *Service) error {
		if err := tx.Store.SetAccountCredentials(ctx, acc.ID, acc.Email, string(hash)); err != nil {
			return err
		}
		return tx.Store.DeleteTokensByUser(ctx, acc.ID)
	})
	if err != nil {
		return err
	}
	s.authEvent(ctx, slog.LevelInfo, "password_reset", clientIP, acc.ID)
	s.sendNotice(ctx, mail.Message{
		To:      acc.Email,
		Subject: "Your Vynno password was reset",
		Text: "The password for your Vynno account was just reset with a code sent to this address. Every device was signed out.\n\n" +
			"If you did not do this, someone else can read this mailbox: reset the password again and secure the mailbox.\n",
	})
	return nil
}

// consumeEmailChallenge spends a matching code. The guess is reserved in the store
// before the code is compared, so concurrent requests cannot make more than
// OTPMaxAttempts comparisons against one challenge. A challenge bound to another
// account, or one already spent, reads as no challenge and does not count a guess.
// A spent challenge stays in place so its send cooldown and hourly cap still hold.
//
// A resend starts a new challenge with a fresh set of guesses, so wrong codes are
// also counted per client and per email+purpose in the limiter, across resends.
// Both slots are reserved before the comparison and given back when the code was
// right; the email slot is also given back when there was no challenge to guess at.
func (s *Service) consumeEmailChallenge(ctx context.Context, email, purpose, code string, userID uuid.UUID, clientIP string) error {
	now := s.Now()
	ipKey, emailKey := guessIPKey(clientIP), guessEmailKey(purpose, email)
	if ok, retry := s.Limiter.Reserve(emailKey, now, s.Limits.GuessEmailFails, s.Limits.GuessEmailWindow); !ok {
		s.authEvent(ctx, slog.LevelWarn, "code_guess_limited_email", clientIP, userID, "purpose", purpose)
		return domain.ErrRateLimitedAfter(retry)
	}
	if ok, retry := s.Limiter.Reserve(ipKey, now, s.Limits.GuessIPFails, s.Limits.GuessIPWindow); !ok {
		s.Limiter.Release(emailKey, now)
		s.authEvent(ctx, slog.LevelWarn, "code_guess_limited_client", clientIP, userID, "purpose", purpose)
		return domain.ErrRateLimitedAfter(retry)
	}
	release := func() {
		s.Limiter.Release(emailKey, now)
		s.Limiter.Release(ipKey, now)
	}

	ch, err := s.Store.ReserveChallengeGuess(ctx, email, purpose, userID, domain.OTPMaxAttempts)
	if err != nil {
		var de *domain.Error
		if errors.As(err, &de) && de.Code == domain.CodeNotFound {
			s.Limiter.Release(emailKey, now)
			s.authEvent(ctx, slog.LevelWarn, "code_rejected", clientIP, userID, "purpose", purpose)
			return domain.ErrInvalidCode()
		}
		release()
		return err
	}
	if domain.OTPExpired(ch.ExpiresAt, now) ||
		subtle.ConstantTimeCompare([]byte(ch.CodeHash), []byte(hashToken(code))) != 1 {
		s.authEvent(ctx, slog.LevelWarn, "code_rejected", clientIP, userID, "purpose", purpose)
		if domain.OTPGuessesSpent(ch.AttemptCount) {
			return domain.ErrRateLimited()
		}
		return domain.ErrInvalidCode()
	}
	// Delete only the code we compared: a concurrent request that already used it,
	// or a resend that replaced it, leaves nothing to delete.
	consumed, err := s.Store.ConsumeEmailChallenge(ctx, email, purpose, ch.CodeHash)
	if err != nil {
		release()
		return err
	}
	release()
	if !consumed {
		return domain.ErrInvalidCode()
	}
	return nil
}

func (s *Service) Login(ctx context.Context, in LoginInput, clientIP string) (AuthResult, error) {
	return s.login(ctx, in, clientIP, store.ScopeSession)
}

// IssueMCPToken checks the password like Login and returns a token that only the
// local read-only MCP accepts. The HTTP API refuses it.
func (s *Service) IssueMCPToken(ctx context.Context, in LoginInput, clientIP string) (AuthResult, error) {
	return s.login(ctx, in, clientIP, store.ScopeMCP)
}

func (s *Service) login(ctx context.Context, in LoginInput, clientIP, scope string) (AuthResult, error) {
	email, err := domain.NormalizeEmail(in.Email)
	if err != nil {
		return AuthResult{}, domain.ErrInvalidCredentials()
	}
	now := s.Now()
	if retry, limited := s.reserveLoginAttempt(email, clientIP, now); limited {
		s.authEvent(ctx, slog.LevelWarn, "login_limited", clientIP, uuid.Nil)
		return AuthResult{}, domain.ErrRateLimitedAfter(retry)
	}
	if _, err := domain.NormalizePassword(in.Password); err != nil {
		s.authEvent(ctx, slog.LevelWarn, "login_failed", clientIP, uuid.Nil)
		return AuthResult{}, domain.ErrInvalidCredentials()
	}
	acc, err := s.Store.GetAccountByEmail(ctx, email)
	if err != nil {
		var de *domain.Error
		if errors.As(err, &de) && de.Code == domain.CodeNotFound {
			_ = comparePassword(dummyPasswordHash(), []byte(in.Password))
			s.authEvent(ctx, slog.LevelWarn, "login_failed", clientIP, uuid.Nil)
			return AuthResult{}, domain.ErrInvalidCredentials()
		}
		// The store failed, so nothing was learned about the password. Report the
		// failure as it is (500, logged) instead of blaming the credentials.
		s.releaseLoginAttempt(email, clientIP, now)
		return AuthResult{}, err
	}
	if acc.PasswordHash == "" {
		_ = comparePassword(dummyPasswordHash(), []byte(in.Password))
		s.authEvent(ctx, slog.LevelWarn, "login_failed", clientIP, acc.ID)
		return AuthResult{}, domain.ErrInvalidCredentials()
	}
	if err := comparePassword([]byte(acc.PasswordHash), []byte(in.Password)); err != nil {
		s.authEvent(ctx, slog.LevelWarn, "login_failed", clientIP, acc.ID)
		return AuthResult{}, domain.ErrInvalidCredentials()
	}
	s.loginSucceeded(email, clientIP, now)
	s.authEvent(ctx, slog.LevelInfo, "login", clientIP, acc.ID, "scope", scope)
	return s.issueToken(ctx, acc.ID, domain.RememberMe(in.RememberMe), scope)
}

func (s *Service) Logout(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return domain.ErrUnauthorized()
	}
	return s.Store.DeleteTokenByHash(ctx, hashToken(rawToken))
}

// ResolveToken is the HTTP API's check: the cookie or Bearer value must be a live
// session token. A token issued for the MCP is refused like an unknown one.
func (s *Service) ResolveToken(ctx context.Context, rawToken string) (uuid.UUID, error) {
	return s.resolveToken(ctx, rawToken, false)
}

// ResolveMCPToken is the local MCP's check. It accepts an MCP token, and also a
// session token, which is what `cmd/mcp token` printed before scopes existed.
func (s *Service) ResolveMCPToken(ctx context.Context, rawToken string) (uuid.UUID, error) {
	return s.resolveToken(ctx, rawToken, true)
}

func (s *Service) resolveToken(ctx context.Context, rawToken string, allowMCP bool) (uuid.UUID, error) {
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
	if tok.Scope != store.ScopeSession && (!allowMCP || tok.Scope != store.ScopeMCP) {
		return uuid.Nil, domain.ErrUnauthorized()
	}
	return tok.UserID, nil
}

func (s *Service) issueToken(ctx context.Context, userID uuid.UUID, remember bool, scope string) (AuthResult, error) {
	raw, err := randomToken()
	if err != nil {
		return AuthResult{}, err
	}
	if err := s.Store.CreateToken(ctx, store.Token{
		ID:        s.NewID(),
		UserID:    userID,
		TokenHash: hashToken(raw),
		ExpiresAt: s.Now().Add(TokenTTL),
		Scope:     scope,
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
