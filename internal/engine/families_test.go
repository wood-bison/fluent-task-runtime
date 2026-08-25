package engine

import (
	"strings"
	"testing"

	"github.com/wood-bison/fluent-task-runtime/contracts"
)

func familyManifestForCompatibility(kind, language, profile string) taskFamilyManifest {
	return taskFamilyManifest{
		ContractVersion: taskFamilyManifestContract,
		ReleaseID:       "task-family-release-test",
		Families: []taskFamilyEntry{{
			Key:            "task-family.compatibility-test",
			Title:          contracts.LocalizedContractText{EN: "Compatibility", RU: "Совместимость"},
			Brief:          contracts.LocalizedContractText{EN: "Test", RU: "Тест"},
			CapabilityKeys: []string{"capability.runtime.compatibility-test"},
			ExecutionKind:  kind,
			RubricRef:      "rubric.compatibility-test.v1",
			Status:         "released",
			Revisions: []taskFamilyRevisionEntry{{
				TaskID:        "compatibility-test",
				Revision:      1,
				Language:      language,
				Profile:       profile,
				Availability:  "runnable",
				ImmutableHash: strings.Repeat("a", 64),
			}},
		}},
	}
}

func TestTaskFamilyManifestRejectsSqlRevisionInsideCodeFamily(t *testing.T) {
	err := validateTaskFamilyManifest(familyManifestForCompatibility("code", "sql", "postgres"))
	if err == nil || !strings.Contains(err.Error(), "SQL must be a separate SQL family") {
		t.Fatalf("SQL revision was accepted inside a code family: %v", err)
	}
}

func TestTaskFamilyManifestRequiresPostgresProfileForSqlFamily(t *testing.T) {
	err := validateTaskFamilyManifest(familyManifestForCompatibility("sql", "typescript", "node"))
	if err == nil || !strings.Contains(err.Error(), "SQL families must use sql/postgres") {
		t.Fatalf("non-PostgreSQL revision was accepted inside an SQL family: %v", err)
	}
}

func TestTaskFamilyManifestAllowsExplicitSqlFamily(t *testing.T) {
	if err := validateTaskFamilyManifest(familyManifestForCompatibility("sql", "sql", "postgres")); err != nil {
		t.Fatalf("valid SQL family was rejected: %v", err)
	}
}
