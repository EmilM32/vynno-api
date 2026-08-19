package devdata

import "testing"

func TestDatabaseName(t *testing.T) {
	got, err := DatabaseName("postgres://vynno:vynno@localhost:5433/vynno_dev?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	if got != DevDatabase {
		t.Fatalf("got %q", got)
	}
}

func TestRequireDevDatabase(t *testing.T) {
	if err := RequireDevDatabase("postgres://vynno:vynno@localhost:5433/vynno_dev?sslmode=disable"); err != nil {
		t.Fatal(err)
	}
	err := RequireDevDatabase("postgres://vynno:vynno@localhost:5433/vynno?sslmode=disable")
	if err == nil {
		t.Fatal("expected refusal of production database")
	}
	if err := RequireDevDatabase("postgres://vynno:vynno@localhost:5433/"); err == nil {
		t.Fatal("expected error for missing name")
	}
}
