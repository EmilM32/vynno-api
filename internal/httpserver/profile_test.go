package httpserver

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var jpegMagic = []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 0x4a, 0x46, 0x49, 0x46}

func doMultipart(t *testing.T, r http.Handler, path, field, filename string, data []byte, opts ...reqOpt) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile(field, filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPut, path, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	for _, opt := range opts {
		opt(req)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestPatchMeDisplayName(t *testing.T) {
	r := testRouter(t)
	ck := loginCookie(t, r)
	auth := withCookie(ck)

	w := doJSON(t, r, http.MethodPatch, "/v1/me", map[string]any{"displayName": "  New Name  "}, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("PATCH /me = %d %s", w.Code, w.Body.String())
	}
	var p profileDTO
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.DisplayName != "New Name" {
		t.Fatalf("displayName = %q", p.DisplayName)
	}
	if p.Email != "alexdev@vynno.local" {
		t.Fatalf("email changed: %q", p.Email)
	}
	if p.AvatarURL != nil {
		t.Fatalf("avatarUrl should still be null")
	}

	w = doJSON(t, r, http.MethodPatch, "/v1/me", map[string]any{"displayName": ""}, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("PATCH empty displayName = %d %s", w.Code, w.Body.String())
	}
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.DisplayName != "" {
		t.Fatalf("cleared displayName = %q", p.DisplayName)
	}
	if p.Email != "alexdev@vynno.local" {
		t.Fatalf("email after clear = %q", p.Email)
	}

	w = doJSON(t, r, http.MethodPatch, "/v1/me", map[string]any{"displayName": nil}, auth)
	assertCode(t, w, http.StatusBadRequest, "invalid_body")

	w = doJSON(t, r, http.MethodPatch, "/v1/me", map[string]any{"handle": "@other"}, auth)
	assertCode(t, w, http.StatusBadRequest, "invalid_body")

	w = doJSON(t, r, http.MethodPatch, "/v1/me", map[string]any{"email": "other@example.com"}, auth)
	assertCode(t, w, http.StatusBadRequest, "invalid_body")
}

func TestAvatarUploadReplaceDeleteAndPublicGet(t *testing.T) {
	r := testRouter(t)
	ck := loginCookie(t, r)
	auth := withCookie(ck)

	w := doMultipart(t, r, "/v1/me/avatar", "file", "me.jpg", jpegMagic)
	assertCode(t, w, http.StatusUnauthorized, "unauthorized")

	w = doMultipart(t, r, "/v1/me/avatar", "file", "me.jpg", jpegMagic, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT avatar = %d %s", w.Code, w.Body.String())
	}
	var p profileDTO
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.AvatarURL == nil || !strings.HasPrefix(*p.AvatarURL, "http://localhost:8080/v1/avatars/") {
		t.Fatalf("avatarUrl = %v", p.AvatarURL)
	}
	firstURL := *p.AvatarURL

	img := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, strings.TrimPrefix(firstURL, "http://localhost:8080"), nil)
	r.ServeHTTP(img, req)
	if img.Code != http.StatusOK {
		t.Fatalf("GET avatar = %d %s", img.Code, img.Body.String())
	}
	if img.Header().Get("Content-Type") != "image/jpeg" {
		t.Fatalf("content-type = %q", img.Header().Get("Content-Type"))
	}
	got, _ := io.ReadAll(img.Body)
	if !bytes.Equal(got, jpegMagic) {
		t.Fatalf("bytes mismatch")
	}

	w = doMultipart(t, r, "/v1/me/avatar", "file", "me.jpg", jpegMagic, auth)
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.AvatarURL == nil || *p.AvatarURL == firstURL {
		t.Fatalf("replace should allocate a new id: %v", p.AvatarURL)
	}

	old := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, strings.TrimPrefix(firstURL, "http://localhost:8080"), nil)
	r.ServeHTTP(old, req)
	assertCode(t, old, http.StatusNotFound, "not_found")

	w = doMultipart(t, r, "/v1/me/avatar", "file", "x.svg", []byte("<svg></svg>"), auth)
	assertCode(t, w, http.StatusBadRequest, "invalid_body")

	w = doJSON(t, r, http.MethodDelete, "/v1/me/avatar", nil, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("DELETE avatar = %d %s", w.Code, w.Body.String())
	}
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.AvatarURL != nil {
		t.Fatalf("cleared avatarUrl = %v", p.AvatarURL)
	}

	w = doJSON(t, r, http.MethodDelete, "/v1/me/avatar", nil, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("idempotent DELETE = %d %s", w.Code, w.Body.String())
	}

	unknown := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/avatars/00000000-0000-4000-8000-000000000099", nil)
	r.ServeHTTP(unknown, req)
	assertCode(t, unknown, http.StatusNotFound, "not_found")
}
