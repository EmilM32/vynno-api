package store

import (
	"testing"
	"time"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/google/uuid"
)

func TestSessionCursorRoundTrip(t *testing.T) {
	id := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	started := time.Date(2026, 3, 11, 8, 0, 0, 123456789, time.UTC)
	raw := EncodeSessionCursor(started, id)
	gotT, gotID, err := DecodeSessionCursor(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !gotT.Equal(started) {
		t.Fatalf("startedAt = %s, want %s", gotT, started)
	}
	if gotID != id {
		t.Fatalf("id = %s, want %s", gotID, id)
	}
}

func TestDecodeSessionCursorRejectsGarbage(t *testing.T) {
	for _, raw := range []string{"", "%%%", "bm90LWEtdXVpZA", "YQ"} {
		if _, _, err := DecodeSessionCursor(raw); err == nil {
			t.Fatalf("DecodeSessionCursor(%q) succeeded", raw)
		}
	}
}

func TestPaginateSessionsKeyset(t *testing.T) {
	t0 := time.Date(2026, 3, 11, 10, 0, 0, 0, time.UTC)
	idHi := uuid.MustParse("ffffffff-0000-4000-8000-000000000001")
	idLo := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	idMid := uuid.MustParse("88888888-0000-4000-8000-000000000001")
	rows := []domain.Session{
		{ID: idHi.String(), StartedAt: t0, Note: "same-hi"},
		{ID: idMid.String(), StartedAt: t0, Note: "same-mid"},
		{ID: idLo.String(), StartedAt: t0, Note: "same-lo"},
		{ID: uuid.MustParse("11111111-0000-4000-8000-000000000002").String(), StartedAt: t0.Add(-time.Hour), Note: "older"},
	}

	page1, err := paginateSessions(rows, 2, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(page1.Items) != 2 || page1.NextCursor == nil {
		t.Fatalf("page1 = %#v", page1)
	}
	if page1.Items[0].Note != "same-hi" || page1.Items[1].Note != "same-mid" {
		t.Fatalf("page1 notes = %s, %s", page1.Items[0].Note, page1.Items[1].Note)
	}

	page2, err := paginateSessions(rows, 2, *page1.NextCursor)
	if err != nil {
		t.Fatal(err)
	}
	if len(page2.Items) != 2 || page2.NextCursor != nil {
		t.Fatalf("page2 = %#v", page2)
	}
	if page2.Items[0].Note != "same-lo" || page2.Items[1].Note != "older" {
		t.Fatalf("page2 notes = %s, %s", page2.Items[0].Note, page2.Items[1].Note)
	}
}
