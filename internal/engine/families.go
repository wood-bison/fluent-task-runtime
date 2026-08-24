package engine

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/wood-bison/fluent-task-runtime/contracts"
)

const (
	taskFamilyManifestEnv      = "RUNTIME_TASK_FAMILY_MANIFEST"
	taskFamilyManifestContract = "fluent-task-runtime.task-families.v1"
)

var (
	taskFamilyKeyPattern = regexp.MustCompile(`^task-family\.[a-z0-9][a-z0-9-]*$`)
	capabilityKeyPattern = regexp.MustCompile(`^capability\.[a-z0-9][a-z0-9-]*(?:\.[a-z0-9][a-z0-9-]*)?$`)
	hashPattern          = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

type taskFamilyManifest struct {
	ContractVersion string            `json:"contractVersion"`
	ReleaseID       string            `json:"releaseId"`
	Families        []taskFamilyEntry `json:"families"`
}

type taskFamilyEntry struct {
	Key            string                          `json:"key"`
	Title          contracts.LocalizedContractText `json:"title"`
	Brief          contracts.LocalizedContractText `json:"brief"`
	CapabilityKeys []string                        `json:"capabilityKeys"`
	RubricRef      string                          `json:"rubricRef"`
	Status         string                          `json:"status"`
	Revisions      []taskFamilyRevisionEntry       `json:"revisions"`
}

type taskFamilyRevisionEntry struct {
	TaskID        string `json:"taskId"`
	Revision      int    `json:"revision"`
	Language      string `json:"language"`
	Profile       string `json:"profile"`
	Availability  string `json:"availability"`
	ImmutableHash string `json:"immutableHash"`
}

type taskFamilyRevisionKey struct {
	TaskID   string
	Revision int
}

func loadTaskFamilyManifest() (*taskFamilyManifest, error) {
	path := strings.TrimSpace(os.Getenv(taskFamilyManifestEnv))
	if path == "" {
		candidate := filepath.Join("task-families", "manifest.json")
		if _, err := os.Stat(candidate); err != nil {
			if os.IsNotExist(err) {
				return nil, nil
			}
			return nil, fmt.Errorf("stat task family manifest %q: %w", candidate, err)
		}
		path = candidate
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read task family manifest %q: %w", path, err)
	}
	var manifest taskFamilyManifest
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("decode task family manifest %q: %w", path, err)
	}
	if manifest.ContractVersion != taskFamilyManifestContract {
		return nil, fmt.Errorf("task family manifest %q has unsupported contractVersion %q", path, manifest.ContractVersion)
	}
	if strings.TrimSpace(manifest.ReleaseID) == "" || len(manifest.Families) == 0 {
		return nil, fmt.Errorf("task family manifest %q requires releaseId and families", path)
	}
	if err := validateTaskFamilyManifest(manifest); err != nil {
		return nil, fmt.Errorf("task family manifest %q: %w", path, err)
	}
	return &manifest, nil
}

func validateTaskFamilyManifest(manifest taskFamilyManifest) error {
	seenFamilies := make(map[string]struct{}, len(manifest.Families))
	seenRevisions := make(map[taskFamilyRevisionKey]string)
	for _, family := range manifest.Families {
		family.Key = strings.TrimSpace(family.Key)
		if !taskFamilyKeyPattern.MatchString(family.Key) {
			return fmt.Errorf("invalid family key %q", family.Key)
		}
		if _, exists := seenFamilies[family.Key]; exists {
			return fmt.Errorf("duplicate family key %q", family.Key)
		}
		seenFamilies[family.Key] = struct{}{}
		if strings.TrimSpace(family.Title.EN) == "" || strings.TrimSpace(family.Title.RU) == "" || strings.TrimSpace(family.Brief.EN) == "" || strings.TrimSpace(family.Brief.RU) == "" {
			return fmt.Errorf("family %q requires EN/RU title and brief", family.Key)
		}
		if strings.TrimSpace(family.RubricRef) == "" {
			return fmt.Errorf("family %q requires rubricRef", family.Key)
		}
		if family.Status != "released" && family.Status != "unreleased" && family.Status != "draft" {
			return fmt.Errorf("family %q has invalid status %q", family.Key, family.Status)
		}
		if len(family.CapabilityKeys) == 0 || len(family.Revisions) == 0 {
			return fmt.Errorf("family %q requires capabilityKeys and revisions", family.Key)
		}
		seenCapabilities := make(map[string]struct{}, len(family.CapabilityKeys))
		for _, capability := range family.CapabilityKeys {
			capability = strings.TrimSpace(capability)
			if !capabilityKeyPattern.MatchString(capability) {
				return fmt.Errorf("family %q has invalid capability key %q", family.Key, capability)
			}
			if _, exists := seenCapabilities[capability]; exists {
				return fmt.Errorf("family %q repeats capability key %q", family.Key, capability)
			}
			seenCapabilities[capability] = struct{}{}
		}
		for _, revision := range family.Revisions {
			key := taskFamilyRevisionKey{TaskID: strings.TrimSpace(revision.TaskID), Revision: revision.Revision}
			if key.TaskID == "" || revision.Revision < 1 {
				return fmt.Errorf("family %q has invalid revision identity", family.Key)
			}
			if revision.Language == "" || revision.Profile == "" || !hashPattern.MatchString(revision.ImmutableHash) {
				return fmt.Errorf("family %q revision %s@%d has incomplete language/profile/hash", family.Key, key.TaskID, key.Revision)
			}
			if revision.Availability != "runnable" && revision.Availability != "brief_only" && revision.Availability != "profile_unavailable" && revision.Availability != "superseded" && revision.Availability != "unreleased" {
				return fmt.Errorf("family %q revision %s@%d has invalid availability %q", family.Key, key.TaskID, key.Revision, revision.Availability)
			}
			if owner, exists := seenRevisions[key]; exists {
				return fmt.Errorf("revision %s@%d belongs to families %q and %q", key.TaskID, key.Revision, owner, family.Key)
			}
			seenRevisions[key] = family.Key
		}
	}
	return nil
}

func bindTaskFamilies(tasks []contracts.Task, manifest *taskFamilyManifest, tasksRoot string) ([]contracts.Task, contracts.TaskFamilies, error) {
	if manifest == nil {
		return tasks, contracts.TaskFamilies{}, nil
	}
	byRevision := make(map[taskFamilyRevisionKey]int, len(tasks))
	for index, task := range tasks {
		byRevision[taskFamilyRevisionKey{TaskID: task.ID, Revision: task.Revision}] = index
	}
	seen := make(map[taskFamilyRevisionKey]struct{}, len(tasks))
	result := contracts.TaskFamilies{ContractVersion: contracts.TaskFamilyContractVersion, ReleaseID: manifest.ReleaseID, Families: make([]contracts.TaskFamily, 0, len(manifest.Families))}
	for _, family := range manifest.Families {
		projection := contracts.TaskFamily{
			Key: family.Key, Title: family.Title, Brief: family.Brief,
			CapabilityKeys: cloneStrings(family.CapabilityKeys), RubricRef: family.RubricRef,
			Status: family.Status, Revisions: make([]contracts.TaskFamilyRevision, 0, len(family.Revisions)),
		}
		bindingSeen := make(map[string]struct{})
		for _, revision := range family.Revisions {
			key := taskFamilyRevisionKey{TaskID: revision.TaskID, Revision: revision.Revision}
			index, exists := byRevision[key]
			if !exists {
				return nil, contracts.TaskFamilies{}, fmt.Errorf("family %q references missing task revision %s@%d", family.Key, key.TaskID, key.Revision)
			}
			if _, exists := seen[key]; exists {
				return nil, contracts.TaskFamilies{}, fmt.Errorf("task revision %s@%d is assigned more than once", key.TaskID, key.Revision)
			}
			seen[key] = struct{}{}
			task := &tasks[index]
			if task.TaskFamilyKey != "" && task.TaskFamilyKey != family.Key {
				return nil, contracts.TaskFamilies{}, fmt.Errorf("release manifest assigns %s@%d to family %q, family manifest says %q", key.TaskID, key.Revision, task.TaskFamilyKey, family.Key)
			}
			if task.Language != revision.Language || task.Profile != revision.Profile {
				return nil, contracts.TaskFamilies{}, fmt.Errorf("family %q revision %s@%d disagrees with descriptor language/profile", family.Key, key.TaskID, key.Revision)
			}
			actualHash, err := hashTaskRevision(filepath.Join(tasksRoot, task.ID))
			if err != nil {
				return nil, contracts.TaskFamilies{}, fmt.Errorf("hash task revision %s@%d: %w", key.TaskID, key.Revision, err)
			}
			if actualHash != revision.ImmutableHash {
				return nil, contracts.TaskFamilies{}, fmt.Errorf("task revision %s@%d immutable hash mismatch: got %s want %s", key.TaskID, key.Revision, actualHash, revision.ImmutableHash)
			}
			task.TaskFamilyKey = family.Key
			task.ImmutableHash = revision.ImmutableHash
			task.Availability = revision.Availability
			task.Runnable = task.Runnable && revision.Availability == "runnable"
			for _, binding := range task.QuestionBindings {
				bindingKey := binding.StableKey + "|" + binding.RevisionID + "|" + binding.ContentHash
				if _, exists := bindingSeen[bindingKey]; !exists {
					bindingSeen[bindingKey] = struct{}{}
					projection.QuestionBindings = append(projection.QuestionBindings, binding)
				}
			}
			projection.Revisions = append(projection.Revisions, contracts.TaskFamilyRevision{
				TaskID: task.ID, Revision: task.Revision, TaskFamilyKey: family.Key,
				Language: task.Language, Profile: task.Profile, Runtime: task.Runtime,
				Status: task.Status, Availability: task.Availability, Runnable: task.Runnable,
				ImmutableHash: task.ImmutableHash,
			})
		}
		for _, revision := range projection.Revisions {
			projection.RevisionIDs = append(projection.RevisionIDs, revision.TaskID)
			if revision.Runnable {
				projection.Runnable = true
				break
			}
		}
		result.Families = append(result.Families, projection)
	}
	for _, task := range tasks {
		if task.Status == "released" {
			key := taskFamilyRevisionKey{TaskID: task.ID, Revision: task.Revision}
			if _, exists := seen[key]; !exists {
				return nil, contracts.TaskFamilies{}, fmt.Errorf("released task revision %s@%d has no TaskFamily", key.TaskID, key.Revision)
			}
		}
	}
	return tasks, result, nil
}

func hashTaskRevision(root string) (string, error) {
	var paths []string
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink is not allowed in task revision")
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(relative))
		return nil
	}); err != nil {
		return "", err
	}
	sort.Strings(paths)
	hash := sha256.New()
	var length [8]byte
	for _, relative := range paths {
		name := []byte(relative)
		binary.BigEndian.PutUint64(length[:], uint64(len(name)))
		_, _ = hash.Write(length[:])
		_, _ = hash.Write(name)
		body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			return "", err
		}
		binary.BigEndian.PutUint64(length[:], uint64(len(body)))
		_, _ = hash.Write(length[:])
		_, _ = hash.Write(body)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func cloneTaskFamilies(value contracts.TaskFamilies) contracts.TaskFamilies {
	result := contracts.TaskFamilies{ContractVersion: value.ContractVersion, ReleaseID: value.ReleaseID, Families: make([]contracts.TaskFamily, len(value.Families))}
	for index, family := range value.Families {
		result.Families[index] = family
		result.Families[index].CapabilityKeys = cloneStrings(family.CapabilityKeys)
		result.Families[index].QuestionBindings = cloneQuestionBindings(family.QuestionBindings)
		result.Families[index].Revisions = append([]contracts.TaskFamilyRevision(nil), family.Revisions...)
	}
	return result
}
