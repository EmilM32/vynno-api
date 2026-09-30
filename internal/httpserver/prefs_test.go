package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EmilM32/vynno-api/internal/mail"
)

func decodePrefs(t *testing.T, w *httptest.ResponseRecorder) prefsDTO {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d %s", w.Code, w.Body.String())
	}
	var p prefsDTO
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func createProjectID(t *testing.T, r http.Handler, auth reqOpt, name string) string {
	t.Helper()
	w := doJSON(t, r, http.MethodPost, "/v1/projects", map[string]any{"name": name, "color": "#3b82f6"}, auth)
	if w.Code != http.StatusCreated {
		t.Fatalf("create project = %d %s", w.Code, w.Body.String())
	}
	var p projectDTO
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	return p.ID
}

func TestPrefsDefaultsToNullsAndRoundTrips(t *testing.T) {
	r := testRouter(t)
	auth := withCookie(loginCookie(t, r))

	p := decodePrefs(t, doJSON(t, r, http.MethodGet, "/v1/me/prefs", nil, auth))
	if p.DailyTargetMs != nil || p.DefaultProjectID != nil {
		t.Fatalf("fresh prefs = %+v, want nulls", p)
	}
	// Absent optionals are JSON null, not omitted.
	w := doJSON(t, r, http.MethodGet, "/v1/me/prefs", nil, auth)
	if got := w.Body.String(); got != `{"dailyTargetMs":null,"defaultProjectId":null}` {
		t.Fatalf("body = %s", got)
	}

	projectID := createProjectID(t, r, auth, "Docs")
	p = decodePrefs(t, doJSON(t, r, http.MethodPatch, "/v1/me/prefs", map[string]any{
		"dailyTargetMs": 6 * 3_600_000, "defaultProjectId": projectID,
	}, auth))
	if p.DailyTargetMs == nil || *p.DailyTargetMs != 6*3_600_000 {
		t.Fatalf("dailyTargetMs = %v", p.DailyTargetMs)
	}
	if p.DefaultProjectID == nil || *p.DefaultProjectID != projectID {
		t.Fatalf("defaultProjectId = %v", p.DefaultProjectID)
	}

	// Omitted field stays; null clears.
	p = decodePrefs(t, doJSON(t, r, http.MethodPatch, "/v1/me/prefs", map[string]any{"dailyTargetMs": nil}, auth))
	if p.DailyTargetMs != nil {
		t.Fatalf("dailyTargetMs after null = %v", *p.DailyTargetMs)
	}
	if p.DefaultProjectID == nil || *p.DefaultProjectID != projectID {
		t.Fatalf("omitted defaultProjectId changed: %v", p.DefaultProjectID)
	}

	p = decodePrefs(t, doJSON(t, r, http.MethodGet, "/v1/me/prefs", nil, auth))
	if p.DailyTargetMs != nil || p.DefaultProjectID == nil {
		t.Fatalf("GET after patch = %+v", p)
	}
}

func TestPrefsValidation(t *testing.T) {
	r := testRouter(t)
	auth := withCookie(loginCookie(t, r))

	for _, body := range []map[string]any{
		{"dailyTargetMs": 59_999},
		{"dailyTargetMs": 86_400_001},
		{"dailyTargetMs": 1.5},
		{"dailyTargetMs": "28800000"},
		{"defaultProjectId": 42},
		{"theme": "light"},
	} {
		w := doJSON(t, r, http.MethodPatch, "/v1/me/prefs", body, auth)
		assertCode(t, w, http.StatusBadRequest, "invalid_body")
	}
	for _, ms := range []int64{60_000, 86_400_000} {
		p := decodePrefs(t, doJSON(t, r, http.MethodPatch, "/v1/me/prefs", map[string]any{"dailyTargetMs": ms}, auth))
		if p.DailyTargetMs == nil || *p.DailyTargetMs != ms {
			t.Fatalf("bound %d = %v", ms, p.DailyTargetMs)
		}
	}

	w := doJSON(t, r, http.MethodPatch, "/v1/me/prefs", map[string]any{"defaultProjectId": "not-a-uuid"}, auth)
	assertCode(t, w, http.StatusNotFound, "not_found")
	w = doJSON(t, r, http.MethodPatch, "/v1/me/prefs", map[string]any{"defaultProjectId": "0b4f3c3e-8f7e-4a57-9d40-1a2b3c4d5e6f"}, auth)
	assertCode(t, w, http.StatusNotFound, "not_found")

	// A failed patch writes nothing.
	p := decodePrefs(t, doJSON(t, r, http.MethodGet, "/v1/me/prefs", nil, auth))
	if p.DefaultProjectID != nil {
		t.Fatalf("defaultProjectId after 404 = %v", *p.DefaultProjectID)
	}
}

func TestPrefsRejectAnotherUsersProject(t *testing.T) {
	rec := &mail.Recorder{}
	r := testRouterWithMailer(t, rec)
	alex := withCookie(loginCookie(t, r))
	foreign := createProjectID(t, r, alex, "Alex only")

	w := registerWithCode(t, r, rec, "sam@example.com", "sam-pass-123", nil)
	if w.Code != http.StatusCreated {
		t.Fatalf("register = %d %s", w.Code, w.Body.String())
	}
	sam := withCookie(loginAs(t, r, "sam@example.com", "sam-pass-123"))

	w = doJSON(t, r, http.MethodPatch, "/v1/me/prefs", map[string]any{"defaultProjectId": foreign}, sam)
	assertCode(t, w, http.StatusNotFound, "not_found")

	// Prefs are per account.
	decodePrefs(t, doJSON(t, r, http.MethodPatch, "/v1/me/prefs", map[string]any{"dailyTargetMs": 3_600_000}, alex))
	p := decodePrefs(t, doJSON(t, r, http.MethodGet, "/v1/me/prefs", nil, sam))
	if p.DailyTargetMs != nil {
		t.Fatalf("sam sees alex's target: %v", *p.DailyTargetMs)
	}
}

func TestPrefsDefaultProjectClearedOnHardDelete(t *testing.T) {
	r := testRouter(t)
	auth := withCookie(loginCookie(t, r))
	projectID := createProjectID(t, r, auth, "Short-lived")

	decodePrefs(t, doJSON(t, r, http.MethodPatch, "/v1/me/prefs", map[string]any{"defaultProjectId": projectID}, auth))
	w := doJSON(t, r, http.MethodDelete, "/v1/projects/"+projectID, nil, auth)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete project = %d %s", w.Code, w.Body.String())
	}
	p := decodePrefs(t, doJSON(t, r, http.MethodGet, "/v1/me/prefs", nil, auth))
	if p.DefaultProjectID != nil {
		t.Fatalf("defaultProjectId after delete = %v", *p.DefaultProjectID)
	}
}

func TestPrefsRequireAuth(t *testing.T) {
	r := testRouter(t)
	assertCode(t, doJSON(t, r, http.MethodGet, "/v1/me/prefs", nil), http.StatusUnauthorized, "unauthorized")
	assertCode(t, doJSON(t, r, http.MethodPatch, "/v1/me/prefs", map[string]any{"dailyTargetMs": 60_000}), http.StatusUnauthorized, "unauthorized")
}
