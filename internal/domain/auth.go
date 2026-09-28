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

// RecheckEmails normalizes stored addresses. When two distinct originals
// normalize to one value, the pair is reported and neither is rewritten.
// A changed value with no collision is returned as old → new. Nothing is merged.
func RecheckEmails(existing []string) (updates map[string]string, collisions [][2]string) {
	updates = map[string]string{}
	grouped := map[string][]string{}
	seen := map[string]map[string]struct{}{}
	var order []string
	for _, raw := range existing {
		normed, err := NormalizeEmail(raw)
		if err != nil {
			continue
		}
		if _, ok := grouped[normed]; !ok {
			order = append(order, normed)
			seen[normed] = map[string]struct{}{}
		}
		if _, dup := seen[normed][raw]; dup {
			continue
		}
		seen[normed][raw] = struct{}{}
		grouped[normed] = append(grouped[normed], raw)
	}
	for _, normed := range order {
		raws := grouped[normed]
		if len(raws) > 1 {
			for i := 0; i < len(raws); i++ {
				for j := i + 1; j < len(raws); j++ {
					collisions = append(collisions, [2]string{raws[i], raws[j]})
				}
			}
			continue
		}
		if raws[0] != normed {
			updates[raws[0]] = normed
		}
	}
	return updates, collisions
}

// NormalizePassword checks length only. The caller hashes the result.
func NormalizePassword(raw string) (string, error) {
	n := utf8.RuneCountInString(raw)
	if n < passwordMin || n > passwordMax {
		return "", ErrInvalidBody("Password must be 8–128 characters.")
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
