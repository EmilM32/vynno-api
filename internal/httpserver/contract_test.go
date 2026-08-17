package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EmilM32/vynno-api/internal/service"
	"github.com/EmilM32/vynno-api/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func testService() *service.Service {
	user := store.DefaultUserID()
	mem := store.NewMemory(user, store.DefaultProfile(), store.DefaultProject())
	return service.New(mem, user)
}

func doJSON(t *testing.T, r http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
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
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestMeAndProjectsAndSessions(t *testing.T) {
	r := NewRouter(testService())

	w := doJSON(t, r, http.MethodGet, "/v1/me", nil)
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

	w = doJSON(t, r, http.MethodGet, "/v1/projects", nil)
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
	})
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
	})
	assertCode(t, w, http.StatusConflict, "code_in_use")

	w = doJSON(t, r, http.MethodGet, "/v1/sessions/active", nil)
	assertCode(t, w, http.StatusNotFound, "session_not_active")

	w = doJSON(t, r, http.MethodPost, "/v1/sessions", map[string]any{
		"projectId": seedID, "note": "  ",
	})
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
	})
	assertCode(t, w, http.StatusConflict, "session_already_active")

	w = doJSON(t, r, http.MethodPost, "/v1/sessions/"+live.ID+"/pause", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("pause = %d %s", w.Code, w.Body.String())
	}
	w = doJSON(t, r, http.MethodPost, "/v1/sessions/"+live.ID+"/pause", nil)
	assertCode(t, w, http.StatusConflict, "invalid_transition")

	w = doJSON(t, r, http.MethodPost, "/v1/sessions/"+live.ID+"/resume", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("resume = %d %s", w.Code, w.Body.String())
	}
	w = doJSON(t, r, http.MethodPost, "/v1/sessions/"+live.ID+"/stop", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("stop = %d %s", w.Code, w.Body.String())
	}

	w = doJSON(t, r, http.MethodPost, "/v1/projects/"+seedID+"/archive", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("archive seed = %d %s", w.Code, w.Body.String())
	}
	w = doJSON(t, r, http.MethodGet, "/v1/projects", nil)
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 || list.Items[0].ID != created.ID {
		t.Fatalf("default list should omit archived: %#v", list.Items)
	}

	w = doJSON(t, r, http.MethodPost, "/v1/sessions", map[string]any{
		"projectId": seedID, "note": "on archived",
	})
	assertCode(t, w, http.StatusConflict, "project_archived")

	w = doJSON(t, r, http.MethodDelete, "/v1/projects/"+seedID, nil)
	assertCode(t, w, http.StatusConflict, "project_has_sessions")

	w = doJSON(t, r, http.MethodPost, "/v1/projects/"+created.ID+"/archive", nil)
	assertCode(t, w, http.StatusConflict, "last_active_project")

	w = doJSON(t, r, http.MethodGet, "/v1/sessions?status=nope", nil)
	assertCode(t, w, http.StatusBadRequest, "invalid_query")

	w = doJSON(t, r, http.MethodGet, "/v1/projects/"+uuid.NewString(), nil)
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
