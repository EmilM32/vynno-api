package domain

import (
	"net/mail"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/idna"
	"golang.org/x/text/unicode/norm"
)

const (
	emailMin    = 3
	emailMax    = 254
	passwordMin = 8
	passwordMax = 128
	// passwordMaxBytes is bcrypt's input limit. Longer input is rejected by
	// bcrypt.GenerateFromPassword, so the domain refuses it first.
	passwordMaxBytes = 72
)

// NormalizeEmail trims, rejects Cc/Cf, applies NFC, lowercases, and stores the
// domain as IDNA punycode. The local part is at most 64 octets. Non-ASCII
// local parts are kept. The result is one address whose domain contains a dot.
func NormalizeEmail(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" || containsCategory(s, unicode.Cc) || containsCategory(s, unicode.Cf) {
		return "", ErrInvalidBody("Email is not valid.")
	}
	s = strings.ToLower(norm.NFC.String(s))
	at := strings.LastIndex(s, "@")
	if at <= 0 || at == len(s)-1 {
		return "", ErrInvalidBody("Email is not valid.")
	}
	local, domain := s[:at], s[at+1:]
	if len(local) > 64 {
		return "", ErrInvalidBody("Email is not valid.")
	}
	asciiDomain, err := idna.Lookup.ToASCII(domain)
	if err != nil || asciiDomain == "" {
		return "", ErrInvalidBody("Email is not valid.")
	}
	s = local + "@" + asciiDomain
	n := utf8.RuneCountInString(s)
	if n < emailMin || n > emailMax {
		return "", ErrInvalidBody("Email must be 3–254 characters.")
	}
	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Address != s {
		return "", ErrInvalidBody("Email is not valid.")
	}
	if !strings.Contains(asciiDomain, ".") {
		return "", ErrInvalidBody("Email is not valid.")
	}
	return s, nil
}

func containsCategory(s string, cat *unicode.RangeTable) bool {
	for _, r := range s {
		if unicode.Is(cat, r) {
			return true
		}
	}
	return false
}

// NormalizePassword checks length only: 8–128 characters and at most 72 bytes
// of UTF-8 (bcrypt's limit). The caller hashes the result.
func NormalizePassword(raw string) (string, error) {
	n := utf8.RuneCountInString(raw)
	if n < passwordMin || n > passwordMax {
		return "", ErrInvalidBody("Password must be 8–128 characters.")
	}
	if len(raw) > passwordMaxBytes {
		return "", ErrInvalidBody("Password must be at most 72 bytes.")
	}
	return raw, nil
}

// NormalizeDisplayName applies the shared text pipeline. Empty is allowed. Max 80 code points.
func NormalizeDisplayName(raw string) (string, error) {
	n, err := normalizeLabel(raw)
	if err != nil {
		return "", err
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
