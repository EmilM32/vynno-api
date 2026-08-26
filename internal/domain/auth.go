package domain

import (
	"net/mail"
	"strings"
	"unicode/utf8"
)

const (
	emailMin    = 3
	emailMax    = 254
	passwordMin = 8
	passwordMax = 128
)

// NormalizeEmail trims, lowercases, and checks a single address whose domain contains a dot.
func NormalizeEmail(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	n := utf8.RuneCountInString(s)
	if n < emailMin || n > emailMax {
		return "", ErrInvalidBody("Email must be 3–254 characters.")
	}
	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Address != s {
		return "", ErrInvalidBody("Email is not valid.")
	}
	at := strings.LastIndex(s, "@")
	if at <= 0 || at == len(s)-1 {
		return "", ErrInvalidBody("Email is not valid.")
	}
	if !strings.Contains(s[at+1:], ".") {
		return "", ErrInvalidBody("Email is not valid.")
	}
	return s, nil
}

// NormalizePassword checks length only. The caller hashes the result.
func NormalizePassword(raw string) (string, error) {
	n := utf8.RuneCountInString(raw)
	if n < passwordMin || n > passwordMax {
		return "", ErrInvalidBody("Password must be 8–128 characters.")
	}
	return raw, nil
}

// NormalizeDisplayName trims a display name. Empty becomes "".
func NormalizeDisplayName(raw string) (string, error) {
	n := strings.TrimSpace(raw)
	if n == "" {
		return "", nil
	}
	if utf8.RuneCountInString(n) > projectNameMax {
		return "", ErrInvalidBody("Display name must be at most 80 characters.")
	}
	return n, nil
}

// RememberMe defaults to true when the field is omitted.
func RememberMe(v *bool) bool {
	if v == nil {
		return true
	}
	return *v
}
