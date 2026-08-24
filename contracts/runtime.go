package contracts

import "time"

const (
	HealthContractVersion      = "fluent-task-runtime.health.v1"
	ProfileContractVersion     = "fluent-task-runtime.profiles.v1"
	TaskContractVersion        = "fluent-task-runtime.tasks.v1"
	TaskSummaryContractVersion = "fluent-task-runtime.task-summary.v1"
	WorkspaceContractVersion   = "fluent-task-runtime.task-workspace.v1"
	RunContractVersion         = "fluent-task-runtime.run.v1"
)

type Health struct {
	ContractVersion string            `json:"contractVersion"`
	Service         string            `json:"service"`
	State           string            `json:"state"`
	Ready           bool              `json:"ready"`
	Dependencies    map[string]string `json:"dependencies"`
	CheckedAt       time.Time         `json:"checkedAt"`
}

type Profile struct {
	ID             string   `json:"id"`
	DisplayName    string   `json:"displayName"`
	Toolchain      string   `json:"toolchain"`
	Image          string   `json:"image"`
	Status         string   `json:"status"`
	Network        string   `json:"network"`
	HiddenTests    bool     `json:"hiddenTests"`
	SupportedTasks []string `json:"supportedTasks"`
}

type Profiles struct {
	ContractVersion string    `json:"contractVersion"`
	Profiles        []Profile `json:"profiles"`
}

// QuestionBinding is the immutable content identity that a task revision was
// authored and tested against. Question Brain owns the referenced content;
// the runtime only carries the identity so a run can be audited later.
type QuestionBinding struct {
	StableKey   string `json:"stableKey"`
	RevisionID  string `json:"revisionId"`
	ContentHash string `json:"contentHash"`
}

// Task is an immutable task-revision descriptor. It is deliberately metadata
// only: starter files and hidden tests stay in the pinned OCI image until the
// sandbox execution gate is closed.
type Task struct {
	ID            string   `json:"taskId"`
	Revision      int      `json:"revision"`
	Profile       string   `json:"profile"`
	Runtime       string   `json:"runtime"`
	Image         string   `json:"image"`
	Status        string   `json:"status"`
	Runnable      bool     `json:"runnable"`
	Network       string   `json:"network"`
	HiddenTests   bool     `json:"hiddenTests"`
	CheckCommand  []string `json:"checkCommand,omitempty"`
	EditableFiles []string `json:"editableFiles,omitempty"`
	TimeoutMS     int      `json:"timeoutMs,omitempty"`
	MemoryMB      int      `json:"memoryMb,omitempty"`
	CPUs          float64  `json:"cpus,omitempty"`
	User          string   `json:"user,omitempty"`
	Artifacts     []string `json:"artifacts,omitempty"`
	// QuestionReleaseID, QuestionBindings and CapabilityKeys are immutable
	// metadata from the executable task revision back to Question Brain. The
	// runtime never owns or mutates question data.
	QuestionReleaseID string            `json:"questionReleaseId,omitempty"`
	QuestionBindings  []QuestionBinding `json:"questionBindings"`
	CapabilityKeys    []string          `json:"capabilityKeys"`
	// QuestionKeys is a deprecated compatibility projection for Lab clients
	// that only understand the first binding contract. New integrations must
	// use QuestionBindings so revision and content identity cannot be lost.
	QuestionKeys []string `json:"questionKeys,omitempty"`
}

type Tasks struct {
	ContractVersion string `json:"contractVersion"`
	Tasks           []Task `json:"tasks"`
}

// TaskSummary is the release-facing part of a task descriptor. It deliberately
// omits executable commands and sandbox paths so callers can inspect release
// joins without receiving execution internals.
type TaskSummary struct {
	TaskID            string            `json:"taskId"`
	Revision          int               `json:"revision"`
	Status            string            `json:"status"`
	Runnable          bool              `json:"runnable"`
	Profile           string            `json:"profile"`
	Runtime           string            `json:"runtime"`
	QuestionReleaseID string            `json:"questionReleaseId,omitempty"`
	QuestionBindings  []QuestionBinding `json:"questionBindings"`
	CapabilityKeys    []string          `json:"capabilityKeys"`
	BindingState      string            `json:"bindingState"`
}

type TaskSummaryResponse struct {
	ContractVersion   string        `json:"contractVersion"`
	BindingState      string        `json:"bindingState"`
	Runnable          bool          `json:"runnable"`
	RuntimeReleaseID  string        `json:"runtimeReleaseId,omitempty"`
	QuestionReleaseID string        `json:"questionReleaseId,omitempty"`
	Tasks             []TaskSummary `json:"tasks"`
}

// TaskWorkspace contains only learner-visible task material. Hidden tests,
// test harnesses, image names and execution commands are intentionally absent.
type TaskWorkspace struct {
	ContractVersion string            `json:"contractVersion"`
	TaskID          string            `json:"taskId"`
	Revision        int               `json:"revision"`
	Status          string            `json:"status"`
	Profile         string            `json:"profile"`
	Runtime         string            `json:"runtime"`
	Brief           string            `json:"brief"`
	EditableFiles   []string          `json:"editableFiles"`
	StarterFiles    map[string]string `json:"starterFiles"`
}

type RuntimeError struct {
	ContractVersion string `json:"contractVersion"`
	Code            string `json:"code"`
	Message         string `json:"message"`
	Retryable       bool   `json:"retryable"`
}

type RunRequest struct {
	TaskID       string            `json:"taskId"`
	TaskRevision int               `json:"taskRevision"`
	Files        map[string]string `json:"files"`
	Locale       string            `json:"locale"`
	Correlation  string            `json:"correlationId"`
}

type TestResult struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
	Output   string `json:"output,omitempty"`
	TestCode string `json:"test_code,omitempty"`
}

type TestResults struct {
	Version int          `json:"version"`
	Status  string       `json:"status"`
	Message string       `json:"message,omitempty"`
	Tests   []TestResult `json:"tests,omitempty"`
}

// RunResponse mirrors the Lab task-run envelope. The nested results document
// stays compatible with the adopted Exercism runner protocol; the runtime
// adds only correlation and bounded process diagnostics around it.
type RunResponse struct {
	ContractVersion string            `json:"contractVersion"`
	Results         TestResults       `json:"results"`
	CorrelationID   string            `json:"correlationId"`
	DurationMS      float64           `json:"durationMs"`
	ExitCode        *int              `json:"exitCode"`
	Stdout          string            `json:"stdout"`
	Stderr          string            `json:"stderr"`
	Artifacts       map[string]string `json:"artifacts,omitempty"`
}
