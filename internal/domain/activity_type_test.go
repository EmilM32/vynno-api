package domain

import "testing"

func TestNormalizeActivityTypeName(t *testing.T) {
	t.Parallel()
	got, err := NormalizeActivityTypeName("  Deep Work  ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "Deep Work" {
		t.Fatalf("got %q", got)
	}
	if _, err := NormalizeActivityTypeName(""); err == nil {
		t.Fatal("expected invalid empty name")
	}
	got, err = NormalizeActivityTypeName("Coding!")
	if err != nil {
		t.Fatal(err)
	}
	if got != "Coding!" {
		t.Fatalf("punctuation should be kept, got %q", got)
	}
}

func TestNormalizeActivityColor(t *testing.T) {
	t.Parallel()
	got, err := NormalizeActivityColor(" Primary ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "primary" {
		t.Fatalf("got %q", got)
	}
	if _, err := NormalizeActivityColor("#3b82f6"); err == nil {
		t.Fatal("expected hex rejected")
	}
	if _, err := NormalizeActivityColor("blue"); err == nil {
		t.Fatal("expected unknown token rejected")
	}
}

func TestCanDeleteActivityType(t *testing.T) {
	t.Parallel()
	if err := CanDeleteActivityType(0); err != nil {
		t.Fatal(err)
	}
	if err := CanDeleteActivityType(1); err == nil {
		t.Fatal("expected activity_type_has_sessions")
	}
}
