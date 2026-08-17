package domain

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	projectNameMin = 1
	projectNameMax = 80
)

var (
	colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	codePattern  = regexp.MustCompile(`^[A-Z0-9-]{1,8}$`)
)

// Project is the server-side project entity (not the wire DTO).
type Project struct {
	ID              string
	Name            string
	Color           string
	Code            *string
	ProgressPercent *int
	Archived        bool
}

// NormalizeName trims and checks 1–80 characters.
func NormalizeName(name string) (string, error) {
	n := strings.TrimSpace(name)
	if utf8.RuneCountInString(n) < projectNameMin || utf8.RuneCountInString(n) > projectNameMax {
		return "", ErrInvalidBody("Name must be 1–80 characters after trim.")
	}
	return n, nil
}

// NormalizeColor accepts #rrggbb (any hex case) and stores lowercase.
func NormalizeColor(color string) (string, error) {
	c := strings.TrimSpace(color)
	if !colorPattern.MatchString(c) {
		return "", ErrInvalidBody("Color must be a #rrggbb hex value.")
	}
	return strings.ToLower(c), nil
}

// NormalizeCode trims, uppercases, and validates. Empty / whitespace means no code (nil).
func NormalizeCode(code *string) (*string, error) {
	if code == nil {
		return nil, nil
	}
	c := strings.ToUpper(strings.TrimSpace(*code))
	if c == "" {
		return nil, nil
	}
	if !codePattern.MatchString(c) {
		return nil, ErrInvalidBody("Code must match A–Z, 0–9, hyphen; 1–8 characters.")
	}
	return &c, nil
}

// CanArchive returns last_active_project when this is the last non-archived project,
// or invalid_transition when the project is already archived.
func CanArchive(p Project, activeCount int) error {
	if p.Archived {
		return ErrInvalidTransition()
	}
	if activeCount <= 1 {
		return ErrLastActiveProject()
	}
	return nil
}

// CanRestore is invalid when the project is not archived.
func CanRestore(p Project) error {
	if !p.Archived {
		return ErrInvalidTransition()
	}
	return nil
}

// CanHardDelete requires zero sessions and that this is not the last active project.
func CanHardDelete(p Project, activeCount, sessionCount int) error {
	if sessionCount > 0 {
		return ErrProjectHasSessions()
	}
	if !p.Archived && activeCount <= 1 {
		return ErrLastActiveProject()
	}
	return nil
}

// CanStartSession checks the project exists (caller) and is not archived.
func CanStartSession(p Project) error {
	if p.Archived {
		return ErrProjectArchived()
	}
	return nil
}
