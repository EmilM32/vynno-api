package domain

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	usernameMin = 3
	usernameMax = 32
	passwordMin = 8
	passwordMax = 128
)

var usernamePattern = regexp.MustCompile(`^[a-z0-9_]{3,32}$`)

// NormalizeUsername trims, lowercases, and checks ^[a-z0-9_]{3,32}$.
func NormalizeUsername(raw string) (string, error) {
	u := strings.ToLower(strings.TrimSpace(raw))
	if !usernamePattern.MatchString(u) {
		return "", ErrInvalidBody("Username must be 3–32 characters: lowercase letters, digits, or underscore.")
	}
	return u, nil
}

// NormalizePassword checks length only. The caller hashes the result.
func NormalizePassword(raw string) (string, error) {
	n := utf8.RuneCountInString(raw)
	if n < passwordMin || n > passwordMax {
		return "", ErrInvalidBody("Password must be 8–128 characters.")
	}
	return raw, nil
}

// NormalizeDisplayName trims a register display name. Empty becomes "".
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

// HandleFromUsername is the default profile handle.
func HandleFromUsername(username string) string {
	return "@" + username
}

// RememberMe defaults to true when the field is omitted.
func RememberMe(v *bool) bool {
	if v == nil {
		return true
	}
	return *v
}
