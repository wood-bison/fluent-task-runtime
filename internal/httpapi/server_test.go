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
	t.Setenv("RUNTIME_RELEASE_MANIFEST", filepath.Join("..", "..", "releases", "task-release-2026-08-24.json"))
	catalogue, err := engine.NewCatalogue()
	if err != nil {
		t.Fatal(err)
	}
	return catalogue
}

func TestHealthAndProfiles(t *testing.T) {
	catalogue := testCatalogue(t)
	handler := NewServer(catalogue)
	for _, path := range []string{"/v1/health/live", "/v1/health/ready", "/v1/profiles", "/v1/tasks", "/v1/tasks/summary", "/v1/tasks/node-event-loop-001/workspace"} {
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

	summaryRecorder := httptest.NewRecorder()
	handler.ServeHTTP(summaryRecorder, httptest.NewRequest(http.MethodGet, "/v1/tasks/summary", nil))
	var summary contracts.TaskSummaryResponse
	if err := json.Unmarshal(summaryRecorder.Body.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.BindingState != "manifest-loaded" || summary.RuntimeReleaseID != "runtime-task-release-2026-08-24" || len(summary.Tasks) != 18 {
		t.Fatalf("unexpected task release summary: %#v", summary)
	}

	workspaceRecorder := httptest.NewRecorder()
	handler.ServeHTTP(workspaceRecorder, httptest.NewRequest(http.MethodGet, "/v1/tasks/node-event-loop-001/workspace?revision=1", nil))
	if workspaceRecorder.Code != http.StatusOK {
		t.Fatalf("workspace status = %d, body = %s", workspaceRecorder.Code, workspaceRecorder.Body.String())
	}
	var workspace contracts.TaskWorkspace
	if err := json.Unmarshal(workspaceRecorder.Body.Bytes(), &workspace); err != nil {
		t.Fatal(err)
	}
	if workspace.ContractVersion != contracts.WorkspaceContractVersion || workspace.Brief == "" || len(workspace.StarterFiles) != 1 {
		t.Fatalf("unexpected task workspace: %#v", workspace)
	}
	for path := range workspace.StarterFiles {
		if strings.HasPrefix(path, "tests/") || strings.Contains(path, "run.js") || strings.Contains(path, "hidden") {
			t.Fatalf("workspace leaked execution internals: %s", path)
		}
	}
}

func TestUnconfiguredManifestIsExplicitlyNonRunnable(t *testing.T) {
	t.Setenv("RUNTIME_TASKS_ROOT", filepath.Join("..", "..", "tasks"))
	t.Setenv("RUNTIME_RELEASE_MANIFEST", "")
	catalogue, err := engine.NewCatalogue()
	if err != nil {
		t.Fatal(err)
	}
	executor := &fakeExecutor{}
	handler := NewServer(catalogue, executor)

	healthRecorder := httptest.NewRecorder()
	handler.ServeHTTP(healthRecorder, httptest.NewRequest(http.MethodGet, "/v1/health/ready", nil))
	if healthRecorder.Code != http.StatusServiceUnavailable || !strings.Contains(healthRecorder.Body.String(), `"ready":false`) || !strings.Contains(healthRecorder.Body.String(), `"state":"degraded"`) {
		t.Fatalf("unconfigured manifest did not fail readiness: %d %s", healthRecorder.Code, healthRecorder.Body.String())
	}

	summaryRecorder := httptest.NewRecorder()
	handler.ServeHTTP(summaryRecorder, httptest.NewRequest(http.MethodGet, "/v1/tasks/summary", nil))
	if summaryRecorder.Code != http.StatusOK || !strings.Contains(summaryRecorder.Body.String(), `"bindingState":"manifest-not-configured"`) || !strings.Contains(summaryRecorder.Body.String(), `"runnable":false`) {
		t.Fatalf("unconfigured manifest was not explicit in summary: %d %s", summaryRecorder.Code, summaryRecorder.Body.String())
	}

	workspaceRecorder := httptest.NewRecorder()
	handler.ServeHTTP(workspaceRecorder, httptest.NewRequest(http.MethodGet, "/v1/tasks/node-event-loop-001/workspace", nil))
	if workspaceRecorder.Code != http.StatusServiceUnavailable || !strings.Contains(workspaceRecorder.Body.String(), `"code":"runtime_not_ready"`) {
		t.Fatalf("unconfigured workspace was not blocked: %d %s", workspaceRecorder.Code, workspaceRecorder.Body.String())
	}

	runRecorder := httptest.NewRecorder()
	handler.ServeHTTP(runRecorder, httptest.NewRequest(http.MethodPost, "/v1/runs", strings.NewReader(`{"taskId":"node-event-loop-001","taskRevision":1,"files":{"index.js":""}}`)))
	if runRecorder.Code != http.StatusServiceUnavailable || !strings.Contains(runRecorder.Body.String(), `"code":"question_release_not_bound"`) || executor.called {
		t.Fatalf("unconfigured run was not blocked: %d %s (called=%v)", runRecorder.Code, runRecorder.Body.String(), executor.called)
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
