package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/EmilM32/vynno-api/internal/mail"
)

const alexEmail = "alexdev@vynno.local"

func lastMailTo(t *testing.T, rec *mail.Recorder, to string) mail.Message {
	t.Helper()
	for i := len(rec.Messages) - 1; i >= 0; i-- {
		if rec.Messages[i].To == to {
			return rec.Messages[i]
		}
	}
	t.Fatalf("no mail to %s", to)
	return mail.Message{}
}

func codeIn(t *testing.T, msg mail.Message) string {
	t.Helper()
	m := otpPat.FindStringSubmatch(msg.Text)
	if m == nil {
		t.Fatalf("no code in %q", msg.Text)
	}
	return m[1]
}

func TestChangePasswordKeepsThisSessionAndSignsOutOthers(t *testing.T) {
	rec := &mail.Recorder{}
	r := testRouterWithMailer(t, rec)
	here := withCookie(loginCookie(t, r))
	elsewhere := withCookie(loginCookie(t, r))

	w := doJSON(t, r, http.MethodPost, "/v1/auth/password/change", map[string]any{
		"currentPassword": testPassword, "newPassword": "brand-new-pass",
	}, here)
	if w.Code != http.StatusNoContent || w.Body.Len() != 0 {
		t.Fatalf("change = %d %q", w.Code, w.Body.String())
	}

	if w := doJSON(t, r, http.MethodGet, "/v1/me", nil, here); w.Code != http.StatusOK {
		t.Fatalf("this session after change = %d", w.Code)
	}
	assertCode(t, doJSON(t, r, http.MethodGet, "/v1/me", nil, elsewhere), http.StatusUnauthorized, "unauthorized")

	w = doJSON(t, r, http.MethodPost, "/v1/auth/login", map[string]any{"email": alexEmail, "password": testPassword})
	assertCode(t, w, http.StatusUnauthorized, "invalid_credentials")
	loginAs(t, r, alexEmail, "brand-new-pass")

	notice := lastMailTo(t, rec, alexEmail)
	if !strings.Contains(notice.Subject, "password was changed") {
		t.Fatalf("notice subject = %q", notice.Subject)
	}
}

func TestChangePasswordRejected(t *testing.T) {
	r := testRouter(t)
	auth := withCookie(loginCookie(t, r))

	w := doJSON(t, r, http.MethodPost, "/v1/auth/password/change", map[string]any{
		"currentPassword": "not-my-password", "newPassword": "brand-new-pass",
	}, auth)
	assertCode(t, w, http.StatusUnauthorized, "invalid_credentials")
	// A wrong password is not a lost session.
	if w := doJSON(t, r, http.MethodGet, "/v1/me", nil, auth); w.Code != http.StatusOK {
		t.Fatalf("session after wrong password = %d", w.Code)
	}

	w = doJSON(t, r, http.MethodPost, "/v1/auth/password/change", map[string]any{
		"currentPassword": testPassword, "newPassword": "short",
	}, auth)
	assertCode(t, w, http.StatusBadRequest, "invalid_body")

	w = doJSON(t, r, http.MethodPost, "/v1/auth/password/change", map[string]any{
		"currentPassword": testPassword, "newPassword": "brand-new-pass", "email": alexEmail,
	}, auth)
	assertCode(t, w, http.StatusBadRequest, "invalid_body")

	w = doJSON(t, r, http.MethodPost, "/v1/auth/password/change", map[string]any{
		"currentPassword": testPassword, "newPassword": "brand-new-pass",
	})
	assertCode(t, w, http.StatusUnauthorized, "unauthorized")

	// Nothing changed.
	loginAs(t, r, alexEmail, testPassword)
}

func TestChangePasswordWrongCurrentSharesLoginCap(t *testing.T) {
	r := testRouter(t)
	auth := withCookie(loginCookie(t, r))
	for i := 0; i < 10; i++ {
		w := doJSON(t, r, http.MethodPost, "/v1/auth/password/change", map[string]any{
			"currentPassword": "not-my-password", "newPassword": "brand-new-pass",
		}, auth)
		assertCode(t, w, http.StatusUnauthorized, "invalid_credentials")
	}
	w := doJSON(t, r, http.MethodPost, "/v1/auth/password/change", map[string]any{
		"currentPassword": testPassword, "newPassword": "brand-new-pass",
	}, auth)
	assertCode(t, w, http.StatusTooManyRequests, "rate_limited")
	if w.Header().Get("Retry-After") == "" {
		t.Fatal("missing Retry-After")
	}
	w = doJSON(t, r, http.MethodPost, "/v1/auth/login", map[string]any{"email": alexEmail, "password": testPassword})
	assertCode(t, w, http.StatusTooManyRequests, "rate_limited")
}

func TestChangeEmailHappyPath(t *testing.T) {
	rec := &mail.Recorder{}
	r := testRouterWithMailer(t, rec)
	here := withCookie(loginCookie(t, r))
	elsewhere := withCookie(loginCookie(t, r))

	w := doJSON(t, r, http.MethodPost, "/v1/auth/email/code", map[string]any{
		"email": "  Alex.New@Example.com ", "password": testPassword,
	}, here)
	if w.Code != http.StatusNoContent {
		t.Fatalf("email/code = %d %s", w.Code, w.Body.String())
	}
	code := codeIn(t, lastMailTo(t, rec, "alex.new@example.com"))

	// Nothing changes until the code is confirmed.
	loginAs(t, r, alexEmail, testPassword)

	w = doJSON(t, r, http.MethodPost, "/v1/auth/email/change", map[string]any{
		"email": "alex.new@example.com", "code": code,
	}, here)
	if w.Code != http.StatusOK {
		t.Fatalf("email/change = %d %s", w.Code, w.Body.String())
	}
	var p profileDTO
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.Email != "alex.new@example.com" {
		t.Fatalf("profile email = %q", p.Email)
	}

	if w := doJSON(t, r, http.MethodGet, "/v1/me", nil, here); w.Code != http.StatusOK {
		t.Fatalf("this session after change = %d", w.Code)
	}
	assertCode(t, doJSON(t, r, http.MethodGet, "/v1/me", nil, elsewhere), http.StatusUnauthorized, "unauthorized")

	w = doJSON(t, r, http.MethodPost, "/v1/auth/login", map[string]any{"email": alexEmail, "password": testPassword})
	assertCode(t, w, http.StatusUnauthorized, "invalid_credentials")
	loginAs(t, r, "alex.new@example.com", testPassword)

	notice := lastMailTo(t, rec, alexEmail)
	if !strings.Contains(notice.Text, "alex.new@example.com") {
		t.Fatalf("old-address notice = %q", notice.Text)
	}

	// The code is spent.
	w = doJSON(t, r, http.MethodPost, "/v1/auth/email/change", map[string]any{
		"email": "alex.new@example.com", "code": code,
	}, here)
	assertCode(t, w, http.StatusUnauthorized, "invalid_code")
}

func TestChangeEmailCodeRejected(t *testing.T) {
	rec := &mail.Recorder{}
	r := testRouterWithMailer(t, rec)
	auth := withCookie(loginCookie(t, r))
	if w := registerWithCode(t, r, rec, "bob@example.com", "bob-pass-12", nil); w.Code != http.StatusCreated {
		t.Fatalf("register bob = %d", w.Code)
	}
	sent := len(rec.Messages)

	w := doJSON(t, r, http.MethodPost, "/v1/auth/email/code", map[string]any{"email": "new@example.com", "password": "wrong-password"}, auth)
	assertCode(t, w, http.StatusUnauthorized, "invalid_credentials")
	w = doJSON(t, r, http.MethodPost, "/v1/auth/email/code", map[string]any{"email": alexEmail, "password": testPassword}, auth)
	assertCode(t, w, http.StatusBadRequest, "invalid_body")
	w = doJSON(t, r, http.MethodPost, "/v1/auth/email/code", map[string]any{"email": "Bob@Example.com", "password": testPassword}, auth)
	assertCode(t, w, http.StatusConflict, "email_in_use")
	w = doJSON(t, r, http.MethodPost, "/v1/auth/email/code", map[string]any{"email": "not an email", "password": testPassword}, auth)
	assertCode(t, w, http.StatusBadRequest, "invalid_body")
	if len(rec.Messages) != sent {
		t.Fatalf("rejected requests sent %d mails", len(rec.Messages)-sent)
	}

	w = doJSON(t, r, http.MethodPost, "/v1/auth/email/change", map[string]any{"email": "new@example.com", "code": "123456"}, auth)
	assertCode(t, w, http.StatusUnauthorized, "invalid_code")
	w = doJSON(t, r, http.MethodPost, "/v1/auth/email/change", map[string]any{"email": "new@example.com", "code": "12a456"}, auth)
	assertCode(t, w, http.StatusBadRequest, "invalid_body")
}

func TestChangeEmailCodeIsBoundToTheAccountThatAsked(t *testing.T) {
	rec := &mail.Recorder{}
	r := testRouterWithMailer(t, rec)
	alex := withCookie(loginCookie(t, r))
	if w := registerWithCode(t, r, rec, "bob@example.com", "bob-pass-12", nil); w.Code != http.StatusCreated {
		t.Fatalf("register bob = %d", w.Code)
	}
	bob := withCookie(loginAs(t, r, "bob@example.com", "bob-pass-12"))

	w := doJSON(t, r, http.MethodPost, "/v1/auth/email/code", map[string]any{"email": "shared@example.com", "password": testPassword}, alex)
	if w.Code != http.StatusNoContent {
		t.Fatalf("email/code = %d %s", w.Code, w.Body.String())
	}
	code := codeIn(t, lastMailTo(t, rec, "shared@example.com"))

	for i := 0; i < 6; i++ {
		w = doJSON(t, r, http.MethodPost, "/v1/auth/email/change", map[string]any{"email": "shared@example.com", "code": code}, bob)
		assertCode(t, w, http.StatusUnauthorized, "invalid_code")
	}
	// Bob's tries did not spend Alex's guesses.
	w = doJSON(t, r, http.MethodPost, "/v1/auth/email/change", map[string]any{"email": "shared@example.com", "code": code}, alex)
	if w.Code != http.StatusOK {
		t.Fatalf("alex change = %d %s", w.Code, w.Body.String())
	}
}

func TestChangeEmailLosesToALaterRegister(t *testing.T) {
	rec := &mail.Recorder{}
	r := testRouterWithMailer(t, rec)
	auth := withCookie(loginCookie(t, r))

	w := doJSON(t, r, http.MethodPost, "/v1/auth/email/code", map[string]any{"email": "carol@example.com", "password": testPassword}, auth)
	if w.Code != http.StatusNoContent {
		t.Fatalf("email/code = %d %s", w.Code, w.Body.String())
	}
	code := codeIn(t, lastMailTo(t, rec, "carol@example.com"))
	if w := registerWithCode(t, r, rec, "carol@example.com", "carol-pass-1", nil); w.Code != http.StatusCreated {
		t.Fatalf("register carol = %d %s", w.Code, w.Body.String())
	}

	w = doJSON(t, r, http.MethodPost, "/v1/auth/email/change", map[string]any{"email": "carol@example.com", "code": code}, auth)
	assertCode(t, w, http.StatusConflict, "email_in_use")
	loginAs(t, r, alexEmail, testPassword)
}
