package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/wood-bison/fluent-task-runtime/contracts"
)

// Catalogue is the immutable profile catalogue exposed by the runtime. Task
// revisions will be added only after their image and harness digests are
// verified; a profile cannot silently imply that every task is runnable.
type Catalogue struct {
	profiles        contracts.Profiles
	tasks           contracts.Tasks
	tasksRoot       string
	releaseManifest *taskReleaseManifest
}

func NewCatalogue() (*Catalogue, error) {
	tasksRoot := defaultTasksRoot()
	releaseManifest, err := loadReleaseManifest()
	if err != nil {
		return nil, err
	}
	tasks, err := loadTasks(tasksRoot, releaseManifest)
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
	return &Catalogue{tasksRoot: tasksRoot, releaseManifest: releaseManifest, profiles: contracts.Profiles{
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
	TaskID            string                      `json:"taskId"`
	Revision          int                         `json:"revision"`
	Status            string                      `json:"status"`
	Language          string                      `json:"language"`
	Runtime           string                      `json:"runtime"`
	Profile           string                      `json:"profile"`
	Image             string                      `json:"image"`
	CheckCommand      []string                    `json:"checkCommand"`
	EditableFiles     []string                    `json:"editableFiles"`
	DeclaredTests     []string                    `json:"declaredTests"`
	TimeoutMS         int                         `json:"timeoutMs"`
	MemoryMB          int                         `json:"memoryMb"`
	CPUs              float64                     `json:"cpus"`
	User              string                      `json:"user"`
	Artifacts         []string                    `json:"artifacts"`
	QuestionBindings  []contracts.QuestionBinding `json:"questionBindings"`
	QuestionKeys      []string                    `json:"questionKeys"`
	QuestionReleaseID string                      `json:"questionReleaseId"`
	CapabilityKeys    []string                    `json:"capabilityKeys"`
}

const (
	releaseManifestEnv      = "RUNTIME_RELEASE_MANIFEST"
	releaseManifestContract = "fluent-task-runtime.task-release.v1"
)

type taskRevisionKey struct {
	ID       string
	Revision int
}

type taskReleaseEntry struct {
	TaskID            string                      `json:"taskId"`
	Revision          int                         `json:"revision"`
	QuestionReleaseID string                      `json:"questionReleaseId,omitempty"`
	QuestionBindings  []contracts.QuestionBinding `json:"questionBindings"`
	CapabilityKeys    []string                    `json:"capabilityKeys"`
}

type taskReleaseManifest struct {
	ContractVersion   string             `json:"contractVersion"`
	ReleaseID         string             `json:"releaseId"`
	QuestionReleaseID string             `json:"questionReleaseId"`
	Tasks             []taskReleaseEntry `json:"tasks"`
	entries           map[taskRevisionKey]taskReleaseEntry
}

func loadTasks(root string, releaseManifest *taskReleaseManifest) ([]contracts.Task, error) {
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
	seenManifestEntries := make(map[taskRevisionKey]struct{})
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
		questionReleaseID := descriptor.QuestionReleaseID
		questionBindings := append([]contracts.QuestionBinding(nil), descriptor.QuestionBindings...)
		questionKeys := append([]string(nil), descriptor.QuestionKeys...)
		capabilityKeys := append([]string(nil), descriptor.CapabilityKeys...)
		if releaseManifest != nil {
			key := taskRevisionKey{ID: id, Revision: descriptor.Revision}
			entry, found := releaseManifest.entries[key]
			if status == "released" && !found {
				return nil, fmt.Errorf("released task descriptor %q is missing from release manifest %q", path, releaseManifest.ReleaseID)
			}
			if found {
				seenManifestEntries[key] = struct{}{}
				var mergeErr error
				questionReleaseID, questionBindings, questionKeys, capabilityKeys, mergeErr = mergeReleaseEntry(
					questionReleaseID, questionBindings, questionKeys, capabilityKeys, releaseManifest.QuestionReleaseID, entry,
				)
				if mergeErr != nil {
					return nil, fmt.Errorf("task descriptor %q: %w", path, mergeErr)
				}
			}
		}
		questionBindings, questionKeys, capabilityKeys, bindingErr := validateQuestionBinding(
			questionBindings, questionKeys, capabilityKeys, questionReleaseID, status,
		)
		if bindingErr != nil {
			return nil, fmt.Errorf("task descriptor %q: %w", path, bindingErr)
		}
		tasks = append(tasks, contracts.Task{
			ID: id, Revision: revision, Language: descriptor.Language, Profile: descriptor.Profile, Runtime: descriptor.Runtime,
			Image: descriptor.Image, Status: status, Runnable: releaseManifest != nil && status == "released", Network: "none", HiddenTests: true,
			CheckCommand:  append([]string(nil), descriptor.CheckCommand...),
			EditableFiles: append([]string(nil), descriptor.EditableFiles...),
			DeclaredTests: append([]string(nil), descriptor.DeclaredTests...),
			TimeoutMS:     descriptor.TimeoutMS, MemoryMB: descriptor.MemoryMB, CPUs: descriptor.CPUs, User: descriptor.User,
			Artifacts:         append([]string(nil), descriptor.Artifacts...),
			QuestionBindings:  questionBindings,
			QuestionKeys:      questionKeys,
			QuestionReleaseID: questionReleaseID,
			CapabilityKeys:    capabilityKeys,
		})
	}
	if releaseManifest != nil {
		for key := range releaseManifest.entries {
			if _, found := seenManifestEntries[key]; !found {
				return nil, fmt.Errorf("release manifest %q references task %s@%d which is not in the catalogue", releaseManifest.ReleaseID, key.ID, key.Revision)
			}
		}
	}
	if len(tasks) == 0 {
		return nil, fmt.Errorf("task catalogue %q is empty", root)
	}
	return tasks, nil
}

func loadReleaseManifest() (*taskReleaseManifest, error) {
	path := strings.TrimSpace(os.Getenv(releaseManifestEnv))
	if path == "" {
		return nil, nil
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read release manifest %q: %w", path, err)
	}
	var manifest taskReleaseManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, fmt.Errorf("decode release manifest %q: %w", path, err)
	}
	if manifest.ContractVersion != releaseManifestContract {
		return nil, fmt.Errorf("release manifest %q has unsupported contractVersion %q", path, manifest.ContractVersion)
	}
	if strings.TrimSpace(manifest.ReleaseID) == "" {
		return nil, fmt.Errorf("release manifest %q must declare releaseId", path)
	}
	if !isQuestionReleaseID(manifest.QuestionReleaseID) {
		return nil, fmt.Errorf("release manifest %q must declare a valid questionReleaseId", path)
	}
	manifest.entries = make(map[taskRevisionKey]taskReleaseEntry, len(manifest.Tasks))
	for index, entry := range manifest.Tasks {
		entry.TaskID = strings.TrimSpace(entry.TaskID)
		if entry.TaskID == "" || entry.Revision < 1 {
			return nil, fmt.Errorf("release manifest %q entry %d has invalid task identity", path, index)
		}
		key := taskRevisionKey{ID: entry.TaskID, Revision: entry.Revision}
		if _, exists := manifest.entries[key]; exists {
			return nil, fmt.Errorf("release manifest %q contains duplicate task %s@%d", path, entry.TaskID, entry.Revision)
		}
		if entry.QuestionReleaseID != "" && entry.QuestionReleaseID != manifest.QuestionReleaseID {
			return nil, fmt.Errorf("release manifest %q entry %s@%d pins question release %q instead of %q", path, entry.TaskID, entry.Revision, entry.QuestionReleaseID, manifest.QuestionReleaseID)
		}
		if _, _, _, err := validateQuestionBinding(entry.QuestionBindings, nil, entry.CapabilityKeys, manifest.QuestionReleaseID, "released"); err != nil {
			return nil, fmt.Errorf("release manifest %q entry %s@%d: %w", path, entry.TaskID, entry.Revision, err)
		}
		manifest.entries[key] = entry
	}
	return &manifest, nil
}

func mergeReleaseEntry(
	questionReleaseID string,
	questionBindings []contracts.QuestionBinding,
	questionKeys, capabilityKeys []string,
	manifestQuestionReleaseID string,
	entry taskReleaseEntry,
) (string, []contracts.QuestionBinding, []string, []string, error) {
	// A release manifest is the immutable source of truth for the active
	// release. Legacy descriptors retain the release that produced them, so
	// their old questionReleaseId/questionKeys must not prevent generating a
	// new overlay. A descriptor that already carries full immutable bindings is
	// different: it is an integrity assertion and must agree with the manifest.
	descriptorHasBindings := len(questionBindings) > 0
	if descriptorHasBindings && !sameQuestionBindings(questionBindings, entry.QuestionBindings) {
		return "", nil, nil, nil, errors.New("task descriptor questionBindings disagree with immutable release manifest")
	}
	if !descriptorHasBindings {
		questionBindings = append([]contracts.QuestionBinding(nil), entry.QuestionBindings...)
	}
	if descriptorHasBindings && questionReleaseID != "" && questionReleaseID != manifestQuestionReleaseID {
		return "", nil, nil, nil, fmt.Errorf("task descriptor pins question release %q but immutable bindings belong to %q", questionReleaseID, manifestQuestionReleaseID)
	}
	questionReleaseID = manifestQuestionReleaseID
	capabilityKeys = append([]string(nil), entry.CapabilityKeys...)
	questionKeys = manifestCompatibilityKeys(entry)
	return questionReleaseID, questionBindings, questionKeys, capabilityKeys, nil
}

func manifestCompatibilityKeys(entry taskReleaseEntry) []string {
	keys := questionKeysFromBindings(entry.QuestionBindings)
	keys = append(keys, entry.CapabilityKeys...)
	return keys
}

func validateQuestionBinding(
	bindings []contracts.QuestionBinding,
	legacyKeys, capabilityKeys []string,
	releaseID, status string,
) ([]contracts.QuestionBinding, []string, []string, error) {
	if status == "released" && !isQuestionReleaseID(releaseID) {
		return nil, nil, nil, fmt.Errorf("released task must declare a pinned questionReleaseId")
	}
	validatedBindings := make([]contracts.QuestionBinding, 0, len(bindings))
	seenStableKeys := make(map[string]struct{}, len(bindings))
	seenRevisionIDs := make(map[string]struct{}, len(bindings))
	for _, binding := range bindings {
		binding.StableKey = strings.TrimSpace(binding.StableKey)
		binding.RevisionID = strings.TrimSpace(binding.RevisionID)
		binding.ContentHash = strings.TrimSpace(binding.ContentHash)
		if !isStableQuestionKey(binding.StableKey) {
			return nil, nil, nil, fmt.Errorf("invalid question binding stableKey %q; expected question.<key>", binding.StableKey)
		}
		if !isRevisionID(binding.RevisionID) {
			return nil, nil, nil, fmt.Errorf("invalid question binding revisionId for %q", binding.StableKey)
		}
		if !isContentHash(binding.ContentHash) {
			return nil, nil, nil, fmt.Errorf("invalid question binding contentHash for %q", binding.StableKey)
		}
		if _, exists := seenStableKeys[binding.StableKey]; exists {
			return nil, nil, nil, fmt.Errorf("duplicate question binding stableKey %q", binding.StableKey)
		}
		if _, exists := seenRevisionIDs[binding.RevisionID]; exists {
			return nil, nil, nil, fmt.Errorf("duplicate question binding revisionId %q", binding.RevisionID)
		}
		seenStableKeys[binding.StableKey] = struct{}{}
		seenRevisionIDs[binding.RevisionID] = struct{}{}
		validatedBindings = append(validatedBindings, binding)
	}
	validatedKeys := make([]string, 0, len(legacyKeys)+len(validatedBindings))
	seenLegacyKeys := make(map[string]struct{}, len(legacyKeys)+len(validatedBindings))
	for _, key := range legacyKeys {
		key = strings.TrimSpace(key)
		if !isStableBindingKey(key) {
			return nil, nil, nil, fmt.Errorf("invalid legacy question binding %q; expected question.<key> or capability.<key>", key)
		}
		if _, exists := seenLegacyKeys[key]; exists {
			return nil, nil, nil, fmt.Errorf("duplicate legacy question binding %q", key)
		}
		if strings.HasPrefix(key, "question.") || strings.HasPrefix(key, "capability.") {
			seenLegacyKeys[key] = struct{}{}
		}
		validatedKeys = append(validatedKeys, key)
	}
	if len(validatedKeys) == 0 && len(validatedBindings) > 0 {
		validatedKeys = questionKeysFromBindings(validatedBindings)
	}
	validatedCapabilities, err := validateCapabilityKeys(capabilityKeys)
	if err != nil {
		return nil, nil, nil, err
	}
	for _, key := range validatedKeys {
		if strings.HasPrefix(key, "capability.") && !containsString(validatedCapabilities, key) {
			validatedCapabilities = append(validatedCapabilities, key)
		}
	}
	if status == "released" && len(validatedKeys) == 0 && len(validatedBindings) == 0 && len(validatedCapabilities) == 0 {
		return nil, nil, nil, fmt.Errorf("released task must declare at least one question binding or capability key")
	}
	return validatedBindings, validatedKeys, validatedCapabilities, nil
}

var (
	revisionIDPattern  = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	contentHashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
	keyPartPattern     = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,79}$`)
)

func isStableBindingKey(key string) bool {
	return isStableQuestionKey(key) || isStableCapabilityKey(key)
}

func isStableQuestionKey(key string) bool {
	parts := strings.SplitN(strings.TrimSpace(key), ".", 2)
	return len(parts) == 2 && parts[0] == "question" && keyPartPattern.MatchString(parts[1])
}

func isStableCapabilityKey(key string) bool {
	parts := strings.Split(strings.TrimSpace(key), ".")
	if len(parts) < 2 || len(parts) > 3 || parts[0] != "capability" {
		return false
	}
	for _, part := range parts[1:] {
		if !keyPartPattern.MatchString(part) {
			return false
		}
	}
	return true
}

func isRevisionID(value string) bool {
	return revisionIDPattern.MatchString(value)
}

func isContentHash(value string) bool {
	return contentHashPattern.MatchString(value)
}

func validateCapabilityKeys(keys []string) ([]string, error) {
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if !isStableCapabilityKey(key) {
			return nil, fmt.Errorf("invalid capability key %q; expected capability.<slug> or capability.<domain>.<slug>", key)
		}
		if containsString(result, key) {
			return nil, fmt.Errorf("duplicate capability key %q", key)
		}
		result = append(result, key)
	}
	return result, nil
}

func questionKeysFromBindings(bindings []contracts.QuestionBinding) []string {
	result := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		result = append(result, binding.StableKey)
	}
	return result
}

func sameQuestionBindings(left, right []contracts.QuestionBinding) bool {
	if len(left) != len(right) {
		return false
	}
	for _, binding := range left {
		found := false
		for _, other := range right {
			if binding == other {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
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
		result.Tasks[index] = cloneTask(c.tasks.Tasks[index])
	}
	return result
}

func (c *Catalogue) Task(id string, revision int) (contracts.Task, bool) {
	for _, task := range c.tasks.Tasks {
		if task.ID == id && task.Revision == revision {
			return cloneTask(task), true
		}
	}
	return contracts.Task{}, false
}

// LatestTask returns the highest revision for a task ID. Callers may still
// request an exact revision through Task when reproducing an old run; the
// HTTP layer decides whether a declared revision is ready for learners.
func (c *Catalogue) LatestTask(id string) (contracts.Task, bool) {
	var latest contracts.Task
	found := false
	for _, task := range c.tasks.Tasks {
		if task.ID != id || (found && task.Revision <= latest.Revision) {
			continue
		}
		latest = task
		found = true
	}
	if !found {
		return contracts.Task{}, false
	}
	return cloneTask(latest), true
}

// ReleaseBound reports whether an explicit, validated runtime release
// manifest selected this catalogue. Descriptor-only compatibility data is
// intentionally non-runnable.
func (c *Catalogue) ReleaseBound() bool {
	return c.releaseManifest != nil
}

// ReleaseSummary returns the release join without exposing task execution
// internals. A missing manifest is explicit: descriptors may still be loaded
// through the legacy compatibility path, but the catalogue is not a fully
// bound Question Brain release.
func (c *Catalogue) ReleaseSummary() contracts.TaskSummaryResponse {
	state := "manifest-not-configured"
	runtimeReleaseID := ""
	questionReleaseID := ""
	if c.releaseManifest != nil {
		state = "manifest-loaded"
		runtimeReleaseID = c.releaseManifest.ReleaseID
		questionReleaseID = c.releaseManifest.QuestionReleaseID
	}
	result := contracts.TaskSummaryResponse{
		ContractVersion:   contracts.TaskSummaryContractVersion,
		BindingState:      state,
		Runnable:          c.releaseManifest != nil,
		RuntimeReleaseID:  runtimeReleaseID,
		QuestionReleaseID: questionReleaseID,
		Tasks:             make([]contracts.TaskSummary, 0, len(c.tasks.Tasks)),
	}
	for _, task := range c.tasks.Tasks {
		bindingState := "unbound"
		if len(task.QuestionBindings) > 0 {
			bindingState = "full"
		} else if len(task.CapabilityKeys) > 0 {
			bindingState = "capability-only"
		} else if len(task.QuestionKeys) > 0 {
			bindingState = "legacy"
		}
		result.Tasks = append(result.Tasks, contracts.TaskSummary{
			TaskID:            task.ID,
			Revision:          task.Revision,
			Status:            task.Status,
			Runnable:          task.Runnable,
			Profile:           task.Profile,
			Runtime:           task.Runtime,
			QuestionReleaseID: task.QuestionReleaseID,
			QuestionBindings:  cloneQuestionBindings(task.QuestionBindings),
			CapabilityKeys:    cloneStrings(task.CapabilityKeys),
			BindingState:      bindingState,
		})
	}
	return result
}

const (
	workspaceBriefLimit = 256 * 1024
	workspaceFileLimit  = 256 * 1024
	workspaceTotalLimit = 2 * 1024 * 1024
)

// TaskWorkspace reads only learner-visible material from a released task.
// The path is rooted at the catalogue entry and never traverses tests or
// harness directories.
func (c *Catalogue) TaskWorkspace(id string, revision int) (contracts.TaskWorkspace, error) {
	if !c.ReleaseBound() {
		return contracts.TaskWorkspace{}, errors.New("an explicit runtime release manifest is required")
	}
	task, found := c.Task(id, revision)
	if !found {
		return contracts.TaskWorkspace{}, fmt.Errorf("task %q revision %d is not in the runtime catalogue", id, revision)
	}
	if task.Status != "released" {
		return contracts.TaskWorkspace{}, fmt.Errorf("task %q revision %d is not released", id, revision)
	}
	taskRoot := filepath.Join(c.tasksRoot, task.ID)
	brief, err := readWorkspaceBounded(filepath.Join(taskRoot, "brief.md"), workspaceBriefLimit)
	if err != nil {
		return contracts.TaskWorkspace{}, fmt.Errorf("read task brief: %w", err)
	}
	starterRoot := filepath.Join(taskRoot, "starter")
	starterFiles := make(map[string]string)
	totalBytes := 0
	if err := filepath.WalkDir(starterRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("task starter contains a symlink")
		}
		relative, err := filepath.Rel(starterRoot, path)
		if err != nil || !safeRelativePath(filepath.ToSlash(relative)) {
			return errors.New("task starter contains an unsafe path")
		}
		body, err := readWorkspaceBounded(path, workspaceFileLimit)
		if err != nil {
			return err
		}
		totalBytes += len(body)
		if totalBytes > workspaceTotalLimit {
			return errors.New("task starter exceeds the workspace size limit")
		}
		starterFiles[filepath.ToSlash(relative)] = string(body)
		return nil
	}); err != nil {
		return contracts.TaskWorkspace{}, fmt.Errorf("read task starter: %w", err)
	}
	return contracts.TaskWorkspace{
		ContractVersion:   contracts.WorkspaceContractVersion,
		TaskID:            task.ID,
		Revision:          task.Revision,
		Status:            task.Status,
		Language:          task.Language,
		Profile:           task.Profile,
		Runtime:           task.Runtime,
		Brief:             string(brief),
		DeclaredTestCount: len(task.DeclaredTests),
		EditableFiles:     append([]string(nil), task.EditableFiles...),
		StarterFiles:      starterFiles,
	}, nil
}

func readWorkspaceBounded(path string, limit int) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("workspace file is not a regular file")
	}
	return readBounded(path, limit)
}

func cloneTask(task contracts.Task) contracts.Task {
	task.CheckCommand = cloneStrings(task.CheckCommand)
	task.EditableFiles = cloneStrings(task.EditableFiles)
	task.DeclaredTests = cloneStrings(task.DeclaredTests)
	task.Artifacts = cloneStrings(task.Artifacts)
	task.QuestionBindings = cloneQuestionBindings(task.QuestionBindings)
	task.QuestionKeys = cloneStrings(task.QuestionKeys)
	task.CapabilityKeys = cloneStrings(task.CapabilityKeys)
	return task
}

func cloneStrings(values []string) []string {
	if values == nil {
		return nil
	}
	result := make([]string, len(values))
	copy(result, values)
	return result
}

func cloneQuestionBindings(values []contracts.QuestionBinding) []contracts.QuestionBinding {
	if values == nil {
		return nil
	}
	result := make([]contracts.QuestionBinding, len(values))
	copy(result, values)
	return result
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
