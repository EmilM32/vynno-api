package domain

import (
	"slices"
	"strings"
)

// ActivityColorTokens are the closed set stored on activity types.
// Chip and chart CSS live in the SPA.
var ActivityColorTokens = []string{
	"primary",
	"secondary",
	"tertiary",
	"error",
	"on-surface-variant",
	"outline",
	"primary-container",
	"secondary-container",
}

// ActivityType is a user-owned dictionary row (not the wire DTO).
type ActivityType struct {
	ID    string
	Name  string
	Color string
}

// NormalizeActivityTypeName trims and checks 1–80 characters. Case is preserved.
func NormalizeActivityTypeName(name string) (string, error) {
	return NormalizeName(name)
}

// NormalizeActivityColor accepts a known theme token.
func NormalizeActivityColor(color string) (string, error) {
	c := strings.ToLower(strings.TrimSpace(color))
	if !slices.Contains(ActivityColorTokens, c) {
		return "", ErrInvalidBody("Color must be a known token.")
	}
	return c, nil
}

// CanDeleteActivityType requires zero sessions referencing the row.
func CanDeleteActivityType(sessionCount int) error {
	if sessionCount > 0 {
		return ErrActivityTypeHasSessions()
	}
	return nil
}
