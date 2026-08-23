package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCatalogueHonoursTaskReleaseMetadata(t *testing.T) {
	root := t.TempDir()
	writeTask := func(id, status, user string) {
		t.Helper()
		directory := filepath.Join(root, id)
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
		body := `{"taskId":"` + id + `","revision":7,"status":"` + status + `","profile":"node","runtime":"Node.js 24","image":"fluent-runtime-task-node:1","checkCommand":["node"],"editableFiles":["main.js"],"timeoutMs":20000,"memoryMb":512,"cpus":1,"user":"` + user + `","declaredTests":["main task"]}`
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
	if !ok || released.Status != "released" || released.User != "1000:1000" {
		t.Fatalf("released descriptor was not preserved: %#v (ok=%v)", released, ok)
	}
	declared, ok := catalogue.Task("declared-task", 7)
	if !ok || declared.Status != "declared" {
		t.Fatalf("declared descriptor was not preserved: %#v (ok=%v)", declared, ok)
	}
}
