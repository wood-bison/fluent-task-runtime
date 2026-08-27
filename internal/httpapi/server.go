package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/wood-bison/fluent-task-runtime/contracts"
	"github.com/wood-bison/fluent-task-runtime/internal/engine"
)

type Server struct {
	catalogue *engine.Catalogue
	executor  engine.RunExecutor
	metrics   *runtimeMetrics
}

// runtimeMetrics is deliberately tiny and cardinality-free. Runtime is an
// execution authority, so Prometheus receives aggregate request/run counters;
// task ids, correlation ids and learner data stay in traces/evidence instead
// of becoming unbounded metric labels.
type runtimeMetrics struct {
	httpRequests atomic.Uint64
	runRequests  atomic.Uint64
	runPasses    atomic.Uint64
	runFailures  atomic.Uint64
	runErrors    atomic.Uint64
}

func NewServer(catalogue *engine.Catalogue, executors ...engine.RunExecutor) http.Handler {
	var executor engine.RunExecutor = engine.NewDockerExecutor(catalogue)
	if len(executors) > 0 && executors[0] != nil {
		executor = executors[0]
	}
	metrics := &runtimeMetrics{}
	server := &Server{catalogue: catalogue, executor: executor, metrics: metrics}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/health/live", server.live)
	mux.HandleFunc("GET /v1/health/ready", server.ready)
	mux.HandleFunc("GET /v1/metrics", metrics.prometheus)
	mux.HandleFunc("GET /v1/profiles", server.profiles)
	mux.HandleFunc("GET /v1/tasks", server.tasks)
	mux.HandleFunc("GET /v1/tasks/summary", server.taskSummary)
	mux.HandleFunc("GET /v1/task-families", server.taskFamilies)
	mux.HandleFunc("GET /v1/task-families/{familyKey}", server.taskFamily)
	mux.HandleFunc("GET /v1/tasks/{taskId}/workspace", server.workspace)
	mux.HandleFunc("POST /v1/runs", server.runs)
	return requestLog(mux, metrics)
}

func (m *runtimeMetrics) prometheus(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("content-type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "# HELP fel_runtime_http_requests_total Total HTTP requests handled by Task Runtime.\n")
	fmt.Fprintf(w, "# TYPE fel_runtime_http_requests_total counter\nfel_runtime_http_requests_total %d\n", m.httpRequests.Load())
	fmt.Fprintf(w, "# HELP fel_runtime_run_requests_total Total task run requests accepted by Task Runtime.\n")
	fmt.Fprintf(w, "# TYPE fel_runtime_run_requests_total counter\nfel_runtime_run_requests_total %d\n", m.runRequests.Load())
	fmt.Fprintf(w, "# HELP fel_runtime_run_results_total Task run results grouped by bounded outcome.\n")
	fmt.Fprintf(w, "# TYPE fel_runtime_run_results_total counter\nfel_runtime_run_results_total{outcome=\"pass\"} %d\n", m.runPasses.Load())
	fmt.Fprintf(w, "fel_runtime_run_results_total{outcome=\"fail\"} %d\n", m.runFailures.Load())
	fmt.Fprintf(w, "fel_runtime_run_results_total{outcome=\"error\"} %d\n", m.runErrors.Load())
}

func (s *Server) live(w http.ResponseWriter, _ *http.Request) {
	identity := s.buildIdentity("")
	writeJSON(w, http.StatusOK, contracts.Health{
		ContractVersion: contracts.HealthContractVersion,
		Service:         "fluent-task-runtime",
		State:           "alive",
		Ready:           true,
		Dependencies:    map[string]string{},
		CheckedAt:       time.Now().UTC(),
		SourceRevision:  identity.SourceRevision,
		ReleaseID:       identity.ReleaseID,
		Environment:     identity.Environment,
	})
}

func (s *Server) ready(w http.ResponseWriter, _ *http.Request) {
	sandbox := "not-configured"
	if provider, ok := s.executor.(interface{ Ready(context.Context) string }); ok {
		sandbox = provider.Ready(context.Background())
	}
	release := s.catalogue.ReleaseSummary()
	questionRelease := release.QuestionReleaseID
	if questionRelease == "" {
		questionRelease = "descriptor-compatibility-only"
	}
	status := http.StatusOK
	state := "ready"
	if !release.Runnable {
		status = http.StatusServiceUnavailable
		state = "degraded"
	}
	identity := s.buildIdentity(release.RuntimeReleaseID)
	writeJSON(w, status, contracts.Health{
		ContractVersion: contracts.HealthContractVersion,
		Service:         "fluent-task-runtime",
		State:           state,
		Ready:           release.Runnable,
		Dependencies: map[string]string{
			"catalogue":        "ready",
			"sandbox":          sandbox,
			"questionBindings": release.BindingState,
			"questionRelease":  questionRelease,
		},
		CheckedAt:      time.Now().UTC(),
		SourceRevision: identity.SourceRevision,
		ReleaseID:      identity.ReleaseID,
		Environment:    identity.Environment,
	})
}

type buildIdentity struct {
	SourceRevision string
	ReleaseID      string
	Environment    string
}

func (s *Server) buildIdentity(releaseID string) buildIdentity {
	sourceRevision := strings.TrimSpace(os.Getenv("SOURCE_REVISION"))
	if sourceRevision == "" {
		sourceRevision = strings.TrimSpace(os.Getenv("TASK_RUNTIME_SOURCE_REVISION"))
	}
	if sourceRevision == "" {
		sourceRevision = "unknown"
	}
	if releaseID == "" {
		releaseID = strings.TrimSpace(os.Getenv("FEL_RELEASE_ID"))
	}
	if releaseID == "" {
		releaseID = "unreleased"
	}
	environment := strings.TrimSpace(os.Getenv("FEL_ENVIRONMENT"))
	if environment == "" {
		environment = strings.TrimSpace(os.Getenv("NODE_ENV"))
	}
	if environment == "" {
		environment = "unknown"
	}
	return buildIdentity{
		SourceRevision: boundedIdentity(sourceRevision),
		ReleaseID:      boundedIdentity(releaseID),
		Environment:    boundedIdentity(environment),
	}
}

func boundedIdentity(value string) string {
	var builder strings.Builder
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || strings.ContainsRune("._:/+-", char) {
			builder.WriteRune(char)
		} else {
			builder.WriteByte('-')
		}
		if builder.Len() >= 79 {
			break
		}
	}
	if builder.Len() == 0 {
		return "unknown"
	}
	return builder.String()
}

func (s *Server) profiles(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.catalogue.Profiles())
}

func (s *Server) tasks(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.catalogue.Tasks())
}

func (s *Server) taskSummary(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.catalogue.ReleaseSummary())
}

func (s *Server) taskFamilies(w http.ResponseWriter, _ *http.Request) {
	families := s.catalogue.TaskFamilies()
	if families.Families == nil {
		families.Families = []contracts.TaskFamily{}
	}
	writeJSON(w, http.StatusOK, families)
}

func (s *Server) taskFamily(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimSpace(r.PathValue("familyKey"))
	if key == "" {
		writeJSON(w, http.StatusBadRequest, contracts.RuntimeError{ContractVersion: contracts.TaskFamilyContractVersion, Code: "invalid_request", Message: "familyKey is required", Retryable: false})
		return
	}
	family, found := s.catalogue.TaskFamily(key)
	if !found {
		writeJSON(w, http.StatusNotFound, contracts.RuntimeError{ContractVersion: contracts.TaskFamilyContractVersion, Code: "unknown_task_family", Message: "task family is not in the runtime release", Retryable: false})
		return
	}
	writeJSON(w, http.StatusOK, contracts.TaskFamilyResponse{ContractVersion: contracts.TaskFamilyContractVersion, ReleaseID: s.catalogue.TaskFamilies().ReleaseID, Family: family})
}

func (s *Server) workspace(w http.ResponseWriter, r *http.Request) {
	taskID := strings.TrimSpace(r.PathValue("taskId"))
	if taskID == "" {
		writeJSON(w, http.StatusBadRequest, contracts.RuntimeError{ContractVersion: contracts.WorkspaceContractVersion, Code: "invalid_request", Message: "taskId is required", Retryable: false})
		return
	}
	if !s.catalogue.ReleaseBound() {
		writeJSON(w, http.StatusServiceUnavailable, contracts.RuntimeError{ContractVersion: contracts.WorkspaceContractVersion, Code: "runtime_not_ready", Message: "task workspace requires an explicit runtime release manifest", Retryable: false})
		return
	}
	var task contracts.Task
	var found bool
	revisionText := strings.TrimSpace(r.URL.Query().Get("revision"))
	if revisionText == "" {
		// A workspace is evidence for an immutable TaskRevision. Selecting the
		// highest revision here would make a deep link silently change meaning
		// after a release, so callers must carry the exact revision tuple.
		writeJSON(w, http.StatusBadRequest, contracts.RuntimeError{ContractVersion: contracts.WorkspaceContractVersion, Code: "revision_required", Message: "an explicit task revision is required", Retryable: false})
		return
	} else {
		revision, err := strconv.Atoi(revisionText)
		if err != nil || revision < 1 {
			writeJSON(w, http.StatusBadRequest, contracts.RuntimeError{ContractVersion: contracts.WorkspaceContractVersion, Code: "invalid_request", Message: "revision must be a positive integer", Retryable: false})
			return
		}
		task, found = s.catalogue.Task(taskID, revision)
	}
	if !found {
		writeJSON(w, http.StatusNotFound, contracts.RuntimeError{ContractVersion: contracts.WorkspaceContractVersion, Code: "unknown_task", Message: "task revision is not in the runtime catalogue", Retryable: false})
		return
	}
	if task.Status != "released" || !task.Runnable || task.Availability != "runnable" {
		writeJSON(w, http.StatusNotImplemented, contracts.RuntimeError{ContractVersion: contracts.WorkspaceContractVersion, Code: "runtime_not_ready", Message: "task workspace is not released for this revision", Retryable: false})
		return
	}
	workspace, err := s.catalogue.TaskWorkspace(task.ID, task.Revision)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, contracts.RuntimeError{ContractVersion: contracts.WorkspaceContractVersion, Code: "workspace_unavailable", Message: "learner workspace could not be loaded", Retryable: false})
		return
	}
	writeJSON(w, http.StatusOK, workspace)
}

func (s *Server) runs(w http.ResponseWriter, r *http.Request) {
	s.metrics.runRequests.Add(1)
	ctx, span := otel.Tracer("fluent-task-runtime/run").Start(r.Context(), "task.run", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()
	r = r.WithContext(ctx)
	var request contracts.RunRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, contracts.RuntimeError{ContractVersion: contracts.RunContractVersion, Code: "invalid_request", Message: "run request must be valid JSON", Retryable: false})
		return
	}
	if strings.TrimSpace(request.TaskID) == "" || request.TaskRevision < 1 {
		writeJSON(w, http.StatusBadRequest, contracts.RuntimeError{ContractVersion: contracts.RunContractVersion, Code: "invalid_request", Message: "taskId and positive taskRevision are required", Retryable: false})
		return
	}
	if !s.catalogue.ReleaseBound() {
		writeJSON(w, http.StatusServiceUnavailable, contracts.RuntimeError{ContractVersion: contracts.RunContractVersion, Code: "question_release_not_bound", Message: "task execution requires an explicit runtime release manifest", Retryable: false})
		return
	}
	span.SetAttributes(
		attribute.String("fluent.task.id", request.TaskID),
		attribute.Int("fluent.task.revision", request.TaskRevision),
		attribute.String("fluent.run.correlation_id", request.Correlation),
	)
	task, found := s.catalogue.Task(request.TaskID, request.TaskRevision)
	if !found {
		writeJSON(w, http.StatusNotFound, contracts.RuntimeError{ContractVersion: contracts.RunContractVersion, Code: "unknown_task", Message: "task revision is not in the runtime catalogue", Retryable: false})
		return
	}
	if task.Status != "released" || !task.Runnable || task.Availability != "runnable" {
		writeJSON(w, http.StatusNotImplemented, contracts.RuntimeError{ContractVersion: contracts.RunContractVersion, Code: "runtime_not_ready", Message: "task execution is not released for this revision", Retryable: false})
		return
	}
	response, err := s.executor.Run(r.Context(), request)
	if err != nil {
		s.metrics.runErrors.Add(1)
		if execution, ok := err.(*engine.ExecutionError); ok {
			status := execution.Status
			if status == 0 {
				status = http.StatusBadGateway
			}
			writeJSON(w, status, contracts.RuntimeError{ContractVersion: contracts.RunContractVersion, Code: execution.Code, Message: execution.Message, Retryable: execution.Retryable})
			return
		}
		writeJSON(w, http.StatusBadGateway, contracts.RuntimeError{ContractVersion: contracts.RunContractVersion, Code: "runner_failed", Message: "task execution failed", Retryable: false})
		return
	}
	span.SetAttributes(attribute.String("fluent.run.result", response.Results.Status))
	if response.Results.Status == "pass" {
		s.metrics.runPasses.Add(1)
	} else {
		s.metrics.runFailures.Add(1)
	}
	writeJSON(w, http.StatusOK, response)
}

func requestLog(next http.Handler, metrics *runtimeMetrics) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metrics.httpRequests.Add(1)
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		ctx, span := otel.Tracer("fluent-task-runtime/http").Start(ctx, r.Method+" "+r.URL.Path, trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()
		started := time.Now()
		requestID := r.Header.Get("x-request-id")
		if requestID == "" {
			requestID = "runtime-local"
		}
		w.Header().Set("x-request-id", requestID)
		span.SetAttributes(
			attribute.String("http.request.method", r.Method),
			attribute.String("url.path", r.URL.Path),
			attribute.String("fluent.request.id", requestID),
		)
		writer := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(writer, r.WithContext(ctx))
		span.SetAttributes(
			attribute.Int("http.response.status_code", writer.statusCode()),
			attribute.Float64("http.server.duration_ms", float64(time.Since(started).Microseconds())/1000),
		)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func (w *statusWriter) statusCode() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
