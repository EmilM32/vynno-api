package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/EmilM32/vynno-api/internal/mail"
)

func TestLoginInvalidCredentials(t *testing.T) {
	r := testRouter(t)
	w := doJSON(t, r, http.MethodPost, "/v1/auth/login", map[string]any{
		"email": "alexdev@vynno.local", "password": "wrong-password",
	})
	assertCode(t, w, http.StatusUnauthorized, "invalid_credentials")
}

func TestRegisterAndIsolation(t *testing.T) {
	rec := &mail.Recorder{}
	r := testRouterWithMailer(t, rec)
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

	w = registerWithCode(t, r, rec, "bob@example.com", "bob-pass-1", map[string]any{"displayName": "Bob"})
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
	if strings.Contains(w.Body.String(), `"code"`) {
		t.Fatalf("response leaked code: %s", w.Body.String())
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

	w = doJSON(t, r, http.MethodPost, "/v1/auth/register/code", map[string]any{"email": "bob@example.com"})
	assertCode(t, w, http.StatusConflict, "email_in_use")

	w = doJSON(t, r, http.MethodPost, "/v1/auth/register", map[string]any{
		"email": "bob@example.com", "password": "other-pass-1", "code": "000000",
	})
	assertCode(t, w, http.StatusConflict, "email_in_use")

	w = registerWithCode(t, r, rec, "  Pat@Example.COM  ", "pat-pass-1", nil)
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

func TestRegisterCodeHappyPath(t *testing.T) {
	rec := &mail.Recorder{}
	r := testRouterWithMailer(t, rec)
	w := doJSON(t, r, http.MethodPost, "/v1/auth/register/code", map[string]any{"email": "carol@example.com"})
	if w.Code != http.StatusNoContent {
		t.Fatalf("register/code = %d %s", w.Code, w.Body.String())
	}
	if w.Body.Len() != 0 {
		t.Fatalf("code response body = %s", w.Body.String())
	}
	if len(rec.Messages) != 1 {
		t.Fatalf("mail count = %d", len(rec.Messages))
	}
	msg := rec.Messages[0]
	if msg.To != "carol@example.com" || msg.Subject != "Your Vynno confirmation code" {
		t.Fatalf("mail = %+v", msg)
	}
	code := otpFromRecorder(t, rec)
	w = doJSON(t, r, http.MethodPost, "/v1/auth/register", map[string]any{
		"email": "carol@example.com", "password": "carol-pass-1", "code": code,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("register = %d %s", w.Code, w.Body.String())
	}
	found := false
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookie {
			found = true
		}
	}
	if !found {
		t.Fatal("register missing cookie")
	}
}

func TestRegisterCodeRejected(t *testing.T) {
	rec := &mail.Recorder{}
	r, svc := testAuth(t, rec)

	w := doJSON(t, r, http.MethodPost, "/v1/auth/register", map[string]any{
		"email": "dana@example.com", "password": "dana-pass-1",
	})
	assertCode(t, w, http.StatusBadRequest, "invalid_body")

	w = doJSON(t, r, http.MethodPost, "/v1/auth/register", map[string]any{
		"email": "dana@example.com", "password": "dana-pass-1", "code": "12ab56",
	})
	assertCode(t, w, http.StatusBadRequest, "invalid_body")

	w = doJSON(t, r, http.MethodPost, "/v1/auth/register", map[string]any{
		"email": "dana@example.com", "password": "dana-pass-1", "code": "000000",
	})
	assertCode(t, w, http.StatusUnauthorized, "invalid_code")

	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	svc.Now = func() time.Time { return now }
	w = doJSON(t, r, http.MethodPost, "/v1/auth/register/code", map[string]any{"email": "dana@example.com"})
	if w.Code != http.StatusNoContent {
		t.Fatalf("register/code = %d %s", w.Code, w.Body.String())
	}
	code := otpFromRecorder(t, rec)

	w = doJSON(t, r, http.MethodPost, "/v1/auth/register", map[string]any{
		"email": "dana@example.com", "password": "dana-pass-1", "code": "111111",
	})
	assertCode(t, w, http.StatusUnauthorized, "invalid_code")

	svc.Now = func() time.Time { return now.Add(domain.OTPTTL + time.Second) }
	w = doJSON(t, r, http.MethodPost, "/v1/auth/register", map[string]any{
		"email": "dana@example.com", "password": "dana-pass-1", "code": code,
	})
	assertCode(t, w, http.StatusUnauthorized, "invalid_code")
}

func TestRegisterCodeTakenEmailDoesNotSend(t *testing.T) {
	rec := &mail.Recorder{}
	r := testRouterWithMailer(t, rec)
	w := doJSON(t, r, http.MethodPost, "/v1/auth/register/code", map[string]any{
		"email": "alexdev@vynno.local",
	})
	assertCode(t, w, http.StatusConflict, "email_in_use")
	if len(rec.Messages) != 0 {
		t.Fatalf("sent mail for taken email: %+v", rec.Messages)
	}
}

func TestRegisterCodeCooldownAndSendCap(t *testing.T) {
	rec := &mail.Recorder{}
	r, svc := testAuth(t, rec)
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	svc.Now = func() time.Time { return now }

	w := doJSON(t, r, http.MethodPost, "/v1/auth/register/code", map[string]any{"email": "erin@example.com"})
	if w.Code != http.StatusNoContent {
		t.Fatalf("first send = %d %s", w.Code, w.Body.String())
	}
	w = doJSON(t, r, http.MethodPost, "/v1/auth/register/code", map[string]any{"email": "erin@example.com"})
	assertCode(t, w, http.StatusTooManyRequests, "rate_limited")

	for i := 1; i < domain.OTPSendsPerHour; i++ {
		now = now.Add(domain.OTPSendCooldown)
		w = doJSON(t, r, http.MethodPost, "/v1/auth/register/code", map[string]any{"email": "erin@example.com"})
		if w.Code != http.StatusNoContent {
			t.Fatalf("send %d = %d %s", i+1, w.Code, w.Body.String())
		}
	}
	now = now.Add(domain.OTPSendCooldown)
	w = doJSON(t, r, http.MethodPost, "/v1/auth/register/code", map[string]any{"email": "erin@example.com"})
	assertCode(t, w, http.StatusTooManyRequests, "rate_limited")
}

func TestRegisterCodeGuessCap(t *testing.T) {
	rec := &mail.Recorder{}
	r := testRouterWithMailer(t, rec)
	w := doJSON(t, r, http.MethodPost, "/v1/auth/register/code", map[string]any{"email": "finn@example.com"})
	if w.Code != http.StatusNoContent {
		t.Fatalf("register/code = %d %s", w.Code, w.Body.String())
	}
	real := otpFromRecorder(t, rec)
	wrong := "000000"
	if wrong == real {
		wrong = "111111"
	}
	for i := 0; i < domain.OTPMaxAttempts-1; i++ {
		w = doJSON(t, r, http.MethodPost, "/v1/auth/register", map[string]any{
			"email": "finn@example.com", "password": "finn-pass-1", "code": wrong,
		})
		assertCode(t, w, http.StatusUnauthorized, "invalid_code")
	}
	w = doJSON(t, r, http.MethodPost, "/v1/auth/register", map[string]any{
		"email": "finn@example.com", "password": "finn-pass-1", "code": wrong,
	})
	assertCode(t, w, http.StatusTooManyRequests, "rate_limited")
	w = doJSON(t, r, http.MethodPost, "/v1/auth/register", map[string]any{
		"email": "finn@example.com", "password": "finn-pass-1", "code": real,
	})
	assertCode(t, w, http.StatusUnauthorized, "invalid_code")
}

func TestPasswordForgotUnknownEmailNoMail(t *testing.T) {
	rec := &mail.Recorder{}
	r := testRouterWithMailer(t, rec)
	w := doJSON(t, r, http.MethodPost, "/v1/auth/password/forgot", map[string]any{
		"email": "nobody@example.com",
	})
	if w.Code != http.StatusNoContent {
		t.Fatalf("forgot unknown = %d %s", w.Code, w.Body.String())
	}
	if w.Body.Len() != 0 {
		t.Fatalf("forgot body = %s", w.Body.String())
	}
	if len(rec.Messages) != 0 {
		t.Fatalf("sent mail for unknown email: %+v", rec.Messages)
	}
}

func TestPasswordForgotMalformedEmail(t *testing.T) {
	r := testRouter(t)
	w := doJSON(t, r, http.MethodPost, "/v1/auth/password/forgot", map[string]any{
		"email": "not-an-email",
	})
	assertCode(t, w, http.StatusBadRequest, "invalid_body")
}

func TestPasswordForgotDoesNotLeakEnumeration(t *testing.T) {
	rec := &mail.Recorder{}
	r, svc := testAuth(t, rec)
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	svc.Now = func() time.Time { return now }

	known := doJSON(t, r, http.MethodPost, "/v1/auth/password/forgot", map[string]any{
		"email": "alexdev@vynno.local",
	})
	unknown := doJSON(t, r, http.MethodPost, "/v1/auth/password/forgot", map[string]any{
		"email": "nobody@example.com",
	})
	if known.Code != http.StatusNoContent || unknown.Code != http.StatusNoContent {
		t.Fatalf("first forgot known=%d unknown=%d", known.Code, unknown.Code)
	}
	if len(rec.Messages) != 1 || rec.Messages[0].To != "alexdev@vynno.local" {
		t.Fatalf("mail = %+v", rec.Messages)
	}

	known2 := doJSON(t, r, http.MethodPost, "/v1/auth/password/forgot", map[string]any{
		"email": "alexdev@vynno.local",
	})
	unknown2 := doJSON(t, r, http.MethodPost, "/v1/auth/password/forgot", map[string]any{
		"email": "nobody@example.com",
	})
	assertCode(t, known2, http.StatusTooManyRequests, "rate_limited")
	assertCode(t, unknown2, http.StatusTooManyRequests, "rate_limited")
}

func TestPasswordResetHappyPath(t *testing.T) {
	rec := &mail.Recorder{}
	r := testRouterWithMailer(t, rec)
	const email = "gina@example.com"
	w := registerWithCode(t, r, rec, email, "old-pass-1", nil)
	if w.Code != http.StatusCreated {
		t.Fatalf("register = %d %s", w.Code, w.Body.String())
	}

	w = doJSON(t, r, http.MethodPost, "/v1/auth/password/forgot", map[string]any{"email": email})
	if w.Code != http.StatusNoContent {
		t.Fatalf("forgot = %d %s", w.Code, w.Body.String())
	}
	msg := rec.Messages[len(rec.Messages)-1]
	if msg.To != email || msg.Subject != "Your Vynno password reset code" {
		t.Fatalf("reset mail = %+v", msg)
	}
	if strings.Contains(w.Body.String(), otpFromRecorder(t, rec)) {
		t.Fatalf("response leaked code: %s", w.Body.String())
	}
	code := otpFromRecorder(t, rec)

	w = doJSON(t, r, http.MethodPost, "/v1/auth/password/reset", map[string]any{
		"email": email, "code": code, "password": "new-pass-1",
	})
	if w.Code != http.StatusNoContent {
		t.Fatalf("reset = %d %s", w.Code, w.Body.String())
	}
	if w.Body.Len() != 0 {
		t.Fatalf("reset body = %s", w.Body.String())
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookie && c.Value != "" && c.MaxAge != -1 {
			t.Fatalf("reset must not set a session cookie: %+v", c)
		}
	}

	w = doJSON(t, r, http.MethodPost, "/v1/auth/login", map[string]any{
		"email": email, "password": "old-pass-1",
	})
	assertCode(t, w, http.StatusUnauthorized, "invalid_credentials")

	ck := loginAs(t, r, email, "new-pass-1")
	if ck == nil {
		t.Fatal("login with new password failed")
	}
}

func TestPasswordResetRevokesSessions(t *testing.T) {
	rec := &mail.Recorder{}
	r := testRouterWithMailer(t, rec)
	const email = "hank@example.com"
	w := registerWithCode(t, r, rec, email, "old-pass-1", nil)
	if w.Code != http.StatusCreated {
		t.Fatalf("register = %d %s", w.Code, w.Body.String())
	}
	ck1 := loginAs(t, r, email, "old-pass-1")
	ck2 := loginAs(t, r, email, "old-pass-1")

	w = doJSON(t, r, http.MethodPost, "/v1/auth/password/forgot", map[string]any{"email": email})
	if w.Code != http.StatusNoContent {
		t.Fatalf("forgot = %d %s", w.Code, w.Body.String())
	}
	w = doJSON(t, r, http.MethodPost, "/v1/auth/password/reset", map[string]any{
		"email": email, "code": otpFromRecorder(t, rec), "password": "new-pass-1",
	})
	if w.Code != http.StatusNoContent {
		t.Fatalf("reset = %d %s", w.Code, w.Body.String())
	}

	w = doJSON(t, r, http.MethodGet, "/v1/me", nil, withCookie(ck1))
	assertCode(t, w, http.StatusUnauthorized, "unauthorized")
	w = doJSON(t, r, http.MethodGet, "/v1/me", nil, withCookie(ck2))
	assertCode(t, w, http.StatusUnauthorized, "unauthorized")
}

func TestPasswordResetRejected(t *testing.T) {
	rec := &mail.Recorder{}
	r, svc := testAuth(t, rec)

	w := doJSON(t, r, http.MethodPost, "/v1/auth/password/reset", map[string]any{
		"email": "iris@example.com", "password": "iris-pass-1",
	})
	assertCode(t, w, http.StatusBadRequest, "invalid_body")

	w = doJSON(t, r, http.MethodPost, "/v1/auth/password/reset", map[string]any{
		"email": "iris@example.com", "code": "12ab56", "password": "iris-pass-1",
	})
	assertCode(t, w, http.StatusBadRequest, "invalid_body")

	w = doJSON(t, r, http.MethodPost, "/v1/auth/password/reset", map[string]any{
		"email": "iris@example.com", "code": "000000", "password": "iris-pass-1",
	})
	assertCode(t, w, http.StatusUnauthorized, "invalid_code")

	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	svc.Now = func() time.Time { return now }
	_ = registerWithCode(t, r, rec, "iris@example.com", "iris-pass-1", nil)
	w = doJSON(t, r, http.MethodPost, "/v1/auth/password/forgot", map[string]any{"email": "iris@example.com"})
	if w.Code != http.StatusNoContent {
		t.Fatalf("forgot = %d %s", w.Code, w.Body.String())
	}
	code := otpFromRecorder(t, rec)

	w = doJSON(t, r, http.MethodPost, "/v1/auth/password/reset", map[string]any{
		"email": "iris@example.com", "code": "111111", "password": "new-pass-1",
	})
	assertCode(t, w, http.StatusUnauthorized, "invalid_code")

	svc.Now = func() time.Time { return now.Add(domain.OTPTTL + time.Second) }
	w = doJSON(t, r, http.MethodPost, "/v1/auth/password/reset", map[string]any{
		"email": "iris@example.com", "code": code, "password": "new-pass-1",
	})
	assertCode(t, w, http.StatusUnauthorized, "invalid_code")
}

func TestPasswordResetGuessCap(t *testing.T) {
	rec := &mail.Recorder{}
	r := testRouterWithMailer(t, rec)
	_ = registerWithCode(t, r, rec, "jade@example.com", "jade-pass-1", nil)
	w := doJSON(t, r, http.MethodPost, "/v1/auth/password/forgot", map[string]any{"email": "jade@example.com"})
	if w.Code != http.StatusNoContent {
		t.Fatalf("forgot = %d %s", w.Code, w.Body.String())
	}
	real := otpFromRecorder(t, rec)
	wrong := "000000"
	if wrong == real {
		wrong = "111111"
	}
	for i := 0; i < domain.OTPMaxAttempts-1; i++ {
		w = doJSON(t, r, http.MethodPost, "/v1/auth/password/reset", map[string]any{
			"email": "jade@example.com", "code": wrong, "password": "new-pass-1",
		})
		assertCode(t, w, http.StatusUnauthorized, "invalid_code")
	}
	w = doJSON(t, r, http.MethodPost, "/v1/auth/password/reset", map[string]any{
		"email": "jade@example.com", "code": wrong, "password": "new-pass-1",
	})
	assertCode(t, w, http.StatusTooManyRequests, "rate_limited")
	w = doJSON(t, r, http.MethodPost, "/v1/auth/password/reset", map[string]any{
		"email": "jade@example.com", "code": real, "password": "new-pass-1",
	})
	assertCode(t, w, http.StatusUnauthorized, "invalid_code")
}

func TestBootstrapLoginSkipsMail(t *testing.T) {
	rec := &mail.Recorder{}
	r := testRouterWithMailer(t, rec)
	ck := loginCookie(t, r)
	if ck == nil {
		t.Fatal("missing cookie")
	}
	if len(rec.Messages) != 0 {
		t.Fatalf("login sent mail: %+v", rec.Messages)
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
