package httpserver

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestProjectProgressPercent(t *testing.T) {
	r := testRouter(t)
	ck := loginCookie(t, r)
	auth := withCookie(ck)

	w := doJSON(t, r, http.MethodPost, "/v1/projects", map[string]any{
		"name": "Progress", "color": "#22c55e", "code": "PRG", "progressPercent": 60,
	}, auth)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /projects = %d %s", w.Code, w.Body.String())
	}
	var created projectDTO
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ProgressPercent == nil || *created.ProgressPercent != 60 {
		t.Fatalf("create progressPercent = %#v", created.ProgressPercent)
	}

	w = doJSON(t, r, http.MethodPatch, "/v1/projects/"+created.ID, map[string]any{
		"progressPercent": 80,
	}, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("PATCH 80 = %d %s", w.Code, w.Body.String())
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ProgressPercent == nil || *created.ProgressPercent != 80 {
		t.Fatalf("patch 80 = %#v", created.ProgressPercent)
	}

	w = doJSON(t, r, http.MethodPatch, "/v1/projects/"+created.ID, map[string]any{
		"name": "Progress renamed",
	}, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("PATCH name = %d %s", w.Code, w.Body.String())
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ProgressPercent == nil || *created.ProgressPercent != 80 {
		t.Fatalf("omit should leave 80, got %#v", created.ProgressPercent)
	}

	w = doJSON(t, r, http.MethodPatch, "/v1/projects/"+created.ID, map[string]any{
		"progressPercent": nil,
	}, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("PATCH null = %d %s", w.Code, w.Body.String())
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ProgressPercent != nil {
		t.Fatalf("cleared progressPercent = %#v", created.ProgressPercent)
	}

	w = doJSON(t, r, http.MethodPost, "/v1/projects", map[string]any{
		"name": "Over", "color": "#000000", "progressPercent": 101,
	}, auth)
	assertCode(t, w, http.StatusBadRequest, "invalid_body")

	w = doJSON(t, r, http.MethodPatch, "/v1/projects/"+created.ID, map[string]any{
		"progressPercent": -1,
	}, auth)
	assertCode(t, w, http.StatusBadRequest, "invalid_body")
}
