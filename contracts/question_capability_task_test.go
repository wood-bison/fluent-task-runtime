package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func contractFixture() QuestionCapabilityTaskContract {
	hash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	return QuestionCapabilityTaskContract{
		ContractVersion: QuestionCapabilityTaskContractVersion,
		QuestionCards:   []ContractQuestionCard{{StableKey: "question.rate-limiter", RevisionID: "qrev-1", ContentHash: hash, Locales: []string{"en", "ru"}, Status: "published"}},
		Capabilities:    []Capability{{Key: "capability.distributed-systems.rate-limiter", Title: LocalizedContractText{EN: "Rate limiting", RU: "Ограничение частоты"}, Lifecycle: "active"}},
		CapabilityDomainBindings: []CapabilityDomainBinding{
			{CapabilityKey: "capability.distributed-systems.rate-limiter", DomainKey: "domain.distributed-systems", Role: "primary"},
			{CapabilityKey: "capability.distributed-systems.rate-limiter", DomainKey: "domain.http-api", Role: "secondary"},
		},
		QuestionCapabilityBindings: []QuestionCapabilityBinding{{Question: QuestionBinding{StableKey: "question.rate-limiter", RevisionID: "qrev-1", ContentHash: hash}, CapabilityKey: "capability.distributed-systems.rate-limiter", Role: "primary", Provenance: "review"}},
		TaskFamilies:               []TaskFamily{{Key: "task-family.token-bucket", Title: LocalizedContractText{EN: "Token bucket", RU: "Token bucket"}, CapabilityKeys: []string{"capability.distributed-systems.rate-limiter"}, RevisionIDs: []string{"task.token-bucket-go"}, Status: "released"}},
		TaskRevisions:              []TaskRevision{{TaskID: "task.token-bucket-go", Revision: 1, TaskFamilyKey: "task-family.token-bucket", Language: "go", Profile: "go", Status: "released", ImmutableHash: hash}},
	}
}

func TestQuestionCapabilityTaskContractAcceptsMultipleDomains(t *testing.T) {
	if err := contractFixture().Validate(); err != nil {
		t.Fatalf("valid fixture rejected: %v", err)
	}
}

func TestQuestionCapabilityTaskContractRejectsTaskSequenceCapability(t *testing.T) {
	fixture := contractFixture()
	fixture.Capabilities[0].Key = "capability.runtime.rate-limiter-001"
	if err := fixture.Validate(); err == nil {
		t.Fatal("task sequence capability key was accepted")
	}
}

func TestQuestionCapabilityTaskContractRejectsQuestionCapabilityProjection(t *testing.T) {
	if err := ValidateQuestionKeysProjection([]string{"question.rate-limiter", "capability.distributed-systems.rate-limiter"}, []string{"capability.distributed-systems.rate-limiter"}); err == nil {
		t.Fatal("capability leaked into questionKeys")
	}
}

func TestWorkspaceFixtureValidatesInRuntime(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate contract fixture")
	}
	payload, err := os.ReadFile(filepath.Join(filepath.Dir(sourceFile), "..", "docs", "contracts", "question-capability-task.v1.fixture.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var contract QuestionCapabilityTaskContract
	if err := json.Unmarshal(payload, &contract); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	if err := contract.Validate(); err != nil {
		t.Fatalf("workspace fixture rejected: %v", err)
	}
}
