package httpapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// historicalTasksRoot models the immutable catalogue that accompanied the
// pre-rate-limiter language releases. Current production uses the full tasks
// tree; historical manifests are tested against their matching snapshot.
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
