package domain

import (
	"bytes"
	"strings"
	"unicode/utf8"
)

const (
	AvatarMaxBytes = 1 << 20

	AvatarJPEG = "image/jpeg"
	AvatarPNG  = "image/png"
	AvatarWebP = "image/webp"
)

// Profile is the display profile.
type Profile struct {
	DisplayName string
	Email       string
	// AvatarURL is the stored public path (/v1/avatars/{id}), or nil.
	AvatarURL *string
}

// Avatar is the stored photo. Not on the profile JSON.
type Avatar struct {
	ID          string
	ContentType string
	Bytes       []byte
}

// AvatarPath is the public path stored in profiles.avatar_url.
func AvatarPath(id string) string {
	return "/v1/avatars/" + id
}

// NormalizeRequiredDisplayName trims and requires 1–80 characters.
func NormalizeRequiredDisplayName(raw string) (string, error) {
	n := strings.TrimSpace(raw)
	if n == "" {
		return "", ErrInvalidBody("Display name is required.")
	}
	if utf8.RuneCountInString(n) > projectNameMax {
		return "", ErrInvalidBody("Display name must be at most 80 characters.")
	}
	return n, nil
}

// DetectAvatarContentType sniffs magic bytes. Size must be 1..AvatarMaxBytes.
func DetectAvatarContentType(b []byte) (string, error) {
	if len(b) == 0 {
		return "", ErrInvalidBody("Avatar file is required.")
	}
	if len(b) > AvatarMaxBytes {
		return "", ErrInvalidBody("Avatar must be at most 1 MiB.")
	}
	switch {
	case isJPEG(b):
		return AvatarJPEG, nil
	case isPNG(b):
		return AvatarPNG, nil
	case isWebP(b):
		return AvatarWebP, nil
	default:
		return "", ErrInvalidBody("Avatar must be a JPEG, PNG, or WebP image.")
	}
}

func isJPEG(b []byte) bool {
	return len(b) >= 3 && b[0] == 0xff && b[1] == 0xd8 && b[2] == 0xff
}

func isPNG(b []byte) bool {
	return bytes.HasPrefix(b, []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a})
}

func isWebP(b []byte) bool {
	return len(b) >= 12 && string(b[:4]) == "RIFF" && string(b[8:12]) == "WEBP"
}
