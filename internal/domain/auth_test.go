package domain

import "testing"

func TestNormalizeUsername(t *testing.T) {
	t.Parallel()
	got, err := NormalizeUsername("  Alex_Dev  ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "alex_dev" {
		t.Fatalf("got %q", got)
	}
	if _, err := NormalizeUsername("ab"); err == nil {
		t.Fatal("expected too short")
	}
	if _, err := NormalizeUsername("Alex-Dev"); err == nil {
		t.Fatal("expected hyphen rejected")
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
