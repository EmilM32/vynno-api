package domain

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

const (
	noteMax   = 500
	ticketMax = 64
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

// NormalizeNote applies NFC, rejects disallowed controls, and trims.
// Empty after trim is "Untitled session". Length is Unicode code points.
func NormalizeNote(note string) (string, error) {
	s := norm.NFC.String(note)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if isBidiControl(r) {
			return "", ErrInvalidBody("Note contains invalid characters.")
		}
		if unicode.Is(unicode.Cc, r) && r != '\t' && r != '\n' && r != '\r' {
			return "", ErrInvalidBody("Note contains invalid characters.")
		}
		b.WriteRune(r)
	}
	s = strings.TrimFunc(b.String(), unicode.IsSpace)
	if s == "" {
		return UntitledNote, nil
	}
	if utf8.RuneCountInString(s) > noteMax {
		return "", ErrInvalidBody("Note must be at most 500 characters.")
	}
	return s, nil
}

// NormalizeTicketID applies the name pipeline. Empty after normalize is nil.
// Length is Unicode code points, max 64.
func NormalizeTicketID(raw *string) (*string, error) {
	if raw == nil {
		return nil, nil
	}
	s, err := normalizeLabel(*raw)
	if err != nil {
		return nil, err
	}
	if s == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(s) > ticketMax {
		return nil, ErrInvalidBody("ticketId must be at most 64 characters.")
	}
	return &s, nil
}
