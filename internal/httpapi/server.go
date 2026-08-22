package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/wood-bison/fluent-task-runtime/contracts"
	"github.com/wood-bison/fluent-task-runtime/internal/engine"
)

type Server struct {
	catalogue *engine.Catalogue
	executor  engine.RunExecutor
}

func NewServer(catalogue *engine.Catalogue, executors ...engine.RunExecutor) http.Handler {
	var executor engine.RunExecutor = engine.NewDockerExecutor(catalogue)
	if len(executors) > 0 && executors[0] != nil {
		executor = executors[0]
	}
	server := &Server{catalogue: catalogue, executor: executor}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/health/live", server.live)
	mux.HandleFunc("GET /v1/health/ready", server.ready)
	mux.HandleFunc("GET /v1/profiles", server.profiles)
	mux.HandleFunc("GET /v1/tasks", server.tasks)
	mux.HandleFunc("POST /v1/runs", server.runs)
	return requestLog(mux)
}

func (s *Server) live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, contracts.Health{
		ContractVersion: contracts.HealthContractVersion,
		Service:         "fluent-task-runtime",
		State:           "alive",
		Ready:           true,
		Dependencies:    map[string]string{},
		CheckedAt:       time.Now().UTC(),
	})
}

func (s *Server) ready(w http.ResponseWriter, _ *http.Request) {
	sandbox := "not-configured"
	if provider, ok := s.executor.(interface{ Ready(context.Context) string }); ok {
		sandbox = provider.Ready(context.Background())
	}
	writeJSON(w, http.StatusOK, contracts.Health{
		ContractVersion: contracts.HealthContractVersion,
		Service:         "fluent-task-runtime",
		State:           "ready",
		Ready:           true,
		Dependencies:    map[string]string{"catalogue": "ready", "sandbox": sandbox},
		CheckedAt:       time.Now().UTC(),
	})
}

func (s *Server) profiles(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.catalogue.Profiles())
}

func (s *Server) tasks(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.catalogue.Tasks())
}

func (s *Server) runs(w http.ResponseWriter, r *http.Request) {
	var request contracts.RunRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, contracts.RuntimeError{ContractVersion: contracts.RunContractVersion, Code: "invalid_request", Message: "run request must be valid JSON", Retryable: false})
		return
	}
	if strings.TrimSpace(request.TaskID) == "" || request.TaskRevision < 1 {
		writeJSON(w, http.StatusBadRequest, contracts.RuntimeError{ContractVersion: contracts.RunContractVersion, Code: "invalid_request", Message: "taskId and positive taskRevision are required", Retryable: false})
		return
	}
	task, found := s.catalogue.Task(request.TaskID, request.TaskRevision)
	if !found {
		writeJSON(w, http.StatusNotFound, contracts.RuntimeError{ContractVersion: contracts.RunContractVersion, Code: "unknown_task", Message: "task revision is not in the runtime catalogue", Retryable: false})
		return
	}
	if task.Status != "released" {
		writeJSON(w, http.StatusNotImplemented, contracts.RuntimeError{ContractVersion: contracts.RunContractVersion, Code: "runtime_not_ready", Message: "task execution is not released for this revision", Retryable: false})
		return
	}
	response, err := s.executor.Run(r.Context(), request)
	if err != nil {
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
	writeJSON(w, http.StatusOK, response)
}

func requestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		requestID := r.Header.Get("x-request-id")
		if requestID == "" {
			requestID = "runtime-local"
		}
		w.Header().Set("x-request-id", requestID)
		next.ServeHTTP(w, r)
		_ = started // structured tracing is added with the sandbox worker gate
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
