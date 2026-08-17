package domain

import "testing"

func TestNormalizeName(t *testing.T) {
	t.Parallel()
	got, err := NormalizeName("  Identity  ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "Identity" {
		t.Fatalf("got %q", got)
	}
	if _, err := NormalizeName("   "); err == nil {
		t.Fatal("expected invalid empty name")
	}
}

func TestNormalizeColor(t *testing.T) {
	t.Parallel()
	got, err := NormalizeColor("#3B82F6")
	if err != nil {
		t.Fatal(err)
	}
	if got != "#3b82f6" {
		t.Fatalf("got %q", got)
	}
	if _, err := NormalizeColor("blue"); err == nil {
		t.Fatal("expected invalid color")
	}
}

func TestNormalizeCode(t *testing.T) {
	t.Parallel()
	in := " auth "
	got, err := NormalizeCode(&in)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || *got != "AUTH" {
		t.Fatalf("got %#v", got)
	}
	empty := "  "
	got, err = NormalizeCode(&empty)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("empty code should clear, got %#v", got)
	}
	bad := "too-long-code"
	if _, err := NormalizeCode(&bad); err == nil {
		t.Fatal("expected invalid code")
	}
}

func TestCanArchiveLastActive(t *testing.T) {
	t.Parallel()
	p := Project{Archived: false}
	if err := CanArchive(p, 1); err == nil {
		t.Fatal("expected last_active_project")
	}
	if err := CanArchive(p, 2); err != nil {
		t.Fatal(err)
	}
	p.Archived = true
	if err := CanArchive(p, 2); err == nil {
		t.Fatal("expected invalid_transition")
	}
}

func TestCanHardDelete(t *testing.T) {
	t.Parallel()
	p := Project{Archived: false}
	if err := CanHardDelete(p, 1, 0); err == nil {
		t.Fatal("expected last_active_project")
	}
	if err := CanHardDelete(p, 2, 3); err == nil {
		t.Fatal("expected project_has_sessions")
	}
	if err := CanHardDelete(p, 2, 0); err != nil {
		t.Fatal(err)
	}
}
