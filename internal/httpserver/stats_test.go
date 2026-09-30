package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/EmilM32/vynno-api/internal/mail"
)

func dayTotalsPath(from, to, tz string) string {
	q := url.Values{}
	q.Set("from", from)
	q.Set("to", to)
	q.Set("timeZone", tz)
	return "/v1/stats/days?" + q.Encode()
}

func decodeDayTotals(t *testing.T, w *httptest.ResponseRecorder) []dayTotalDTO {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d %s", w.Code, w.Body.String())
	}
	var out listDTO[dayTotalDTO]
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out.Items
}

func manualSession(t *testing.T, r http.Handler, auth reqOpt, body map[string]any) {
	t.Helper()
	w := doJSON(t, r, http.MethodPost, "/v1/sessions/manual", body, auth)
	if w.Code != http.StatusCreated {
		t.Fatalf("manual = %d %s", w.Code, w.Body.String())
	}
}

func TestDayTotalsGroupsStoppedSessions(t *testing.T) {
	rec := &mail.Recorder{}
	r := testRouterWithMailer(t, rec)
	auth := withCookie(loginCookie(t, r))
	projectID := createProjectID(t, r, auth, "Stats")

	w := doJSON(t, r, http.MethodPost, "/v1/activity-types", map[string]any{"name": "review", "color": "primary"}, auth)
	var act activityTypeDTO
	if err := json.Unmarshal(w.Body.Bytes(), &act); err != nil {
		t.Fatal(err)
	}

	// 23:30 Warsaw on Sep 27 (runs past midnight) and 09:00 Warsaw on Sep 28.
	manualSession(t, r, auth, map[string]any{"projectId": projectID, "note": "late", "startedAt": "2026-09-27T21:30:00.000Z", "endedAt": "2026-09-27T23:00:00.000Z"})
	manualSession(t, r, auth, map[string]any{"projectId": projectID, "note": "a", "activityTypeId": act.ID, "startedAt": "2026-09-28T07:00:00.000Z", "endedAt": "2026-09-28T07:45:00.500Z"})
	manualSession(t, r, auth, map[string]any{"projectId": projectID, "note": "b", "activityTypeId": act.ID, "startedAt": "2026-09-28T08:00:00.000Z", "endedAt": "2026-09-28T08:15:00.000Z"})
	// Outside the range.
	manualSession(t, r, auth, map[string]any{"projectId": projectID, "note": "old", "startedAt": "2026-09-20T08:00:00.000Z", "endedAt": "2026-09-20T09:00:00.000Z"})

	// A live session and another account's sessions never count.
	w = doJSON(t, r, http.MethodPost, "/v1/sessions", map[string]any{"projectId": projectID, "note": "live"}, auth)
	if w.Code != http.StatusCreated {
		t.Fatalf("start = %d %s", w.Code, w.Body.String())
	}
	if w := registerWithCode(t, r, rec, "sam@example.com", "sam-pass-123", nil); w.Code != http.StatusCreated {
		t.Fatalf("register = %d", w.Code)
	}
	sam := withCookie(loginAs(t, r, "sam@example.com", "sam-pass-123"))
	var samProjects listDTO[projectDTO]
	_ = json.Unmarshal(doJSON(t, r, http.MethodGet, "/v1/projects", nil, sam).Body.Bytes(), &samProjects)
	manualSession(t, r, sam, map[string]any{"projectId": samProjects.Items[0].ID, "note": "sam", "startedAt": "2026-09-28T10:00:00.000Z", "endedAt": "2026-09-28T11:00:00.000Z"})

	w = doJSON(t, r, http.MethodGet, dayTotalsPath("2026-09-27", "2026-09-28", "Europe/Warsaw"), nil, auth)
	items := decodeDayTotals(t, w)
	if len(items) != 2 {
		t.Fatalf("items = %+v", items)
	}
	if items[0].Date != "2026-09-27" || items[0].ActivityTypeID != nil || items[0].DurationMs != 90*60_000 || items[0].SessionCount != 1 {
		t.Fatalf("first = %+v", items[0])
	}
	if items[1].Date != "2026-09-28" || items[1].ActivityTypeID == nil || *items[1].ActivityTypeID != act.ID ||
		items[1].DurationMs != 60*60_000+500 || items[1].SessionCount != 2 || items[1].ProjectID != projectID {
		t.Fatalf("second = %+v", items[1])
	}

	// The same instants in UTC fall on other dates.
	w = doJSON(t, r, http.MethodGet, dayTotalsPath("2026-09-28", "2026-09-28", "UTC"), nil, auth)
	items = decodeDayTotals(t, w)
	if len(items) != 1 || items[0].SessionCount != 2 {
		t.Fatalf("UTC items = %+v", items)
	}

	// Absent optionals are null, and an empty range is an empty list.
	w = doJSON(t, r, http.MethodGet, dayTotalsPath("2026-09-27", "2026-09-27", "Europe/Warsaw"), nil, auth)
	if got := w.Body.String(); got != `{"items":[{"date":"2026-09-27","projectId":"`+projectID+`","activityTypeId":null,"durationMs":5400000,"sessionCount":1}]}` {
		t.Fatalf("body = %s", got)
	}
	w = doJSON(t, r, http.MethodGet, dayTotalsPath("2026-01-01", "2026-01-31", "UTC"), nil, auth)
	if got := w.Body.String(); got != `{"items":[]}` {
		t.Fatalf("empty body = %s", got)
	}
}

func TestDayTotalsRejectsBadQueries(t *testing.T) {
	r := testRouter(t)
	auth := withCookie(loginCookie(t, r))
	for _, path := range []string{
		"/v1/stats/days",
		"/v1/stats/days?from=2026-09-01&to=2026-09-30",
		dayTotalsPath("2026-09-30", "2026-09-01", "UTC"),
		dayTotalsPath("2025-01-01", "2026-09-30", "UTC"),
		dayTotalsPath("2026-09-01", "2026-09-30", "Nowhere/City"),
		dayTotalsPath("2026-09-01", "tomorrow", "UTC"),
	} {
		assertCode(t, doJSON(t, r, http.MethodGet, path, nil, auth), http.StatusBadRequest, "invalid_query")
	}
	assertCode(t, doJSON(t, r, http.MethodGet, dayTotalsPath("2026-09-01", "2026-09-30", "UTC"), nil), http.StatusUnauthorized, "unauthorized")
}
