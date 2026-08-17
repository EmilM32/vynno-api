package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EmilM32/vynno-api/internal/service"
	"github.com/EmilM32/vynno-api/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const testPassword = "test-pass-1"

func testOpts() Options {
	return Options{
		SPAOrigins:      []string{"http://localhost:5173"},
		PublicAPIOrigin: "http://localhost:8080",
	}
}

func testRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	user := store.DefaultUserID()
	mem := store.NewMemory(user, store.DefaultProfile(), store.DefaultProject())
	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := mem.Bootstrap(context.Background(), user, "alexdev", string(hash), store.DefaultProfile(), store.DefaultProject()); err != nil {
		t.Fatal(err)
	}
	return NewRouter(service.New(mem), testOpts())
}

func doJSON(t *testing.T, r http.Handler, method, path string, body any, opts ...reqOpt) *httptest.ResponseRecorder {
	t.Helper()
	var buf *bytes.Buffer
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		buf = bytes.NewBuffer(b)
	} else {
		buf = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, opt := range opts {
		opt(req)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

type reqOpt func(*http.Request)

func withCookie(c *http.Cookie) reqOpt {
	return func(r *http.Request) {
		if c != nil {
			r.AddCookie(c)
		}
	}
}

func withBearer(token string) reqOpt {
	return func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+token)
	}
}

func withOrigin(origin string) reqOpt {
	return func(r *http.Request) {
		r.Header.Set("Origin", origin)
	}
}

func withReferer(ref string) reqOpt {
	return func(r *http.Request) {
		r.Header.Set("Referer", ref)
	}
}

func loginCookie(t *testing.T, r http.Handler) *http.Cookie {
	t.Helper()
	return loginAs(t, r, "alexdev", testPassword)
}

func loginAs(t *testing.T, r http.Handler, username, password string) *http.Cookie {
	t.Helper()
	w := doJSON(t, r, http.MethodPost, "/v1/auth/login", map[string]any{
		"username": username, "password": password,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("login = %d %s", w.Code, w.Body.String())
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookie {
			return c
		}
	}
	t.Fatal("missing session cookie")
	return nil
}

func TestUnauthenticatedRejected(t *testing.T) {
	r := testRouter(t)
	w := doJSON(t, r, http.MethodGet, "/v1/me", nil)
	assertCode(t, w, http.StatusUnauthorized, "unauthorized")
	w = doJSON(t, r, http.MethodPatch, "/v1/me", map[string]any{"displayName": "X"})
	assertCode(t, w, http.StatusUnauthorized, "unauthorized")
	w = doJSON(t, r, http.MethodDelete, "/v1/me/avatar", nil)
	assertCode(t, w, http.StatusUnauthorized, "unauthorized")
	w = doJSON(t, r, http.MethodPost, "/v1/sessions", map[string]any{"projectId": uuid.NewString()})
	assertCode(t, w, http.StatusUnauthorized, "unauthorized")
}

func TestMeAndProjectsAndSessions(t *testing.T) {
	r := testRouter(t)
	ck := loginCookie(t, r)
	auth := withCookie(ck)

	w := doJSON(t, r, http.MethodGet, "/v1/me", nil, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /me = %d %s", w.Code, w.Body.String())
	}
	var profile profileDTO
	if err := json.Unmarshal(w.Body.Bytes(), &profile); err != nil {
		t.Fatal(err)
	}
	if profile.DisplayName == "" || profile.Handle == "" {
		t.Fatalf("profile: %+v", profile)
	}
	if profile.AvatarURL != nil {
		t.Fatalf("avatarUrl should be null")
	}

	w = doJSON(t, r, http.MethodGet, "/v1/projects", nil, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /projects = %d %s", w.Code, w.Body.String())
	}
	var list listDTO[projectDTO]
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("projects = %#v", list.Items)
	}
	seedID := list.Items[0].ID

	w = doJSON(t, r, http.MethodPost, "/v1/projects", map[string]any{
		"name": "Tools", "color": "#22c55e", "code": "tool",
	}, auth)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /projects = %d %s", w.Code, w.Body.String())
	}
	var created projectDTO
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Code == nil || *created.Code != "TOOL" {
		t.Fatalf("code not normalized: %#v", created.Code)
	}

	w = doJSON(t, r, http.MethodPost, "/v1/projects", map[string]any{
		"name": "Dup", "color": "#000000", "code": "TOOL",
	}, auth)
	assertCode(t, w, http.StatusConflict, "code_in_use")

	w = doJSON(t, r, http.MethodGet, "/v1/sessions/active", nil, auth)
	assertCode(t, w, http.StatusNotFound, "session_not_active")

	w = doJSON(t, r, http.MethodPost, "/v1/sessions", map[string]any{
		"projectId": seedID, "note": "  ",
	}, auth)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /sessions = %d %s", w.Code, w.Body.String())
	}
	var live sessionDTO
	if err := json.Unmarshal(w.Body.Bytes(), &live); err != nil {
		t.Fatal(err)
	}
	if live.Note != "Untitled session" || live.Status != "active" || live.Tags == nil {
		t.Fatalf("session: %+v", live)
	}

	w = doJSON(t, r, http.MethodPost, "/v1/sessions", map[string]any{
		"projectId": seedID, "note": "second",
	}, auth)
	assertCode(t, w, http.StatusConflict, "session_already_active")

	w = doJSON(t, r, http.MethodPost, "/v1/sessions/"+live.ID+"/pause", nil, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("pause = %d %s", w.Code, w.Body.String())
	}
	w = doJSON(t, r, http.MethodPost, "/v1/sessions/"+live.ID+"/pause", nil, auth)
	assertCode(t, w, http.StatusConflict, "invalid_transition")

	w = doJSON(t, r, http.MethodPost, "/v1/sessions/"+live.ID+"/resume", nil, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("resume = %d %s", w.Code, w.Body.String())
	}
	w = doJSON(t, r, http.MethodPost, "/v1/sessions/"+live.ID+"/stop", nil, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("stop = %d %s", w.Code, w.Body.String())
	}

	w = doJSON(t, r, http.MethodPost, "/v1/projects/"+seedID+"/archive", nil, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("archive seed = %d %s", w.Code, w.Body.String())
	}
	w = doJSON(t, r, http.MethodGet, "/v1/projects", nil, auth)
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 || list.Items[0].ID != created.ID {
		t.Fatalf("default list should omit archived: %#v", list.Items)
	}

	w = doJSON(t, r, http.MethodPost, "/v1/sessions", map[string]any{
		"projectId": seedID, "note": "on archived",
	}, auth)
	assertCode(t, w, http.StatusConflict, "project_archived")

	w = doJSON(t, r, http.MethodDelete, "/v1/projects/"+seedID, nil, auth)
	assertCode(t, w, http.StatusConflict, "project_has_sessions")

	w = doJSON(t, r, http.MethodPost, "/v1/projects/"+created.ID+"/archive", nil, auth)
	assertCode(t, w, http.StatusConflict, "last_active_project")

	w = doJSON(t, r, http.MethodGet, "/v1/sessions?status=nope", nil, auth)
	assertCode(t, w, http.StatusBadRequest, "invalid_query")

	w = doJSON(t, r, http.MethodGet, "/v1/projects/"+uuid.NewString(), nil, auth)
	assertCode(t, w, http.StatusNotFound, "not_found")
}

func assertCode(t *testing.T, w *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("status = %d (%s), want %d / %s", w.Code, w.Body.String(), status, code)
	}
	var env errorEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.Error.Code != code {
		t.Fatalf("code = %q, want %q (%s)", env.Error.Code, code, w.Body.String())
	}
}
