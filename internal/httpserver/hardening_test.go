package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/EmilM32/vynno-api/internal/mail"
	"github.com/gin-gonic/gin"
)

func TestWriteError_UnknownErrorUses500Code(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	writeError(c, errors.New("boom"))
	assertCode(t, w, http.StatusInternalServerError, domain.CodeInternalError)
}

func TestLoginRateLimitSetsRetryAfter(t *testing.T) {
	r, svc := testAuth(t, mail.Discard())
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	svc.Now = func() time.Time { return now }
	for i := 0; i < 10; i++ {
		w := doJSON(t, r, http.MethodPost, "/v1/auth/login", map[string]any{
			"email": "alexdev@vynno.local", "password": "wrong-password",
		})
		assertCode(t, w, http.StatusUnauthorized, "invalid_credentials")
	}
	w := doJSON(t, r, http.MethodPost, "/v1/auth/login", map[string]any{
		"email": "alexdev@vynno.local", "password": testPassword,
	})
	assertCode(t, w, http.StatusTooManyRequests, "rate_limited")
	if w.Header().Get("Retry-After") != "900" {
		t.Fatalf("Retry-After = %q", w.Header().Get("Retry-After"))
	}
}

func TestRegisterCodeIPCapDoesNotSend(t *testing.T) {
	rec := &mail.Recorder{}
	r := testRouterWithMailer(t, rec)
	w := doJSON(t, r, http.MethodPost, "/v1/auth/register/code", map[string]any{"email": "not-an-email"})
	assertCode(t, w, http.StatusBadRequest, "invalid_body")
	w = doJSON(t, r, http.MethodPost, "/v1/auth/register/code", map[string]any{"email": "alexdev@vynno.local"})
	assertCode(t, w, http.StatusConflict, "email_in_use")
	if len(rec.Messages) != 0 {
		t.Fatalf("mail before a real send: %d", len(rec.Messages))
	}
	for i := 1; i <= 5; i++ {
		w = doJSON(t, r, http.MethodPost, "/v1/auth/register/code", map[string]any{
			"email": "cap" + strconv.Itoa(i) + "@example.com",
		})
		if w.Code != http.StatusNoContent {
			t.Fatalf("send %d = %d %s", i, w.Code, w.Body.String())
		}
	}
	if len(rec.Messages) != 5 {
		t.Fatalf("mail = %d", len(rec.Messages))
	}
	w = doJSON(t, r, http.MethodPost, "/v1/auth/register/code", map[string]any{"email": "cap6@example.com"})
	assertCode(t, w, http.StatusTooManyRequests, "rate_limited")
	if len(rec.Messages) != 5 {
		t.Fatalf("6th sent mail: %d", len(rec.Messages))
	}
	if w.Header().Get("Retry-After") == "" {
		t.Fatal("missing Retry-After")
	}
}

func TestTrustedProxyUsesForwardedFor(t *testing.T) {
	rec := &mail.Recorder{}
	r := testRouterWithMailer(t, rec)
	send := func(n int, remote, forwarded string) *httptest.ResponseRecorder {
		t.Helper()
		return doJSON(t, r, http.MethodPost, "/v1/auth/register/code", map[string]any{
			"email": "p" + strconv.Itoa(n) + "@example.com",
		}, withRemoteAddr(remote), withForwardedFor(forwarded))
	}
	for i := 1; i <= 5; i++ {
		w := send(i, "198.51.100.8:443", "203.0.113.9")
		if w.Code != http.StatusNoContent {
			t.Fatalf("untrusted send %d = %d %s", i, w.Code, w.Body.String())
		}
	}
	w := send(6, "198.51.100.8:443", "203.0.113.50")
	assertCode(t, w, http.StatusTooManyRequests, "rate_limited")
	if len(rec.Messages) != 5 {
		t.Fatalf("untrusted peer honored X-Forwarded-For: mail=%d", len(rec.Messages))
	}

	for i := 7; i <= 11; i++ {
		w = send(i, "127.0.0.1:443", "203.0.113.77")
		if w.Code != http.StatusNoContent {
			t.Fatalf("trusted send %d = %d %s", i, w.Code, w.Body.String())
		}
	}
	w = send(12, "127.0.0.1:443", "203.0.113.77")
	assertCode(t, w, http.StatusTooManyRequests, "rate_limited")
	w = send(13, "127.0.0.1:443", "203.0.113.88")
	if w.Code != http.StatusNoContent {
		t.Fatalf("other forwarded client = %d %s", w.Code, w.Body.String())
	}
	w = send(14, "203.0.113.77:9", "198.51.100.1")
	assertCode(t, w, http.StatusTooManyRequests, "rate_limited")
}

func TestNULNameIsRejectedAndNotStored(t *testing.T) {
	r := testRouter(t)
	auth := withCookie(loginCookie(t, r))
	before := projectCount(t, r, auth)
	w := doJSON(t, r, http.MethodPost, "/v1/projects", map[string]any{
		"name": "bad\x00name", "color": "#112233",
	}, auth)
	assertCode(t, w, http.StatusBadRequest, "invalid_body")
	if got := projectCount(t, r, auth); got != before {
		t.Fatalf("projects = %d, want %d", got, before)
	}
}

func TestJSONBodyErrors(t *testing.T) {
	r := testRouter(t)
	w := doRaw(t, r, http.MethodPost, "/v1/auth/login", `{"email":1,"password":"long-enough"}`)
	assertCode(t, w, http.StatusBadRequest, "invalid_body")
	if !strings.Contains(w.Body.String(), "email has the wrong type.") {
		t.Fatalf("body = %s", w.Body.String())
	}
	w = doRaw(t, r, http.MethodPost, "/v1/auth/login", `{`)
	assertCode(t, w, http.StatusBadRequest, "invalid_json")
	w = doRaw(t, r, http.MethodPost, "/v1/auth/login", `{"email":"a@example.com","password":"long-enough"}{"x":1}`)
	assertCode(t, w, http.StatusBadRequest, "invalid_json")
	w = doRaw(t, r, http.MethodPost, "/v1/auth/login", ``)
	assertCode(t, w, http.StatusBadRequest, "invalid_json")
	w = doRaw(t, r, http.MethodPost, "/v1/auth/login", `{"email":"a@example.com","password":"long-enough","extra":true}`)
	assertCode(t, w, http.StatusBadRequest, "invalid_body")

	auth := withCookie(loginCookie(t, r))
	w = doJSON(t, r, http.MethodGet, "/v1/projects", nil, auth)
	var projects listDTO[projectDTO]
	if err := json.Unmarshal(w.Body.Bytes(), &projects); err != nil || len(projects.Items) == 0 {
		t.Fatalf("projects: %v %s", err, w.Body.String())
	}
	w = doRaw(t, r, http.MethodPatch, "/v1/projects/"+projects.Items[0].ID, `{"nope":1}`, auth)
	assertCode(t, w, http.StatusBadRequest, "invalid_body")
}

func TestNoRouteAndNoMethod(t *testing.T) {
	r := testRouter(t)
	w := doJSON(t, r, http.MethodGet, "/v1/does-not-exist", nil)
	assertCode(t, w, http.StatusNotFound, "not_found")
	w = doJSON(t, r, http.MethodPost, "/healthz", nil)
	assertCode(t, w, http.StatusNotFound, "not_found")
	w = doJSON(t, r, http.MethodGet, "/v1/auth/login", nil)
	assertCode(t, w, http.StatusNotFound, "not_found")

	w = doJSON(t, r, http.MethodGet, "/healthz", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("healthz = %d %s", w.Code, w.Body.String())
	}
	w = doJSON(t, r, http.MethodGet, "/readyz", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("readyz = %d %s", w.Code, w.Body.String())
	}
	w = doJSON(t, r, http.MethodGet, "/openapi.json", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("openapi = %d", w.Code)
	}
	req := httptest.NewRequest(http.MethodGet, "/swagger/", nil)
	sw := httptest.NewRecorder()
	r.ServeHTTP(sw, req)
	if sw.Code != http.StatusOK {
		t.Fatalf("swagger = %d", sw.Code)
	}
}

func TestManualSessionYear0001Rejected(t *testing.T) {
	r := testRouter(t)
	auth := withCookie(loginCookie(t, r))
	id := firstProjectID(t, r, auth)
	w := doJSON(t, r, http.MethodPost, "/v1/sessions/manual", map[string]any{
		"projectId": id,
		"note":      "ancient",
		"startedAt": "0001-01-01T00:00:00.000Z",
		"endedAt":   "0001-01-01T01:00:00.000Z",
	}, auth)
	assertCode(t, w, http.StatusBadRequest, "invalid_body")
}

func TestNoteAndTicketLength(t *testing.T) {
	r := testRouter(t)
	auth := withCookie(loginCookie(t, r))
	id := firstProjectID(t, r, auth)

	w := doJSON(t, r, http.MethodPost, "/v1/sessions", map[string]any{
		"projectId": id, "note": strings.Repeat("a", 501),
	}, auth)
	assertCode(t, w, http.StatusBadRequest, "invalid_body")
	w = doJSON(t, r, http.MethodGet, "/v1/sessions/active", nil, auth)
	assertCode(t, w, http.StatusNotFound, "session_not_active")

	w = doJSON(t, r, http.MethodPost, "/v1/sessions", map[string]any{
		"projectId": id, "note": "ok", "ticketId": strings.Repeat("t", 65),
	}, auth)
	assertCode(t, w, http.StatusBadRequest, "invalid_body")
	w = doJSON(t, r, http.MethodGet, "/v1/sessions/active", nil, auth)
	assertCode(t, w, http.StatusNotFound, "session_not_active")

	note := strings.Repeat("a", 499) + "😀"
	ticket := strings.Repeat("t", 64)
	w = doJSON(t, r, http.MethodPost, "/v1/sessions", map[string]any{
		"projectId": id, "note": note, "ticketId": ticket,
	}, auth)
	if w.Code != http.StatusCreated {
		t.Fatalf("create = %d %s", w.Code, w.Body.String())
	}
	var sess sessionDTO
	if err := json.Unmarshal(w.Body.Bytes(), &sess); err != nil {
		t.Fatal(err)
	}
	if sess.Note != note || utf8.RuneCountInString(sess.Note) != 500 {
		t.Fatalf("note len = %d", utf8.RuneCountInString(sess.Note))
	}
	if sess.TicketID == nil || *sess.TicketID != ticket || utf8.RuneCountInString(*sess.TicketID) != 64 {
		t.Fatalf("ticket = %#v", sess.TicketID)
	}
}

func TestActivityTypeZeroWidthCollides(t *testing.T) {
	r := testRouter(t)
	auth := withCookie(loginCookie(t, r))
	w := doJSON(t, r, http.MethodPost, "/v1/activity-types", map[string]any{
		"name": "DUP", "color": "primary",
	}, auth)
	if w.Code != http.StatusCreated {
		t.Fatalf("create = %d %s", w.Code, w.Body.String())
	}
	w = doJSON(t, r, http.MethodPost, "/v1/activity-types", map[string]any{
		"name": "D\u200bUP", "color": "secondary",
	}, auth)
	assertCode(t, w, http.StatusConflict, "name_in_use")
}

func TestEmptyStatusQueryIsUnfiltered(t *testing.T) {
	r := testRouter(t)
	auth := withCookie(loginCookie(t, r))
	w := doJSON(t, r, http.MethodGet, "/v1/sessions?status=", nil, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("status= = %d %s", w.Code, w.Body.String())
	}
}

func doRaw(t *testing.T, r http.Handler, method, path, body string, opts ...reqOpt) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for _, opt := range opts {
		opt(req)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func withRemoteAddr(addr string) reqOpt {
	return func(r *http.Request) { r.RemoteAddr = addr }
}

func withForwardedFor(ip string) reqOpt {
	return func(r *http.Request) { r.Header.Set("X-Forwarded-For", ip) }
}

func projectCount(t *testing.T, r http.Handler, auth reqOpt) int {
	t.Helper()
	w := doJSON(t, r, http.MethodGet, "/v1/projects", nil, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("list = %d %s", w.Code, w.Body.String())
	}
	var projects listDTO[projectDTO]
	if err := json.Unmarshal(w.Body.Bytes(), &projects); err != nil {
		t.Fatal(err)
	}
	return len(projects.Items)
}

func firstProjectID(t *testing.T, r http.Handler, auth reqOpt) string {
	t.Helper()
	w := doJSON(t, r, http.MethodGet, "/v1/projects", nil, auth)
	var projects listDTO[projectDTO]
	if err := json.Unmarshal(w.Body.Bytes(), &projects); err != nil || len(projects.Items) == 0 {
		t.Fatalf("projects: %v %s", err, w.Body.String())
	}
	return projects.Items[0].ID
}
