package domain

import (
	"encoding/json"
	"os"
	"testing"
)

type textVector struct {
	Input      string  `json:"input"`
	Kind       string  `json:"kind"`
	Normalized *string `json:"normalized"`
	Error      string  `json:"error"`
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
		v := v
		t.Run(v.Kind+"/"+v.Error, func(t *testing.T) {
			t.Parallel()
			switch v.Kind {
			case "name":
				got, err := NormalizeName(v.Input)
				assertText(t, v, got, err)
			default:
				t.Fatalf("unknown kind %q", v.Kind)
			}
		})
	}
}

func assertText(t *testing.T, v textVector, got string, err error) {
	t.Helper()
	if v.Error != "" {
		assertDomainCode(t, err, v.Error)
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if v.Normalized == nil || got != *v.Normalized {
		want := "<nil>"
		if v.Normalized != nil {
			want = *v.Normalized
		}
		t.Fatalf("got %q, want %q", got, want)
	}
}

func assertDomainCode(t *testing.T, err error, code string) {
	t.Helper()
	de, ok := AsError(err)
	if !ok || de.Code != code {
		t.Fatalf("got %v, want %s", err, code)
	}
}
