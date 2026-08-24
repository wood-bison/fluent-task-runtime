package contracts

import (
	"fmt"
	"regexp"
	"strings"
)

const QuestionCapabilityTaskContractVersion = "question-capability-task.v1"

type LocalizedContractText struct {
	EN string `json:"en"`
	RU string `json:"ru"`
}

// Capability is an observable skill. It is deliberately not a task or a
// runtime profile; Runtime only carries the identity supplied by the release.
type Capability struct {
	Key        string                `json:"key"`
	Title      LocalizedContractText `json:"title"`
	Lifecycle  string                `json:"lifecycle"`
	Aliases    []string              `json:"aliases,omitempty"`
	Supersedes []string              `json:"supersedes,omitempty"`
}

type CapabilityDomainBinding struct {
	CapabilityKey string `json:"capabilityKey"`
	DomainKey     string `json:"domainKey"`
	Role          string `json:"role"`
}

type QuestionCapabilityBinding struct {
	Question      QuestionBinding `json:"question"`
	CapabilityKey string          `json:"capabilityKey"`
	Role          string          `json:"role"`
	Provenance    string          `json:"provenance"`
	Confidence    *float64        `json:"confidence,omitempty"`
}

// TaskFamily is language-neutral. Executable source and sandbox policy are
// owned by one of its immutable TaskRevision records.
type TaskFamily struct {
	Key              string                `json:"key"`
	Title            LocalizedContractText `json:"title"`
	Brief            LocalizedContractText `json:"brief,omitempty"`
	CapabilityKeys   []string              `json:"capabilityKeys"`
	QuestionBindings []QuestionBinding     `json:"questionBindings"`
	RevisionIDs      []string              `json:"revisionIds"`
	RubricRef        string                `json:"rubricRef,omitempty"`
	Status           string                `json:"status"`
	Runnable         bool                  `json:"runnable"`
	Revisions        []TaskFamilyRevision  `json:"revisions,omitempty"`
}

// TaskRevision is the executable, language-specific identity. It is separate
// from the existing Task descriptor so a future API can expose the family
// without making the old response mutable.
type TaskRevision struct {
	TaskID        string `json:"taskId"`
	Revision      int    `json:"revision"`
	TaskFamilyKey string `json:"taskFamilyKey"`
	Language      string `json:"language"`
	Profile       string `json:"profile"`
	Status        string `json:"status"`
	ImmutableHash string `json:"immutableHash"`
}

type ContentRelation struct {
	RelationID string   `json:"relationId"`
	From       string   `json:"from"`
	To         string   `json:"to"`
	Kind       string   `json:"kind"`
	Status     string   `json:"status"`
	Provenance string   `json:"provenance"`
	Confidence *float64 `json:"confidence,omitempty"`
}

type ContractRun struct {
	RunID        string `json:"runId"`
	TaskID       string `json:"taskId"`
	TaskRevision int    `json:"taskRevision"`
	Status       string `json:"status"`
	ResultHash   string `json:"resultHash"`
}

type ContractEvidence struct {
	EvidenceID string `json:"evidenceId"`
	RunID      string `json:"runId"`
	State      string `json:"state"`
	RecordedAt string `json:"recordedAt"`
}

type QuestionCapabilityTaskContract struct {
	ContractVersion            string                      `json:"contractVersion"`
	QuestionCards              []ContractQuestionCard      `json:"questionCards"`
	Capabilities               []Capability                `json:"capabilities"`
	CapabilityDomainBindings   []CapabilityDomainBinding   `json:"capabilityDomainBindings"`
	QuestionCapabilityBindings []QuestionCapabilityBinding `json:"questionCapabilityBindings"`
	TaskFamilies               []TaskFamily                `json:"taskFamilies"`
	TaskRevisions              []TaskRevision              `json:"taskRevisions"`
	ContentRelations           []ContentRelation           `json:"contentRelations"`
	Runs                       []ContractRun               `json:"runs"`
	Evidence                   []ContractEvidence          `json:"evidence"`
}

type ContractQuestionCard struct {
	StableKey   string   `json:"stableKey"`
	RevisionID  string   `json:"revisionId"`
	ContentHash string   `json:"contentHash"`
	Locales     []string `json:"locales"`
	Status      string   `json:"status"`
}

var (
	contractCapabilityKey = regexp.MustCompile(`^capability\.[a-z0-9-]+\.[a-z0-9][a-z0-9-]*$`)
	contractQuestionKey   = regexp.MustCompile(`^question\.[a-z0-9][a-z0-9._-]*$`)
	contractDomainKey     = regexp.MustCompile(`^domain\.[a-z0-9][a-z0-9-]*$`)
	contractSHA256        = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

func validContractCapabilityKey(value string) bool {
	return contractCapabilityKey.MatchString(value) && !regexp.MustCompile(`-\d{3,}$`).MatchString(value)
}

func validQuestionBinding(binding QuestionBinding) bool {
	return contractQuestionKey.MatchString(binding.StableKey) && binding.RevisionID != "" && contractSHA256.MatchString(binding.ContentHash)
}

func (c QuestionCapabilityTaskContract) Validate() error {
	if c.ContractVersion != QuestionCapabilityTaskContractVersion {
		return fmt.Errorf("unsupported contractVersion %q", c.ContractVersion)
	}
	capabilities := make(map[string]struct{}, len(c.Capabilities))
	questions := make(map[string]struct{}, len(c.QuestionCards))
	for i, card := range c.QuestionCards {
		if !contractQuestionKey.MatchString(card.StableKey) || card.RevisionID == "" || !contractSHA256.MatchString(card.ContentHash) {
			return fmt.Errorf("questionCards[%d] has invalid immutable identity", i)
		}
		questions[card.StableKey] = struct{}{}
	}
	for i, capability := range c.Capabilities {
		if !validContractCapabilityKey(capability.Key) || capability.Title.EN == "" || capability.Title.RU == "" {
			return fmt.Errorf("capabilities[%d] has invalid registered capability", i)
		}
		capabilities[capability.Key] = struct{}{}
	}
	for i, binding := range c.CapabilityDomainBindings {
		if _, ok := capabilities[binding.CapabilityKey]; !ok || !contractDomainKey.MatchString(binding.DomainKey) {
			return fmt.Errorf("capabilityDomainBindings[%d] references an unknown capability/domain", i)
		}
	}
	for i, binding := range c.QuestionCapabilityBindings {
		if _, ok := questions[binding.Question.StableKey]; !ok || !validQuestionBinding(binding.Question) {
			return fmt.Errorf("questionCapabilityBindings[%d] references an unknown question", i)
		}
		if _, ok := capabilities[binding.CapabilityKey]; !ok || strings.TrimSpace(binding.Provenance) == "" {
			return fmt.Errorf("questionCapabilityBindings[%d] has an unknown capability/provenance", i)
		}
		if binding.Confidence != nil && (*binding.Confidence < 0 || *binding.Confidence > 1) {
			return fmt.Errorf("questionCapabilityBindings[%d] confidence is outside 0..1", i)
		}
	}
	families := make(map[string]struct{}, len(c.TaskFamilies))
	for i, family := range c.TaskFamilies {
		if !regexp.MustCompile(`^task-family\.[a-z0-9][a-z0-9-]*$`).MatchString(family.Key) || len(family.CapabilityKeys) == 0 || len(family.RevisionIDs) == 0 {
			return fmt.Errorf("taskFamilies[%d] has invalid identity/cardinality", i)
		}
		families[family.Key] = struct{}{}
		if family.Title.EN == "" || family.Title.RU == "" {
			return fmt.Errorf("taskFamilies[%d] requires EN/RU title", i)
		}
		for _, capabilityKey := range family.CapabilityKeys {
			if _, ok := capabilities[capabilityKey]; !ok {
				return fmt.Errorf("taskFamilies[%d] references unknown capability %q", i, capabilityKey)
			}
		}
		for _, binding := range family.QuestionBindings {
			if !validQuestionBinding(binding) {
				return fmt.Errorf("taskFamilies[%d] has invalid question binding", i)
			}
		}
	}
	for i, revision := range c.TaskRevisions {
		_, familyExists := families[revision.TaskFamilyKey]
		if revision.Revision < 1 || revision.Language == "" || revision.Profile == "" || !familyExists || !contractSHA256.MatchString(revision.ImmutableHash) {
			return fmt.Errorf("taskRevisions[%d] is not a complete immutable revision", i)
		}
	}
	return nil
}

// ValidateQuestionKeysProjection keeps the deprecated response field honest.
func ValidateQuestionKeysProjection(questionKeys, capabilityKeys []string) error {
	capabilities := make(map[string]struct{}, len(capabilityKeys))
	for _, key := range capabilityKeys {
		capabilities[key] = struct{}{}
	}
	for _, key := range questionKeys {
		if !contractQuestionKey.MatchString(key) {
			return fmt.Errorf("questionKeys contains non-question key %q", key)
		}
		if _, exists := capabilities[key]; exists {
			return fmt.Errorf("questionKeys contains capability key %q", key)
		}
	}
	return nil
}
