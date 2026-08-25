package contracts

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func goldenCapabilityBundle(t *testing.T) map[string]any {
	t.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate contract fixture")
	}
	paths := []string{
		filepath.Join(filepath.Dir(sourceFile), "..", "..", "..", "docs", "contracts", "capability-mastery-bundle.v1.fixture.json"),
		filepath.Join(filepath.Dir(sourceFile), "..", "docs", "contracts", "capability-mastery-bundle.v1.fixture.json"),
	}
	var payload []byte
	var err error
	for _, path := range paths {
		payload, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}
	if err != nil {
		t.Fatalf("read golden fixture: %v", err)
	}
	var value map[string]any
	if err := json.Unmarshal(payload, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestCapabilityMasteryGoldenFixture(t *testing.T) {
	payload, err := json.Marshal(goldenCapabilityBundle(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateCapabilityMasteryBundleJSON(payload); err != nil {
		t.Fatalf("golden fixture rejected: %v", err)
	}
}

func TestCapabilityMasteryRejectsStaleAndContradictoryJoins(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(map[string]any)
		code   string
	}{
		{"stale card", func(value map[string]any) {
			value["learningSession"].(map[string]any)["cardContentHash"] = strings.Repeat("b", 64)
		}, "stale-question-card-hash"},
		{"stale task", func(value map[string]any) {
			value["learningSession"].(map[string]any)["taskRevision"].(map[string]any)["immutableHash"] = strings.Repeat("b", 64)
		}, "stale-task-revision-hash"},
		{"non-human mastery", func(value map[string]any) { value["mastery"].(map[string]any)["provenance"] = "e2e" }, "non-human-mastery-provenance"},
		{"contradictory release", func(value map[string]any) {
			value["capabilityDossier"].(map[string]any)["taskFamily"].(map[string]any)["status"] = "pending"
		}, "contradictory-released-runnable"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			value := goldenCapabilityBundle(t)
			testCase.mutate(value)
			payload, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			var contractError *CapabilityMasteryContractError
			if !errors.As(ValidateCapabilityMasteryBundleJSON(payload), &contractError) || contractError.Code != testCase.code {
				t.Fatalf("expected %s, got %v", testCase.code, ValidateCapabilityMasteryBundleJSON(payload))
			}
		})
	}
}

func TestCapabilityMasteryRejectsUnsupportedVersion(t *testing.T) {
	value := goldenCapabilityBundle(t)
	value["contractVersion"] = "capability-mastery-bundle.v0"
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var contractError *CapabilityMasteryContractError
	if !errors.As(ValidateCapabilityMasteryBundleJSON(payload), &contractError) || contractError.Code != "unsupported_contract_version" {
		t.Fatalf("expected typed unsupported error, got %v", ValidateCapabilityMasteryBundleJSON(payload))
	}
}
