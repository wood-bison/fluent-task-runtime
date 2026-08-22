package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/wood-bison/fluent-task-runtime/contracts"
	"github.com/wood-bison/fluent-task-runtime/internal/engine"
)

type Server struct {
	catalogue *engine.Catalogue
}

func NewServer(catalogue *engine.Catalogue) http.Handler {
	server := &Server{catalogue: catalogue}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/health/live", server.live)
	mux.HandleFunc("GET /v1/health/ready", server.ready)
	mux.HandleFunc("GET /v1/profiles", server.profiles)
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
	writeJSON(w, http.StatusOK, contracts.Health{
		ContractVersion: contracts.HealthContractVersion,
		Service:         "fluent-task-runtime",
		State:           "ready",
		Ready:           true,
		Dependencies:    map[string]string{"catalogue": "ready", "sandbox": "not-configured"},
		CheckedAt:       time.Now().UTC(),
	})
}

func (s *Server) profiles(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.catalogue.Profiles())
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
	// No execution is exposed before R3. Returning a typed refusal keeps the
	// boundary honest and prevents a UI from interpreting a placeholder as a
	// learner verdict.
	writeJSON(w, http.StatusNotImplemented, contracts.RuntimeError{ContractVersion: contracts.RunContractVersion, Code: "runtime_not_ready", Message: "task execution is not released until the profile and hidden-test harness gate closes", Retryable: false})
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
