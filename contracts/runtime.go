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
	ID          string `json:"taskId"`
	Revision    int    `json:"revision"`
	Profile     string `json:"profile"`
	Runtime     string `json:"runtime"`
	Image       string `json:"image"`
	Status      string `json:"status"`
	Network     string `json:"network"`
	HiddenTests bool   `json:"hiddenTests"`
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
