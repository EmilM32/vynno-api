package domain

import (
	"encoding/json"
	"os"
	"testing"
)

// textVector is one row of testdata/text_vectors.json. The file is canonical
// here; scripts/sync-contract copies it to the frontend, whose normalize.test.ts
// runs the same rows. normalized is the text pipeline result: an empty note is
// "" (the API then stores UntitledNote) and an empty ticket is "" (nil here).
// reason is the frontend's reject reason; the API only reports error.
type textVector struct {
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	Input      string `json:"input"`
	Normalized string `json:"normalized"`
	Error      string `json:"error"`
	Reason     string `json:"reason"`
}

func TestTextVectors(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("testdata/text_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors []textVector
	if err := json.Unmarshal(raw, &vectors); err != nil {
		t.Fatal(err)
	}
	if len(vectors) == 0 {
		t.Fatal("no vectors")
	}
	for _, v := range vectors {
		t.Run(v.Name, func(t *testing.T) {
			t.Parallel()
			var got string
			var err error
			switch v.Kind {
			case "name":
				got, err = NormalizeName(v.Input)
			case "note":
				got, err = NormalizeNote(v.Input)
				if err == nil && got == UntitledNote {
					got = ""
				}
			case "ticket":
				in := v.Input
				var p *string
				p, err = NormalizeTicketID(&in)
				if p != nil {
					got = *p
				}
			default:
				t.Fatalf("unknown kind %q", v.Kind)
			}
			if v.Error != "" {
				assertDomainCode(t, err, v.Error)
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != v.Normalized {
				t.Fatalf("got %q, want %q", got, v.Normalized)
			}
		})
	}
}

func assertDomainCode(t *testing.T, err error, code string) {
	t.Helper()
	de, ok := AsError(err)
	if !ok || de.Code != code {
		t.Fatalf("got %v, want %s", err, code)
	}
}
