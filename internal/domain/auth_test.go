package domain

import (
	"strings"
	"testing"
)

func TestNormalizeEmail(t *testing.T) {
	t.Parallel()
	got, err := NormalizeEmail("  Alex@Example.COM  ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "alex@example.com" {
		t.Fatalf("got %q", got)
	}

	if _, err := NormalizeEmail("ab"); err == nil {
		t.Fatal("expected too short")
	}
	if _, err := NormalizeEmail("not-an-email"); err == nil {
		t.Fatal("expected missing @")
	}
	if _, err := NormalizeEmail("Name <alex@example.com>"); err == nil {
		t.Fatal("expected display-name form rejected")
	}
	if _, err := NormalizeEmail("user@localhost"); err == nil {
		t.Fatal("expected domain without dot rejected")
	}
	if _, err := NormalizeEmail("alexdev@vynno.local"); err != nil {
		t.Fatalf("seed email: %v", err)
	}
	long := strings.Repeat("a", 251) + "@b.c"
	if _, err := NormalizeEmail(long); err == nil {
		t.Fatal("expected too long")
	}
}

func TestNormalizeDisplayNamePipeline(t *testing.T) {
	t.Parallel()
	got, err := NormalizeDisplayName("  Alex  ")
	if err != nil || got != "Alex" {
		t.Fatalf("%q %v", got, err)
	}
	got, err = NormalizeDisplayName(" \u200b ")
	if err != nil || got != "" {
		t.Fatalf("empty display %q %v", got, err)
	}
	if _, err := NormalizeDisplayName(strings.Repeat("a", 81)); err == nil {
		t.Fatal("expected 81 rejected")
	}
	if _, err := NormalizeDisplayName("a\u202eb"); err == nil {
		t.Fatal("expected bidi rejected")
	}
}

func TestNormalizePassword(t *testing.T) {
	t.Parallel()
	if _, err := NormalizePassword("short"); err == nil {
		t.Fatal("expected too short")
	}
	got, err := NormalizePassword("long-enough")
	if err != nil {
		t.Fatal(err)
	}
	if got != "long-enough" {
		t.Fatalf("got %q", got)
	}
}

func TestRememberMe(t *testing.T) {
	t.Parallel()
	if !RememberMe(nil) {
		t.Fatal("omitted should default true")
	}
	f := false
	if RememberMe(&f) {
		t.Fatal("false should stay false")
	}
}
