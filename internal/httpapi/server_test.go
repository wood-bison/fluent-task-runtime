package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wood-bison/fluent-task-runtime/contracts"
	"github.com/wood-bison/fluent-task-runtime/internal/engine"
)

func testCatalogue(t *testing.T) *engine.Catalogue {
	t.Helper()
	t.Setenv("RUNTIME_TASKS_ROOT", filepath.Join("..", "..", "tasks"))
	catalogue, err := engine.NewCatalogue()
	if err != nil {
		t.Fatal(err)
	}
	return catalogue
}

func TestHealthAndProfiles(t *testing.T) {
	catalogue := testCatalogue(t)
	handler := NewServer(catalogue)
	for _, path := range []string{"/v1/health/live", "/v1/health/ready", "/v1/profiles", "/v1/tasks"} {
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
			ID             string   `json:"id"`
			SupportedTasks []string `json:"supportedTasks"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Profiles) != 5 || body.Profiles[0].ID != "node" || body.Profiles[4].ID != "java" {
		t.Fatalf("unexpected profile catalogue: %#v", body.Profiles)
	}
	if len(body.Profiles[0].SupportedTasks) != 12 {
		t.Fatalf("unexpected Node task count: %#v", body.Profiles[0].SupportedTasks)
	}

	tasksRecorder := httptest.NewRecorder()
	handler.ServeHTTP(tasksRecorder, httptest.NewRequest(http.MethodGet, "/v1/tasks", nil))
	if tasksRecorder.Code != http.StatusOK || !strings.Contains(tasksRecorder.Body.String(), `"taskId":"go-rate-limiter-001"`) {
		t.Fatalf("unexpected task catalogue: %d %s", tasksRecorder.Code, tasksRecorder.Body.String())
	}
	if !strings.Contains(tasksRecorder.Body.String(), `"questionReleaseId":"question-release-15e032d7b732f8c1"`) || !strings.Contains(tasksRecorder.Body.String(), `"question.c024"`) {
		t.Fatalf("task catalogue did not expose canonical Question Brain bindings: %s", tasksRecorder.Body.String())
	}
	if !strings.Contains(tasksRecorder.Body.String(), `"questionBindings"`) || !strings.Contains(tasksRecorder.Body.String(), `"capabilityKeys":[]`) {
		t.Fatalf("task catalogue did not expose the versioned binding contract: %s", tasksRecorder.Body.String())
	}
	if body.Profiles[0].SupportedTasks == nil {
		t.Fatal("profile supportedTasks must be an empty array, not null")
	}
}

type fakeExecutor struct {
	called bool
}

func (f *fakeExecutor) Run(_ context.Context, request contracts.RunRequest) (contracts.RunResponse, error) {
	f.called = true
	zero := 0
	return contracts.RunResponse{
		ContractVersion: contracts.RunContractVersion,
		CorrelationID:   request.Correlation,
		ExitCode:        &zero,
		Results: contracts.TestResults{
			Version: 2,
			Status:  "pass",
			Tests:   []contracts.TestResult{{Name: "fake", Status: "pass"}},
		},
	}, nil
}

func TestRunDelegatesReleasedTaskToSandbox(t *testing.T) {
	executor := &fakeExecutor{}
	catalogue := testCatalogue(t)
	handler := NewServer(catalogue, executor)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/runs", strings.NewReader(`{"taskId":"fluent-calculator","taskRevision":1,"files":{"calculator.js":"export function createCalculator(){ return { value: 0 }; }"},"locale":"en","correlationId":"run-123"}`))
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !executor.called {
		t.Fatalf("status = %d, called = %v, body = %s", recorder.Code, executor.called, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"status":"pass"`) {
		t.Fatalf("body = %s", recorder.Body.String())
	}
}
