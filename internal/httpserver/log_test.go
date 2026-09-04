package httpserver

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestIDEchoAndLog(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	r := testRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	const id = "11111111-1111-1111-1111-111111111111"
	req.Header.Set(requestIDHeader, id)
	r.ServeHTTP(w, req)

	if got := w.Header().Get(requestIDHeader); got != id {
		t.Fatalf("X-Request-ID = %q, want %q", got, id)
	}

	var rec struct {
		Msg       string `json:"msg"`
		RequestID string `json:"request_id"`
		Path      string `json:"path"`
	}
	if err := json.NewDecoder(&buf).Decode(&rec); err != nil {
		t.Fatalf("log JSON: %v (%s)", err, buf.String())
	}
	if rec.Msg != "request" || rec.RequestID != id || rec.Path != "/v1/me" {
		t.Fatalf("log = %+v (%s)", rec, buf.String())
	}
}

func TestRequestIDGeneratedWhenMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := testRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	r.ServeHTTP(w, req)
	got := w.Header().Get(requestIDHeader)
	if !requestIDRe.MatchString(got) {
		t.Fatalf("generated X-Request-ID = %q", got)
	}
}
