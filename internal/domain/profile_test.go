package domain

import (
	"bytes"
	"strings"
	"testing"
)

func TestNormalizeRequiredDisplayName(t *testing.T) {
	t.Parallel()
	got, err := NormalizeRequiredDisplayName("  Alex  ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "Alex" {
		t.Fatalf("got %q", got)
	}
	if _, err := NormalizeRequiredDisplayName("   "); err == nil {
		t.Fatal("expected empty rejected")
	}
	if _, err := NormalizeRequiredDisplayName(strings.Repeat("a", 81)); err == nil {
		t.Fatal("expected too long")
	}
}

func TestDetectAvatarContentType(t *testing.T) {
	t.Parallel()
	jpeg := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00}
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00}
	webp := []byte("RIFF\x00\x00\x00\x00WEBP....")

	ct, err := DetectAvatarContentType(jpeg)
	if err != nil || ct != AvatarJPEG {
		t.Fatalf("jpeg: %q %v", ct, err)
	}
	ct, err = DetectAvatarContentType(png)
	if err != nil || ct != AvatarPNG {
		t.Fatalf("png: %q %v", ct, err)
	}
	ct, err = DetectAvatarContentType(webp)
	if err != nil || ct != AvatarWebP {
		t.Fatalf("webp: %q %v", ct, err)
	}

	if _, err := DetectAvatarContentType(nil); err == nil {
		t.Fatal("expected empty rejected")
	}
	if _, err := DetectAvatarContentType([]byte("<svg></svg>")); err == nil {
		t.Fatal("expected svg rejected")
	}
	if _, err := DetectAvatarContentType(bytes.Repeat([]byte{0xff, 0xd8, 0xff}, AvatarMaxBytes)); err == nil {
		t.Fatal("expected oversize rejected")
	}
}

func TestAvatarPath(t *testing.T) {
	t.Parallel()
	if got := AvatarPath("abc"); got != "/v1/avatars/abc" {
		t.Fatalf("got %q", got)
	}
}
