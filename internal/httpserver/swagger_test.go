package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAPIMatchesGinRoutes(t *testing.T) {
	r := testRouter(t)
	fromGin := map[string]map[string]struct{}{}
	for _, rt := range r.Routes() {
		if isDocsRoute(rt.Path) {
			continue
		}
		p := openAPIPath(rt.Path)
		if fromGin[p] == nil {
			fromGin[p] = map[string]struct{}{}
		}
		fromGin[p][rt.Method] = struct{}{}
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("openapi.json = %d %s", w.Code, w.Body.String())
	}
	fromSpec, err := specPathsFromJSON(w.Body.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if formatPathSet(fromGin) != formatPathSet(fromSpec) {
		t.Fatalf("gin routes and OpenAPI paths differ\n-- gin --\n%s\n-- spec --\n%s",
			formatPathSet(fromGin), formatPathSet(fromSpec))
	}
}

func TestOpenAPIDocumentShape(t *testing.T) {
	r := testRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	r.ServeHTTP(w, req)
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("content-type = %q", ct)
	}
	var doc map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc["openapi"] != "3.0.3" {
		t.Fatalf("openapi = %#v", doc["openapi"])
	}
	paths, _ := doc["paths"].(map[string]any)
	for _, p := range []string{
		"/healthz", "/readyz",
		"/v1/auth/login", "/v1/auth/register", "/v1/auth/register/code", "/v1/auth/logout",
		"/v1/auth/password/forgot", "/v1/auth/password/reset",
		"/v1/me", "/v1/me/avatar", "/v1/avatars/{id}",
		"/v1/projects", "/v1/projects/{id}",
		"/v1/activity-types", "/v1/activity-types/{id}",
		"/v1/sessions", "/v1/sessions/active", "/v1/sessions/manual", "/v1/sessions/{id}",
	} {
		if _, ok := paths[p]; !ok {
			t.Fatalf("missing path %s", p)
		}
	}
	if _, ok := paths["/swagger/"]; ok {
		t.Fatal("spec must not document /swagger/")
	}
	if _, ok := paths["/openapi.json"]; ok {
		t.Fatal("spec must not document /openapi.json")
	}
	components, _ := doc["components"].(map[string]any)
	schemas, _ := components["schemas"].(map[string]any)
	for _, name := range []string{
		"Profile", "Project", "ProjectList", "Session", "SessionList",
		"ActivityType", "ErrorEnvelope", "RegisterRequest", "LoginRequest",
		"PasswordForgotRequest", "ResetPasswordRequest",
		"StartSessionRequest", "UpdateSessionRequest", "CreateManualSessionRequest",
		"Count", "HealthResponse",
	} {
		if _, ok := schemas[name]; !ok {
			t.Fatalf("missing schema %s", name)
		}
	}
	sessionList, _ := schemas["SessionList"].(map[string]any)
	props, _ := sessionList["properties"].(map[string]any)
	if _, ok := props["nextCursor"]; !ok {
		t.Fatal("SessionList missing nextCursor")
	}
}

func TestSwaggerUIServesHTML(t *testing.T) {
	r := testRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/swagger/", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /swagger/ = %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(strings.ToLower(w.Body.String()), "swagger") {
		t.Fatalf("expected swagger UI html, got %s", w.Body.String()[:min(200, w.Body.Len())])
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/swagger", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("GET /swagger = %d, want 302", w.Code)
	}
}

func TestCSRFAllowsPublicAPIOrigin(t *testing.T) {
	r := testRouter(t)
	ck := loginCookie(t, r)
	w := doJSON(t, r, http.MethodPost, "/v1/projects", map[string]any{
		"name": "FromSwagger", "color": "#000000",
	}, withCookie(ck), withOrigin("http://localhost:8080"))
	if w.Code != http.StatusCreated {
		t.Fatalf("PUBLIC_API_ORIGIN = %d %s", w.Code, w.Body.String())
	}
}

func isDocsRoute(path string) bool {
	return path == "/openapi.json" || path == "/swagger" || strings.HasPrefix(path, "/swagger/")
}
