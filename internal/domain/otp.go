package domain

import (
	"crypto/rand"
	"strings"
	"time"
)

const (
	PurposeRegister           = "register"
	PurposePasswordReset      = "password_reset"
	OTPTTL                    = 15 * time.Minute
	OTPSendCooldown           = 60 * time.Second
	OTPSendWindow             = time.Hour
	OTPSendsPerHour           = 5
	OTPMaxAttempts            = 5
	otpDigits                 = 6
	otpDigitMod               = 10
	otpUnbiasedMax       byte = 250 // 25 * 10; reject 250–255 to avoid %10 bias
)

// GenerateOTP returns a cryptographically random 6-digit code (leading zeros allowed).
func GenerateOTP() (string, error) {
	var out [otpDigits]byte
	var b [1]byte
	for i := 0; i < otpDigits; i++ {
		for {
			if _, err := rand.Read(b[:]); err != nil {
				return "", err
			}
			if b[0] < otpUnbiasedMax {
				out[i] = '0' + (b[0] % otpDigitMod)
				break
			}
		}
	}
	return string(out[:]), nil
}

// NormalizeOTP requires exactly six digits after trim.
func NormalizeOTP(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if len(s) != otpDigits {
		return "", ErrInvalidBody("Code must be exactly 6 digits.")
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return "", ErrInvalidBody("Code must be exactly 6 digits.")
		}
	}
	return s, nil
}

func OTPExpired(expiresAt, now time.Time) bool {
	return !expiresAt.After(now)
}

func OTPSendCooldownActive(sentAt, now time.Time) bool {
	return now.Before(sentAt.Add(OTPSendCooldown))
}

// AdvanceSendWindow resets the hourly window when it has elapsed.
func AdvanceSendWindow(windowStart time.Time, sendCount int, now time.Time) (time.Time, int) {
	if now.Sub(windowStart) >= OTPSendWindow {
		return now, 0
	}
	return windowStart, sendCount
}

func OTPSendLimited(sendCount int) bool {
	return sendCount >= OTPSendsPerHour
}

func OTPGuessesSpent(attemptCount int) bool {
	return attemptCount >= OTPMaxAttempts
}
