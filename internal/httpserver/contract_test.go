package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
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
	w = doJSON(t, r, http.MethodPost, "/v1/sessions/manual", map[string]any{"projectId": uuid.NewString()})
	assertCode(t, w, http.StatusUnauthorized, "unauthorized")
	w = doJSON(t, r, http.MethodPatch, "/v1/sessions/"+uuid.NewString(), map[string]any{"note": "x"})
	assertCode(t, w, http.StatusUnauthorized, "unauthorized")
	w = doJSON(t, r, http.MethodDelete, "/v1/sessions/"+uuid.NewString(), nil)
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

func TestActivityTypesCRUD(t *testing.T) {
	r := testRouter(t)
	ck := loginCookie(t, r)
	auth := withCookie(ck)

	w := doJSON(t, r, http.MethodGet, "/v1/activity-types", nil, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /activity-types = %d %s", w.Code, w.Body.String())
	}
	var list listDTO[activityTypeDTO]
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 0 {
		t.Fatalf("new user should have empty activity types: %#v", list.Items)
	}

	w = doJSON(t, r, http.MethodPost, "/v1/activity-types", map[string]any{
		"name": " Coding ", "color": "Secondary",
	}, auth)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /activity-types = %d %s", w.Code, w.Body.String())
	}
	var created activityTypeDTO
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Name != "Coding" || created.Color != "secondary" {
		t.Fatalf("normalized: %#v", created)
	}

	w = doJSON(t, r, http.MethodPost, "/v1/activity-types", map[string]any{
		"name": "coding", "color": "primary",
	}, auth)
	assertCode(t, w, http.StatusConflict, "name_in_use")

	w = doJSON(t, r, http.MethodPost, "/v1/activity-types", map[string]any{
		"name": "coding", "color": "#22c55e",
	}, auth)
	assertCode(t, w, http.StatusBadRequest, "invalid_body")

	w = doJSON(t, r, http.MethodPatch, "/v1/activity-types/"+created.ID, map[string]any{
		"color": "error",
	}, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("PATCH = %d %s", w.Code, w.Body.String())
	}

	w = doJSON(t, r, http.MethodGet, "/v1/projects", nil, auth)
	var projects listDTO[projectDTO]
	if err := json.Unmarshal(w.Body.Bytes(), &projects); err != nil {
		t.Fatal(err)
	}
	projectID := projects.Items[0].ID

	w = doJSON(t, r, http.MethodPost, "/v1/sessions", map[string]any{
		"projectId": projectID, "note": "with type", "activityTypeId": created.ID,
	}, auth)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /sessions = %d %s", w.Code, w.Body.String())
	}
	var live sessionDTO
	if err := json.Unmarshal(w.Body.Bytes(), &live); err != nil {
		t.Fatal(err)
	}
	if live.ActivityTypeID == nil || *live.ActivityTypeID != created.ID {
		t.Fatalf("session activityTypeId: %#v", live.ActivityTypeID)
	}

	w = doJSON(t, r, http.MethodDelete, "/v1/activity-types/"+created.ID, nil, auth)
	assertCode(t, w, http.StatusConflict, "activity_type_has_sessions")

	w = doJSON(t, r, http.MethodPost, "/v1/sessions/"+live.ID+"/stop", nil, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("stop = %d %s", w.Code, w.Body.String())
	}

	w = doJSON(t, r, http.MethodGet, "/v1/activity-types/"+created.ID+"/session-count", nil, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("session-count = %d %s", w.Code, w.Body.String())
	}

	w = doJSON(t, r, http.MethodPost, "/v1/sessions", map[string]any{
		"projectId": projectID, "note": "unknown type", "activityTypeId": uuid.NewString(),
	}, auth)
	assertCode(t, w, http.StatusNotFound, "not_found")

	w = doJSON(t, r, http.MethodDelete, "/v1/activity-types/"+created.ID, nil, auth)
	assertCode(t, w, http.StatusConflict, "activity_type_has_sessions")

	other := doJSON(t, r, http.MethodPost, "/v1/activity-types", map[string]any{
		"name": "meeting", "color": "tertiary",
	}, auth)
	if other.Code != http.StatusCreated {
		t.Fatalf("POST meeting = %d %s", other.Code, other.Body.String())
	}
	var meeting activityTypeDTO
	if err := json.Unmarshal(other.Body.Bytes(), &meeting); err != nil {
		t.Fatal(err)
	}
	w = doJSON(t, r, http.MethodDelete, "/v1/activity-types/"+meeting.ID, nil, auth)
	if w.Code != http.StatusNoContent {
		t.Fatalf("DELETE unused = %d %s", w.Code, w.Body.String())
	}
}

func TestSessionEditDeleteManual(t *testing.T) {
	r := testRouter(t)
	ck := loginCookie(t, r)
	auth := withCookie(ck)

	w := doJSON(t, r, http.MethodGet, "/v1/projects", nil, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /projects = %d %s", w.Code, w.Body.String())
	}
	var projects listDTO[projectDTO]
	if err := json.Unmarshal(w.Body.Bytes(), &projects); err != nil {
		t.Fatal(err)
	}
	projectID := projects.Items[0].ID

	w = doJSON(t, r, http.MethodPost, "/v1/projects", map[string]any{
		"name": "Tools", "color": "#22c55e",
	}, auth)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /projects = %d %s", w.Code, w.Body.String())
	}
	var other projectDTO
	if err := json.Unmarshal(w.Body.Bytes(), &other); err != nil {
		t.Fatal(err)
	}

	w = doJSON(t, r, http.MethodPost, "/v1/sessions", map[string]any{
		"projectId": projectID, "note": "Live work",
	}, auth)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /sessions = %d %s", w.Code, w.Body.String())
	}
	var live sessionDTO
	if err := json.Unmarshal(w.Body.Bytes(), &live); err != nil {
		t.Fatal(err)
	}
	startedAt := live.StartedAt

	w = doJSON(t, r, http.MethodPatch, "/v1/sessions/"+live.ID, map[string]any{
		"note": "Renamed live",
	}, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("PATCH live note = %d %s", w.Code, w.Body.String())
	}
	if err := json.Unmarshal(w.Body.Bytes(), &live); err != nil {
		t.Fatal(err)
	}
	if live.Note != "Renamed live" || live.Status != "active" || live.StartedAt != startedAt {
		t.Fatalf("PATCH omit: %+v", live)
	}

	w = doJSON(t, r, http.MethodPatch, "/v1/sessions/"+live.ID, map[string]any{
		"status": "stopped",
	}, auth)
	assertCode(t, w, http.StatusBadRequest, "invalid_body")

	w = doJSON(t, r, http.MethodPatch, "/v1/sessions/"+live.ID, map[string]any{
		"endedAt": "2026-03-11T10:00:00.000Z",
	}, auth)
	assertCode(t, w, http.StatusBadRequest, "invalid_body")

	w = doJSON(t, r, http.MethodPost, "/v1/sessions/manual", map[string]any{
		"projectId": projectID,
		"note":      "Forgot to start",
		"startedAt": "2026-03-11T08:00:00.000Z",
		"endedAt":   "2026-03-11T10:15:00.000Z",
	}, auth)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /sessions/manual = %d %s", w.Code, w.Body.String())
	}
	var log sessionDTO
	if err := json.Unmarshal(w.Body.Bytes(), &log); err != nil {
		t.Fatal(err)
	}
	if log.Status != "stopped" || log.EndedAt == nil || log.PausedAt != nil {
		t.Fatalf("manual: %+v", log)
	}

	w = doJSON(t, r, http.MethodPatch, "/v1/sessions/"+log.ID, map[string]any{
		"projectId":      other.ID,
		"ticketId":       nil,
		"activityTypeId": nil,
		"endedAt":        "2026-03-11T11:00:00.000Z",
	}, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("PATCH log = %d %s", w.Code, w.Body.String())
	}
	if err := json.Unmarshal(w.Body.Bytes(), &log); err != nil {
		t.Fatal(err)
	}
	if log.ProjectID != other.ID || log.TicketID != nil || *log.EndedAt != "2026-03-11T11:00:00.000Z" {
		t.Fatalf("patched log: %+v", log)
	}

	w = doJSON(t, r, http.MethodPatch, "/v1/sessions/"+log.ID, map[string]any{
		"endedAt": "2026-03-11T07:00:00.000Z",
	}, auth)
	assertCode(t, w, http.StatusBadRequest, "invalid_body")

	w = doJSON(t, r, http.MethodPatch, "/v1/sessions/"+log.ID, map[string]any{
		"pausedMs": int64(9 * 60 * 60 * 1000),
	}, auth)
	assertCode(t, w, http.StatusBadRequest, "invalid_body")

	w = doJSON(t, r, http.MethodDelete, "/v1/sessions/"+live.ID, nil, auth)
	if w.Code != http.StatusNoContent {
		t.Fatalf("DELETE live = %d %s", w.Code, w.Body.String())
	}
	w = doJSON(t, r, http.MethodGet, "/v1/sessions/active", nil, auth)
	assertCode(t, w, http.StatusNotFound, "session_not_active")

	w = doJSON(t, r, http.MethodPost, "/v1/sessions", map[string]any{
		"projectId": projectID, "note": "After delete",
	}, auth)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST after delete live = %d %s", w.Code, w.Body.String())
	}
	if err := json.Unmarshal(w.Body.Bytes(), &live); err != nil {
		t.Fatal(err)
	}

	w = doJSON(t, r, http.MethodDelete, "/v1/sessions/"+log.ID, nil, auth)
	if w.Code != http.StatusNoContent {
		t.Fatalf("DELETE log = %d %s", w.Code, w.Body.String())
	}
	w = doJSON(t, r, http.MethodGet, "/v1/sessions/"+log.ID, nil, auth)
	assertCode(t, w, http.StatusNotFound, "not_found")

	w = doJSON(t, r, http.MethodPost, "/v1/sessions/"+live.ID+"/stop", nil, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("stop = %d %s", w.Code, w.Body.String())
	}

	w = doJSON(t, r, http.MethodPost, "/v1/projects/"+projectID+"/archive", nil, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("archive = %d %s", w.Code, w.Body.String())
	}
	w = doJSON(t, r, http.MethodPost, "/v1/sessions/manual", map[string]any{
		"projectId": projectID,
		"note":      "On archived",
		"startedAt": "2026-03-10T08:00:00.000Z",
		"endedAt":   "2026-03-10T09:00:00.000Z",
	}, auth)
	if w.Code != http.StatusCreated {
		t.Fatalf("manual on archived = %d %s", w.Code, w.Body.String())
	}
	w = doJSON(t, r, http.MethodPost, "/v1/sessions", map[string]any{
		"projectId": projectID, "note": "start archived",
	}, auth)
	assertCode(t, w, http.StatusConflict, "project_archived")

	w = doJSON(t, r, http.MethodDelete, "/v1/sessions/"+uuid.NewString(), nil, auth)
	assertCode(t, w, http.StatusNotFound, "not_found")
}

func TestListSessionsPagination(t *testing.T) {
	r := testRouter(t)
	auth := withCookie(loginCookie(t, r))

	w := doJSON(t, r, http.MethodGet, "/v1/projects", nil, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /projects = %d %s", w.Code, w.Body.String())
	}
	var projects listDTO[projectDTO]
	if err := json.Unmarshal(w.Body.Bytes(), &projects); err != nil {
		t.Fatal(err)
	}
	projectID := projects.Items[0].ID

	starts := []string{
		"2026-03-11T12:00:00.000Z",
		"2026-03-11T11:00:00.000Z",
		"2026-03-11T10:00:00.000Z",
		"2026-03-11T10:00:00.000Z",
		"2026-03-11T09:00:00.000Z",
	}
	for i, started := range starts {
		ended := "2026-03-11T13:00:00.000Z"
		w = doJSON(t, r, http.MethodPost, "/v1/sessions/manual", map[string]any{
			"projectId": projectID,
			"note":      "page-" + strconv.Itoa(i),
			"startedAt": started,
			"endedAt":   ended,
		}, auth)
		if w.Code != http.StatusCreated {
			t.Fatalf("manual %d = %d %s", i, w.Code, w.Body.String())
		}
	}

	w = doJSON(t, r, http.MethodGet, "/v1/sessions?limit=0", nil, auth)
	assertCode(t, w, http.StatusBadRequest, "invalid_query")
	w = doJSON(t, r, http.MethodGet, "/v1/sessions?limit=101", nil, auth)
	assertCode(t, w, http.StatusBadRequest, "invalid_query")
	w = doJSON(t, r, http.MethodGet, "/v1/sessions?limit=-1", nil, auth)
	assertCode(t, w, http.StatusBadRequest, "invalid_query")
	w = doJSON(t, r, http.MethodGet, "/v1/sessions?cursor=not-a-cursor", nil, auth)
	assertCode(t, w, http.StatusBadRequest, "invalid_query")

	w = doJSON(t, r, http.MethodGet, "/v1/sessions?limit=100", nil, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("full list = %d %s", w.Code, w.Body.String())
	}
	var full sessionListDTO
	if err := json.Unmarshal(w.Body.Bytes(), &full); err != nil {
		t.Fatal(err)
	}
	if full.NextCursor != nil {
		t.Fatal("expected last page")
	}
	if len(full.Items) != len(starts) {
		t.Fatalf("full items = %d, want %d", len(full.Items), len(starts))
	}

	var got []sessionDTO
	seen := map[string]bool{}
	cursor := ""
	pages := 0
	for {
		path := "/v1/sessions?limit=2"
		if cursor != "" {
			path += "&cursor=" + cursor
		}
		w = doJSON(t, r, http.MethodGet, path, nil, auth)
		if w.Code != http.StatusOK {
			t.Fatalf("page = %d %s", w.Code, w.Body.String())
		}
		var page sessionListDTO
		if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
			t.Fatal(err)
		}
		pages++
		for _, item := range page.Items {
			if seen[item.ID] {
				t.Fatalf("duplicate id %s", item.ID)
			}
			seen[item.ID] = true
			got = append(got, item)
		}
		if page.NextCursor == nil {
			break
		}
		cursor = *page.NextCursor
		if pages > 10 {
			t.Fatal("too many pages")
		}
	}
	if len(got) != len(full.Items) {
		t.Fatalf("paged %d, full %d", len(got), len(full.Items))
	}
	for i := range full.Items {
		if got[i].ID != full.Items[i].ID {
			t.Fatalf("order mismatch at %d: %s vs %s", i, got[i].ID, full.Items[i].ID)
		}
	}

	w = doJSON(t, r, http.MethodGet, "/v1/sessions?status=stopped&limit=2", nil, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("filtered = %d %s", w.Code, w.Body.String())
	}
	var filtered sessionListDTO
	if err := json.Unmarshal(w.Body.Bytes(), &filtered); err != nil {
		t.Fatal(err)
	}
	if len(filtered.Items) != 2 || filtered.NextCursor == nil {
		t.Fatalf("filtered page = %#v", filtered)
	}
	for _, item := range filtered.Items {
		if item.Status != "stopped" {
			t.Fatalf("status = %s", item.Status)
		}
	}
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
