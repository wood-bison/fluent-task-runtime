package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wood-bison/fluent-task-runtime/internal/engine"
)

func TestHealthAndProfiles(t *testing.T) {
	handler := NewServer(engine.NewCatalogue())
	for _, path := range []string{"/v1/health/live", "/v1/health/ready", "/v1/profiles"} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s: status = %d", path, recorder.Code)
		}
		if !strings.Contains(recorder.Header().Get("content-type"), "application/json") {
			t.Fatalf("%s: missing JSON content type", path)
		}
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/profiles", nil))
	var body struct {
		Profiles []struct {
			ID string `json:"id"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Profiles) != 5 || body.Profiles[0].ID != "node" || body.Profiles[4].ID != "java" {
		t.Fatalf("unexpected profile catalogue: %#v", body.Profiles)
	}
}

func TestRunRefusesBeforeExecutionGate(t *testing.T) {
	handler := NewServer(engine.NewCatalogue())
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/runs", strings.NewReader(`{"taskId":"go.rate-limiter.001","taskRevision":1}`))
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotImplemented)
	}
	if !strings.Contains(recorder.Body.String(), `"code":"runtime_not_ready"`) {
		t.Fatalf("body = %s", recorder.Body.String())
	}
}
