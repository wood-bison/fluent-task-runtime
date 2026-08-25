package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wood-bison/fluent-task-runtime/contracts"
)

func TestCatalogueHonoursTaskReleaseMetadata(t *testing.T) {
	root := t.TempDir()
	writeTask := func(id, status, user string) {
		t.Helper()
		directory := filepath.Join(root, id)
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
		binding := ""
		if status == "released" {
			binding = `,"questionBindings":[{"stableKey":"question.q777","revisionId":"11111111-1111-4111-8111-111111111111","contentHash":"` + strings.Repeat("a", 64) + `"}],"questionReleaseId":"question-release-15e032d7b732f8c1"`
		}
		body := `{"taskId":"` + id + `","revision":7,"status":"` + status + `","profile":"node","runtime":"Node.js 24","image":"fluent-runtime-task-node:1","checkCommand":["node"],"editableFiles":["main.js"],"timeoutMs":20000,"memoryMb":512,"cpus":1,"user":"` + user + `","declaredTests":["main task"]` + binding + `}`
		if err := os.WriteFile(filepath.Join(directory, "task.json"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeTask("released-task", "released", "1000:1000")
	writeTask("declared-task", "declared", "")
	t.Setenv("RUNTIME_TASKS_ROOT", root)

	catalogue, err := NewCatalogue()
	if err != nil {
		t.Fatal(err)
	}
	released, ok := catalogue.Task("released-task", 7)
	if !ok || released.Status != "released" || released.User != "1000:1000" || released.QuestionReleaseID != "question-release-15e032d7b732f8c1" || len(released.QuestionBindings) != 1 || released.QuestionBindings[0].StableKey != "question.q777" {
		t.Fatalf("released descriptor was not preserved: %#v (ok=%v)", released, ok)
	}
	declared, ok := catalogue.Task("declared-task", 7)
	if !ok || declared.Status != "declared" {
		t.Fatalf("declared descriptor was not preserved: %#v (ok=%v)", declared, ok)
	}
	if summary := catalogue.ReleaseSummary(); summary.BindingState != "manifest-not-configured" || summary.Runnable || len(summary.Tasks) != 2 || summary.Tasks[0].Runnable || summary.Tasks[1].Runnable {
		t.Fatalf("legacy compatibility state was not explicit: %#v", summary)
	}
}

func TestCatalogueRejectsMissingExplicitReleaseManifest(t *testing.T) {
	t.Setenv("RUNTIME_RELEASE_MANIFEST", filepath.Join(t.TempDir(), "missing.json"))
	if _, err := NewCatalogue(); err == nil || !strings.Contains(err.Error(), "read release manifest") {
		t.Fatalf("missing explicitly selected release manifest was silently ignored: %v", err)
	}
}

func TestCatalogueRejectsLegacyQuestionBindingsForReleasedTasks(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "legacy-task")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"taskId":"legacy-task","revision":1,"status":"released","profile":"node","runtime":"Node.js 24","image":"fluent-runtime-task-node:1","checkCommand":["node"],"editableFiles":["main.js"],"timeoutMs":20000,"memoryMb":512,"cpus":1,"questionKeys":["Q777"],"questionReleaseId":"question-release-15e032d7b732f8c1","declaredTests":["main task"]}`
	if err := os.WriteFile(filepath.Join(directory, "task.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RUNTIME_TASKS_ROOT", root)
	if _, err := NewCatalogue(); err == nil {
		t.Fatal("legacy question binding was accepted")
	}
}

func TestCatalogueLoadsImmutableReleaseManifestOverlay(t *testing.T) {
	t.Setenv("RUNTIME_TASKS_ROOT", historicalTasksRoot(t))
	t.Setenv("RUNTIME_RELEASE_MANIFEST", filepath.Join("..", "..", "releases", "task-release-2026-08-24-qb-d550846f-i2.json"))
	catalogue, err := NewCatalogue()
	if err != nil {
		t.Fatal(err)
	}
	task, ok := catalogue.Task("node-rate-limiter-001", 1)
	if !ok {
		t.Fatal("manifest task was not loaded")
	}
	if task.QuestionReleaseID != "question-release-d550846f4743c4d3" {
		t.Fatalf("unexpected Question Brain release: %q", task.QuestionReleaseID)
	}
	if len(task.QuestionBindings) != 3 || task.QuestionBindings[0].StableKey != "question.q315" {
		t.Fatalf("immutable question bindings were not projected: %#v", task.QuestionBindings)
	}
	if task.QuestionBindings[0].RevisionID == "" || len(task.QuestionBindings[0].ContentHash) != 64 {
		t.Fatalf("binding is missing revision identity or hash: %#v", task.QuestionBindings[0])
	}
	if len(task.CapabilityKeys) != 1 || task.CapabilityKeys[0] != "capability.distributed-systems.rate-limiter" {
		t.Fatalf("capabilityKeys must be projected from the active I2 release: %#v", task.CapabilityKeys)
	}
	legacy, ok := catalogue.Task("project-book-boundary-001", 1)
	if !ok || len(legacy.QuestionBindings) != 0 || len(legacy.CapabilityKeys) != 1 || legacy.CapabilityKeys[0] != "capability.tier1-capstone" {
		t.Fatalf("capability-only release was not projected: %#v (ok=%v)", legacy, ok)
	}

	// The old revision descriptor remains the compatibility source file. The
	// new immutable identities live in the release manifest instead of rewriting
	// a previously released task revision.
	body, err := os.ReadFile(filepath.Join("..", "..", "tasks", "node-rate-limiter-001", "task.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "questionBindings") {
		t.Fatal("old task revision was rewritten instead of using the release manifest")
	}
}

func TestCatalogueLoadsTaskFamiliesAndGroupsLanguageRevisions(t *testing.T) {
	t.Setenv("RUNTIME_TASKS_ROOT", historicalTasksRoot(t))
	t.Setenv("RUNTIME_RELEASE_MANIFEST", filepath.Join("..", "..", "releases", "task-release-2026-08-25-qb-d550846f-g3.json"))
	t.Setenv("RUNTIME_TASK_FAMILY_MANIFEST", historicalTaskFamilyManifest(t))
	catalogue, err := NewCatalogue()
	if err != nil {
		t.Fatal(err)
	}
	families := catalogue.TaskFamilies()
	if families.ContractVersion != contracts.TaskFamilyContractVersion || len(families.Families) != 15 {
		t.Fatalf("unexpected family release: %#v", families)
	}
	rate, ok := catalogue.TaskFamily("task-family.rate-limiter")
	if !ok || len(rate.Revisions) != 4 || !rate.Runnable {
		t.Fatalf("rate limiter family did not group four runnable revisions: %#v (ok=%v)", rate, ok)
	}
	project, ok := catalogue.TaskFamily("task-family.project-book-boundary")
	if !ok || project.Runnable || project.Revisions[0].Availability != "unreleased" {
		t.Fatalf("unreleased project family was advertised as runnable: %#v (ok=%v)", project, ok)
	}
	task, ok := catalogue.Task("node-rate-limiter-001", 1)
	if !ok || task.TaskFamilyKey != "task-family.rate-limiter" || task.ImmutableHash == "" || task.Availability != "runnable" {
		t.Fatalf("task revision did not receive family metadata: %#v (ok=%v)", task, ok)
	}
}

func TestCatalogueLoadsG8ReleaseJoinAndCapabilitySnapshot(t *testing.T) {
	t.Setenv("RUNTIME_TASKS_ROOT", historicalTasksRoot(t))
	t.Setenv("RUNTIME_RELEASE_MANIFEST", filepath.Join("..", "..", "releases", "task-release-2026-08-25-qb-d00a1493-g8.json"))
	t.Setenv("RUNTIME_TASK_FAMILY_MANIFEST", historicalTaskFamilyManifest(t))
	catalogue, err := NewCatalogue()
	if err != nil {
		t.Fatal(err)
	}
	summary := catalogue.ReleaseSummary()
	if summary.RuntimeReleaseID != "runtime-task-release-2026-08-25-qb-d00a1493-g8" ||
		summary.QuestionReleaseID != "question-release-d00a14931e607336" ||
		summary.CapabilityBindingReleaseID != "question-capability-release-3c38b4c8c0fa7f47" ||
		summary.CapabilityRegistryReleaseID != "capability-registry-2026-08-25-v3" ||
		summary.TaskFamilyReleaseID != "task-family-release-2026-08-25" {
		t.Fatalf("G8 release pins were not projected: %#v", summary)
	}
	project, ok := catalogue.Task("project-book-boundary-001", 1)
	if !ok || project.Runnable || len(project.QuestionBindings) != 0 || len(project.CapabilityKeys) != 1 || project.CapabilityKeys[0] != "capability.delivery-observability.execution-boundary" {
		t.Fatalf("capability-only project join was not explicit: %#v (ok=%v)", project, ok)
	}
}

// historicalTasksRoot models the immutable catalogue that accompanied the
// pre-rate-limiter language releases. The production catalogue is intentionally
// current; historical release tests must not mutate old manifests to make them
// appear to contain revisions that did not exist at that release.
func historicalTasksRoot(t *testing.T) string {
	t.Helper()
	source := filepath.Join("..", "..", "tasks")
	destination := filepath.Join(t.TempDir(), "tasks")
	entries, err := os.ReadDir(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "csharp-rate-limiter-001" || entry.Name() == "ts-rate-limiter-001" {
			continue
		}
		if err := copyTaskTree(source, destination, entry.Name()); err != nil {
			t.Fatal(err)
		}
	}
	return destination
}

func copyTaskTree(source, destination, taskID string) error {
	return filepath.WalkDir(filepath.Join(source, taskID), func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, body, 0o644)
	})
}

func historicalTaskFamilyManifest(t *testing.T) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "..", "task-families", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		ContractVersion string                   `json:"contractVersion"`
		ReleaseID       string                   `json:"releaseId"`
		Families        []map[string]interface{} `json:"families"`
	}
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, family := range manifest.Families {
		revisions, ok := family["revisions"].([]interface{})
		if !ok {
			continue
		}
		filtered := revisions[:0]
		for _, value := range revisions {
			revision, ok := value.(map[string]interface{})
			if !ok {
				continue
			}
			taskID, _ := revision["taskId"].(string)
			if taskID != "csharp-rate-limiter-001" && taskID != "ts-rate-limiter-001" {
				filtered = append(filtered, value)
			}
		}
		family["revisions"] = filtered
	}
	filteredBody, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, filteredBody, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCatalogueRejectsCapabilityOutsideG8PinnedRegistry(t *testing.T) {
	manifestPath := filepath.Join(t.TempDir(), "release.json")
	manifest := `{"contractVersion":"fluent-task-runtime.task-release.v3","releaseId":"runtime-task-release-test","workspaceKey":"fluent-interview","questionBrainContractVersion":"question-brain.release.v1","questionReleaseId":"question-release-aaaaaaaaaaaaaaaa","questionSourceSnapshotId":"question-release-aaaaaaaaaaaaaaaa","capabilityBindingReleaseId":"question-capability-release-aaaaaaaaaaaaaaaa","capabilityRegistryReleaseId":"capability-registry-test","capabilityKeys":["capability.valid-key"],"taskFamilyReleaseId":"task-family-test","tasks":[{"taskId":"task","revision":1,"taskFamilyKey":"task-family.task","questionBindings":[],"capabilityKeys":["capability.not-in-snapshot"]}]}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RUNTIME_RELEASE_MANIFEST", manifestPath)
	if _, err := loadReleaseManifest(); err == nil || !strings.Contains(err.Error(), "outside pinned registry") {
		t.Fatalf("capability outside the pinned registry was accepted: %v", err)
	}
}

func TestCatalogueOverlaysLegacyDescriptorReleaseMetadata(t *testing.T) {
	root := t.TempDir()
	taskRoot := filepath.Join(root, "legacy-task")
	if err := os.MkdirAll(taskRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	legacyDescriptor := `{"taskId":"legacy-task","revision":7,"status":"released","profile":"node","runtime":"Node.js 24","image":"fluent-runtime-task-node:1","checkCommand":["node"],"editableFiles":["main.js"],"timeoutMs":20000,"memoryMb":512,"cpus":1}`
	if err := os.WriteFile(filepath.Join(taskRoot, "task.json"), []byte(legacyDescriptor), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := `{"contractVersion":"fluent-task-runtime.task-release.v1","releaseId":"runtime-task-release-new","questionReleaseId":"question-release-aaaaaaaaaaaaaaaa","tasks":[{"taskId":"legacy-task","revision":7,"questionBindings":[{"stableKey":"question.q999","revisionId":"11111111-1111-4111-8111-111111111111","contentHash":"` + strings.Repeat("a", 64) + `"}],"capabilityKeys":[]}]}`
	manifestPath := filepath.Join(root, "release.json")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RUNTIME_TASKS_ROOT", root)
	t.Setenv("RUNTIME_RELEASE_MANIFEST", manifestPath)
	catalogue, err := NewCatalogue()
	if err != nil {
		t.Fatal(err)
	}
	task, ok := catalogue.Task("legacy-task", 7)
	if !ok || !task.Runnable || task.QuestionReleaseID != "question-release-aaaaaaaaaaaaaaaa" || len(task.QuestionBindings) != 1 || task.QuestionBindings[0].StableKey != "question.q999" {
		t.Fatalf("new release did not overlay historical descriptor metadata: %#v (ok=%v)", task, ok)
	}
}

func TestCatalogueRejectsMalformedQuestionBinding(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "bound-task")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	descriptor := map[string]any{
		"taskId": "bound-task", "revision": 2, "status": "released", "profile": "node", "runtime": "Node.js 24",
		"image": "fluent-runtime-task-node:1", "checkCommand": []string{"node"}, "editableFiles": []string{"main.js"},
		"timeoutMs": 20000, "memoryMb": 512, "cpus": 1, "questionReleaseId": "question-release-15e032d7b732f8c1",
		"questionBindings": []contracts.QuestionBinding{{StableKey: "question.q777", RevisionID: "not-a-uuid", ContentHash: strings.Repeat("a", 64)}},
	}
	body, err := json.Marshal(descriptor)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "task.json"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RUNTIME_TASKS_ROOT", root)
	if _, err := NewCatalogue(); err == nil || !strings.Contains(err.Error(), "revisionId") {
		t.Fatalf("malformed question binding was accepted: %v", err)
	}
}
