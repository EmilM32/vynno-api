package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestLoginInvalidCredentials(t *testing.T) {
	r := testRouter(t)
	w := doJSON(t, r, http.MethodPost, "/v1/auth/login", map[string]any{
		"email": "alexdev@vynno.local", "password": "wrong-password",
	})
	assertCode(t, w, http.StatusUnauthorized, "invalid_credentials")
}

func TestRegisterAndIsolation(t *testing.T) {
	r := testRouter(t)
	alice := loginCookie(t, r)

	w := doJSON(t, r, http.MethodGet, "/v1/projects", nil, withCookie(alice))
	var aliceList listDTO[projectDTO]
	if err := json.Unmarshal(w.Body.Bytes(), &aliceList); err != nil {
		t.Fatal(err)
	}
	if len(aliceList.Items) == 0 {
		t.Fatal("alice has no projects")
	}
	aliceProject := aliceList.Items[0].ID

	w = doJSON(t, r, http.MethodPost, "/v1/auth/register", map[string]any{
		"email": "bob@example.com", "password": "bob-pass-1", "displayName": "Bob",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("register = %d %s", w.Code, w.Body.String())
	}
	var created authResponse
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Profile.DisplayName != "Bob" || created.Profile.Email != "bob@example.com" {
		t.Fatalf("profile: %+v", created.Profile)
	}
	var bobCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookie {
			bobCookie = c
		}
	}
	if bobCookie == nil {
		t.Fatal("register missing cookie")
	}

	w = doJSON(t, r, http.MethodGet, "/v1/projects/"+aliceProject, nil, withCookie(bobCookie))
	assertCode(t, w, http.StatusNotFound, "not_found")

	w = doJSON(t, r, http.MethodGet, "/v1/projects", nil, withCookie(bobCookie))
	var bobList listDTO[projectDTO]
	if err := json.Unmarshal(w.Body.Bytes(), &bobList); err != nil {
		t.Fatal(err)
	}
	if len(bobList.Items) != 1 || bobList.Items[0].ID == aliceProject {
		t.Fatalf("bob should have his own project: %#v", bobList.Items)
	}

	w = doJSON(t, r, http.MethodPost, "/v1/auth/register", map[string]any{
		"email": "bob@example.com", "password": "other-pass-1",
	})
	assertCode(t, w, http.StatusConflict, "email_in_use")

	w = doJSON(t, r, http.MethodPost, "/v1/auth/register", map[string]any{
		"email": "  Pat@Example.COM  ", "password": "pat-pass-1",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("register omitted displayName = %d %s", w.Code, w.Body.String())
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Profile.DisplayName != "" || created.Profile.Email != "pat@example.com" {
		t.Fatalf("omitted displayName profile: %+v", created.Profile)
	}
}

func TestLogoutRevokesSession(t *testing.T) {
	r := testRouter(t)
	ck := loginCookie(t, r)
	w := doJSON(t, r, http.MethodPost, "/v1/auth/logout", nil, withCookie(ck))
	if w.Code != http.StatusNoContent {
		t.Fatalf("logout = %d %s", w.Code, w.Body.String())
	}
	w = doJSON(t, r, http.MethodGet, "/v1/me", nil, withCookie(ck))
	assertCode(t, w, http.StatusUnauthorized, "unauthorized")
}

func TestRememberMeCookieMaxAge(t *testing.T) {
	r := testRouter(t)
	w := doJSON(t, r, http.MethodPost, "/v1/auth/login", map[string]any{
		"email": "alexdev@vynno.local", "password": testPassword,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("login default = %d %s", w.Code, w.Body.String())
	}
	found := false
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookie {
			found = true
			if c.MaxAge != rememberMaxAge {
				t.Fatalf("default MaxAge = %d, want %d", c.MaxAge, rememberMaxAge)
			}
			if !c.HttpOnly {
				t.Fatal("cookie should be HttpOnly")
			}
		}
	}
	if !found {
		t.Fatal("missing cookie")
	}

	w = doJSON(t, r, http.MethodPost, "/v1/auth/login", map[string]any{
		"email": "alexdev@vynno.local", "password": testPassword, "rememberMe": false,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("login rememberMe false = %d %s", w.Code, w.Body.String())
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookie && c.MaxAge != 0 {
			t.Fatalf("session cookie MaxAge = %d, want 0", c.MaxAge)
		}
	}
}

func TestBearerAuth(t *testing.T) {
	r := testRouter(t)
	w := doJSON(t, r, http.MethodPost, "/v1/auth/login", map[string]any{
		"email": "alexdev@vynno.local", "password": testPassword,
	})
	var token string
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookie {
			token = c.Value
		}
	}
	if token == "" {
		t.Fatal("no token")
	}
	w = doJSON(t, r, http.MethodGet, "/v1/me", nil, withBearer(token))
	if w.Code != http.StatusOK {
		t.Fatalf("bearer /me = %d %s", w.Code, w.Body.String())
	}
}

func TestCSRFBadOrigin(t *testing.T) {
	r := testRouter(t)
	ck := loginCookie(t, r)
	w := doJSON(t, r, http.MethodPost, "/v1/projects", map[string]any{
		"name": "Evil", "color": "#000000",
	}, withCookie(ck), withOrigin("https://evil.example"))
	if w.Code == http.StatusCreated {
		t.Fatal("disallowed Origin must not create")
	}

	w = doJSON(t, r, http.MethodPost, "/v1/projects", map[string]any{
		"name": "EvilRef", "color": "#000000",
	}, withCookie(ck), withReferer("https://evil.example/attack"))
	assertCode(t, w, http.StatusUnauthorized, "unauthorized")

	w = doJSON(t, r, http.MethodPost, "/v1/projects", map[string]any{
		"name": "Ok", "color": "#000000",
	}, withCookie(ck), withOrigin("http://localhost:5173"))
	if w.Code != http.StatusCreated {
		t.Fatalf("good origin = %d %s", w.Code, w.Body.String())
	}
}

func TestHealthzUnauthenticated(t *testing.T) {
	r := testRouter(t)
	w := doJSON(t, r, http.MethodGet, "/healthz", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("healthz = %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"ok"`) {
		t.Fatalf("body %s", w.Body.String())
	}

	w = doJSON(t, r, http.MethodGet, "/readyz", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("readyz = %d %s", w.Code, w.Body.String())
	}
}
