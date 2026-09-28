package domain

import (
	"strings"
	"testing"

	"golang.org/x/text/unicode/norm"
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

func TestNormalizeEmailIDNAndLocalPart(t *testing.T) {
	t.Parallel()
	nfc, err := NormalizeEmail("żółć@example.com")
	if err != nil {
		t.Fatal(err)
	}
	nfd, err := NormalizeEmail(norm.NFD.String("żółć@example.com"))
	if err != nil {
		t.Fatal(err)
	}
	if nfc != nfd || nfc != "żółć@example.com" {
		t.Fatalf("nfc %q nfd %q", nfc, nfd)
	}

	idn, err := NormalizeEmail("user@exämple.pl")
	if err != nil {
		t.Fatal(err)
	}
	puny, err := NormalizeEmail("user@xn--exmple-cua.pl")
	if err != nil {
		t.Fatal(err)
	}
	if idn != puny || idn != "user@xn--exmple-cua.pl" {
		t.Fatalf("idn %q puny %q", idn, puny)
	}

	local64 := strings.Repeat("a", 64) + "@example.com"
	got, err := NormalizeEmail(local64)
	if err != nil || got != local64 {
		t.Fatalf("64: %q %v", got, err)
	}
	if _, err := NormalizeEmail(strings.Repeat("a", 65) + "@example.com"); err == nil {
		t.Fatal("expected 65 octet local rejected")
	}
	for _, raw := range []string{"us\u200ber@example.com", "user@example.com\u202e", "\ufeffuser@example.com"} {
		if _, err := NormalizeEmail(raw); err == nil {
			t.Fatalf("expected Cf rejected: %q", raw)
		}
	}
	homo, err := NormalizeEmail("\u0430lex@example.com")
	if err != nil || homo != "\u0430lex@example.com" {
		t.Fatalf("homoglyph local: %q %v", homo, err)
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

func TestRecheckEmails(t *testing.T) {
	t.Parallel()
	updates, collisions := RecheckEmails([]string{
		"User@exämple.pl",
		"user@xn--exmple-cua.pl",
	})
	if len(updates) != 0 {
		t.Fatalf("picked a winner: %#v", updates)
	}
	if len(collisions) != 1 || collisions[0] != [2]string{"User@exämple.pl", "user@xn--exmple-cua.pl"} {
		t.Fatalf("collisions = %#v", collisions)
	}

	updates, collisions = RecheckEmails([]string{"User@Example.COM"})
	if len(collisions) != 0 {
		t.Fatalf("collisions = %#v", collisions)
	}
	if updates["User@Example.COM"] != "user@example.com" {
		t.Fatalf("updates = %#v", updates)
	}

	nfc := "żółć@example.com"
	nfd := norm.NFD.String(nfc)
	updates, collisions = RecheckEmails([]string{nfc, nfd})
	if len(updates) != 0 || len(collisions) != 1 {
		t.Fatalf("updates %#v collisions %#v", updates, collisions)
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
