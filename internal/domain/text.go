package domain

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// normalizeLabel applies the shared name pipeline: reject U+FFFD, NFC,
// reject Cc and bidi controls, strip zero-width characters, trim spaces.
// Length checks stay with the caller.
func normalizeLabel(raw string) (string, error) {
	if strings.ContainsRune(raw, '\uFFFD') {
		return "", ErrInvalidBody("Text contains invalid characters.")
	}
	s := norm.NFC.String(raw)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.Is(unicode.Cc, r) || isBidiControl(r) {
			return "", ErrInvalidBody("Text contains invalid characters.")
		}
		if isDroppedFormat(r) {
			continue
		}
		b.WriteRune(r)
	}
	return strings.TrimFunc(b.String(), unicode.IsSpace), nil
}

func isBidiControl(r rune) bool {
	return (r >= 0x202A && r <= 0x202E) || (r >= 0x2066 && r <= 0x2069)
}

func isDroppedFormat(r rune) bool {
	switch r {
	case 0x200B, 0x200C, 0x200D, 0x2060, 0xFEFF:
		return true
	default:
		return false
	}
}
