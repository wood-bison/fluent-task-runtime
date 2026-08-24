package contracts

import "time"

const (
	HealthContractVersion  = "fluent-task-runtime.health.v1"
	ProfileContractVersion = "fluent-task-runtime.profiles.v1"
	TaskContractVersion    = "fluent-task-runtime.tasks.v1"
	RunContractVersion     = "fluent-task-runtime.run.v1"
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
	Network       string   `json:"network"`
	HiddenTests   bool     `json:"hiddenTests"`
	CheckCommand  []string `json:"checkCommand,omitempty"`
	EditableFiles []string `json:"editableFiles,omitempty"`
	TimeoutMS     int      `json:"timeoutMs,omitempty"`
	MemoryMB      int      `json:"memoryMb,omitempty"`
	CPUs          float64  `json:"cpus,omitempty"`
	User          string   `json:"user,omitempty"`
	Artifacts     []string `json:"artifacts,omitempty"`
	// QuestionKeys and QuestionReleaseID are the immutable binding from an
	// executable task revision back to the canonical Question Brain release.
	// They are metadata only: the runtime never owns or mutates question data.
	QuestionKeys      []string `json:"questionKeys,omitempty"`
	QuestionReleaseID string   `json:"questionReleaseId,omitempty"`
}

type Tasks struct {
	ContractVersion string `json:"contractVersion"`
	Tasks           []Task `json:"tasks"`
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
