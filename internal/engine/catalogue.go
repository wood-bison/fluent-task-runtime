package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wood-bison/fluent-task-runtime/contracts"
)

// Catalogue is the immutable profile catalogue exposed by the runtime. Task
// revisions will be added only after their image and harness digests are
// verified; a profile cannot silently imply that every task is runnable.
type Catalogue struct {
	profiles  contracts.Profiles
	tasks     contracts.Tasks
	tasksRoot string
}

func NewCatalogue() (*Catalogue, error) {
	tasksRoot := defaultTasksRoot()
	tasks, err := loadTasks(tasksRoot)
	if err != nil {
		return nil, err
	}
	profiles := []contracts.Profile{
		{ID: "node", DisplayName: "Node.js", Toolchain: "Node.js 24", Image: "fluent-runtime-task-node:1", Status: "declared", Network: "none", HiddenTests: true},
		{ID: "dotnet", DisplayName: ".NET", Toolchain: ".NET 10", Image: "fluent-runtime-task-dotnet:1", Status: "declared", Network: "none", HiddenTests: true},
		{ID: "postgres", DisplayName: "PostgreSQL", Toolchain: "PostgreSQL 17", Image: "fluent-runtime-task-postgres:1", Status: "declared", Network: "none", HiddenTests: true},
		{ID: "go", DisplayName: "Go", Toolchain: "Go 1.24", Image: "fluent-runtime-task-go:1", Status: "declared", Network: "none", HiddenTests: true},
		{ID: "java", DisplayName: "Java", Toolchain: "JDK 21", Image: "fluent-runtime-task-java:1", Status: "declared", Network: "none", HiddenTests: true},
	}
	for index := range profiles {
		for _, task := range tasks {
			if task.Profile == profiles[index].ID {
				profiles[index].SupportedTasks = append(profiles[index].SupportedTasks, task.ID)
			}
		}
		for _, task := range tasks {
			if task.Profile == profiles[index].ID && task.Status == "released" {
				profiles[index].Status = "ready"
				break
			}
		}
	}
	return &Catalogue{tasksRoot: tasksRoot, profiles: contracts.Profiles{
		ContractVersion: contracts.ProfileContractVersion,
		Profiles:        profiles,
	}, tasks: contracts.Tasks{ContractVersion: contracts.TaskContractVersion, Tasks: tasks}}, nil
}

func defaultTasksRoot() string {
	if configured := os.Getenv("RUNTIME_TASKS_ROOT"); configured != "" {
		return configured
	}
	return "tasks"
}

type taskDescriptor struct {
	TaskID            string   `json:"taskId"`
	Revision          int      `json:"revision"`
	Status            string   `json:"status"`
	Runtime           string   `json:"runtime"`
	Profile           string   `json:"profile"`
	Image             string   `json:"image"`
	CheckCommand      []string `json:"checkCommand"`
	EditableFiles     []string `json:"editableFiles"`
	TimeoutMS         int      `json:"timeoutMs"`
	MemoryMB          int      `json:"memoryMb"`
	CPUs              float64  `json:"cpus"`
	User              string   `json:"user"`
	Artifacts         []string `json:"artifacts"`
	QuestionKeys      []string `json:"questionKeys"`
	QuestionReleaseID string   `json:"questionReleaseId"`
}

func loadTasks(root string) ([]contracts.Task, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read task catalogue %q: %w", root, err)
	}
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			ids = append(ids, entry.Name())
		}
	}
	sort.Strings(ids)
	tasks := make([]contracts.Task, 0, len(ids))
	for _, id := range ids {
		path := filepath.Join(root, id, "task.json")
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("read task descriptor %q: %w", path, readErr)
		}
		var descriptor taskDescriptor
		if decodeErr := json.Unmarshal(body, &descriptor); decodeErr != nil || descriptor.TaskID != id {
			if decodeErr != nil {
				return nil, fmt.Errorf("decode task descriptor %q: %w", path, decodeErr)
			}
			return nil, fmt.Errorf("task descriptor %q has taskId %q", path, descriptor.TaskID)
		}
		if descriptor.Revision < 1 {
			return nil, fmt.Errorf("task descriptor %q has invalid revision %d", path, descriptor.Revision)
		}
		if descriptor.TimeoutMS <= 0 || descriptor.MemoryMB <= 0 || descriptor.CPUs <= 0 {
			return nil, fmt.Errorf("task descriptor %q must declare positive timeout, memory, and cpu limits", path)
		}
		revision := descriptor.Revision
		status := "declared"
		if strings.EqualFold(strings.TrimSpace(descriptor.Status), "released") {
			status = "released"
		}
		questionKeys, bindingErr := validateQuestionBinding(descriptor.QuestionKeys, descriptor.QuestionReleaseID, status)
		if bindingErr != nil {
			return nil, fmt.Errorf("task descriptor %q: %w", path, bindingErr)
		}
		tasks = append(tasks, contracts.Task{
			ID: id, Revision: revision, Profile: descriptor.Profile, Runtime: descriptor.Runtime,
			Image: descriptor.Image, Status: status, Network: "none", HiddenTests: true,
			CheckCommand:  append([]string(nil), descriptor.CheckCommand...),
			EditableFiles: append([]string(nil), descriptor.EditableFiles...),
			TimeoutMS:     descriptor.TimeoutMS, MemoryMB: descriptor.MemoryMB, CPUs: descriptor.CPUs, User: descriptor.User,
			Artifacts:    append([]string(nil), descriptor.Artifacts...),
			QuestionKeys: questionKeys, QuestionReleaseID: descriptor.QuestionReleaseID,
		})
	}
	if len(tasks) == 0 {
		return nil, fmt.Errorf("task catalogue %q is empty", root)
	}
	return tasks, nil
}

func validateQuestionBinding(keys []string, releaseID, status string) ([]string, error) {
	if status == "released" && len(keys) == 0 {
		return nil, fmt.Errorf("released task must declare at least one questionKeys or capability binding")
	}
	if status == "released" && !isQuestionReleaseID(releaseID) {
		return nil, fmt.Errorf("released task must declare a pinned questionReleaseId")
	}
	seen := make(map[string]struct{}, len(keys))
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if !isStableBindingKey(key) {
			return nil, fmt.Errorf("invalid question binding %q; expected question.<key> or capability.<key>", key)
		}
		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf("duplicate question binding %q", key)
		}
		seen[key] = struct{}{}
		result = append(result, key)
	}
	return result, nil
}

func isStableBindingKey(key string) bool {
	parts := strings.SplitN(key, ".", 2)
	if len(parts) != 2 || (parts[0] != "question" && parts[0] != "capability") {
		return false
	}
	if len(parts[1]) < 2 || len(parts[1]) > 80 {
		return false
	}
	for _, character := range parts[1] {
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '-' {
			return false
		}
	}
	return true
}

func isQuestionReleaseID(releaseID string) bool {
	if !strings.HasPrefix(releaseID, "question-release-") || len(releaseID) != len("question-release-")+16 {
		return false
	}
	for _, character := range releaseID[len("question-release-"):] {
		if !(character >= '0' && character <= '9') && !(character >= 'a' && character <= 'f') {
			return false
		}
	}
	return true
}

func (c *Catalogue) Tasks() contracts.Tasks {
	result := contracts.Tasks{ContractVersion: c.tasks.ContractVersion, Tasks: make([]contracts.Task, len(c.tasks.Tasks))}
	copy(result.Tasks, c.tasks.Tasks)
	for index := range result.Tasks {
		result.Tasks[index].CheckCommand = append([]string(nil), c.tasks.Tasks[index].CheckCommand...)
		result.Tasks[index].EditableFiles = append([]string(nil), c.tasks.Tasks[index].EditableFiles...)
		result.Tasks[index].Artifacts = append([]string(nil), c.tasks.Tasks[index].Artifacts...)
		result.Tasks[index].QuestionKeys = append([]string(nil), c.tasks.Tasks[index].QuestionKeys...)
	}
	return result
}

func (c *Catalogue) Task(id string, revision int) (contracts.Task, bool) {
	for _, task := range c.tasks.Tasks {
		if task.ID == id && task.Revision == revision {
			return task, true
		}
	}
	return contracts.Task{}, false
}

func (c *Catalogue) TasksRoot() string {
	return c.tasksRoot
}

func (c *Catalogue) Profiles() contracts.Profiles {
	result := contracts.Profiles{ContractVersion: c.profiles.ContractVersion, Profiles: make([]contracts.Profile, len(c.profiles.Profiles))}
	copy(result.Profiles, c.profiles.Profiles)
	for index := range result.Profiles {
		tasks := result.Profiles[index].SupportedTasks
		result.Profiles[index].SupportedTasks = make([]string, len(tasks))
		copy(result.Profiles[index].SupportedTasks, tasks)
	}
	return result
}
