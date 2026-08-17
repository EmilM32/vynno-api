package domain

// Profile is the display profile. Read-only on the wire in v1.
type Profile struct {
	DisplayName string
	Handle      string
	AvatarURL   *string
}
