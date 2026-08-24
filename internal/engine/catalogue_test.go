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
			binding = `,"questionKeys":["question.q777"],"questionReleaseId":"question-release-15e032d7b732f8c1"`
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
	if !ok || released.Status != "released" || released.User != "1000:1000" || released.QuestionReleaseID != "question-release-15e032d7b732f8c1" || len(released.QuestionKeys) != 1 || released.QuestionKeys[0] != "question.q777" {
		t.Fatalf("released descriptor was not preserved: %#v (ok=%v)", released, ok)
	}
	declared, ok := catalogue.Task("declared-task", 7)
	if !ok || declared.Status != "declared" {
		t.Fatalf("declared descriptor was not preserved: %#v (ok=%v)", declared, ok)
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
	t.Setenv("RUNTIME_TASKS_ROOT", filepath.Join("..", "..", "tasks"))
	t.Setenv("RUNTIME_RELEASE_MANIFEST", filepath.Join("..", "..", "releases", "task-release-2026-08-24.json"))
	catalogue, err := NewCatalogue()
	if err != nil {
		t.Fatal(err)
	}
	task, ok := catalogue.Task("node-rate-limiter-001", 1)
	if !ok {
		t.Fatal("manifest task was not loaded")
	}
	if task.QuestionReleaseID != "question-release-15e032d7b732f8c1" {
		t.Fatalf("unexpected Question Brain release: %q", task.QuestionReleaseID)
	}
	if len(task.QuestionBindings) != 3 || task.QuestionBindings[0].StableKey != "question.q315" {
		t.Fatalf("immutable question bindings were not projected: %#v", task.QuestionBindings)
	}
	if task.QuestionBindings[0].RevisionID == "" || len(task.QuestionBindings[0].ContentHash) != 64 {
		t.Fatalf("binding is missing revision identity or hash: %#v", task.QuestionBindings[0])
	}
	if task.CapabilityKeys == nil || len(task.CapabilityKeys) != 0 {
		t.Fatalf("capabilityKeys must be exposed as an empty array for this task: %#v", task.CapabilityKeys)
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
