package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

const (
	taskPortfolioContract = "fluent-task-runtime.task-portfolio.v1"
	targetFamilyCount     = 168
	targetRevisionCount   = 456
)

type taskPortfolioManifest struct {
	ContractVersion           string                      `json:"contractVersion"`
	PortfolioID               string                      `json:"portfolioId"`
	SourceTaskFamilyReleaseID string                      `json:"sourceTaskFamilyReleaseId"`
	Target                    taskPortfolioTarget         `json:"target"`
	CompatibilitySets         []portfolioCompatibilitySet `json:"compatibilitySets"`
	Groups                    []taskPortfolioGroup        `json:"groups"`
	SupersededFamilies        []portfolioSupersededFamily `json:"supersededFamilies"`
}

type taskPortfolioTarget struct {
	FamilyCount   int                       `json:"familyCount"`
	RevisionCount int                       `json:"revisionCount"`
	Kinds         []taskPortfolioKindTarget `json:"kinds"`
}

type taskPortfolioKindTarget struct {
	Kind          string `json:"kind"`
	FamilyCount   int    `json:"familyCount"`
	RevisionCount int    `json:"revisionCount"`
}

type portfolioCompatibilitySet struct {
	ID        string                           `json:"id"`
	Revisions []portfolioRevisionCompatibility `json:"revisions"`
}

type portfolioRevisionCompatibility struct {
	Language string `json:"language"`
	Profile  string `json:"profile"`
}

type taskPortfolioGroup struct {
	ID                       string                    `json:"id"`
	Kind                     string                    `json:"kind"`
	NativeTrack              string                    `json:"nativeTrack,omitempty"`
	CompatibilitySet         string                    `json:"compatibilitySet"`
	TargetFamilyCount        int                       `json:"targetFamilyCount"`
	TargetRevisionsPerFamily int                       `json:"targetRevisionsPerFamily"`
	ExistingFamilies         []portfolioExistingFamily `json:"existingFamilies"`
	PlannedSeries            portfolioPlannedSeries    `json:"plannedSeries"`
}

type portfolioExistingFamily struct {
	FamilyKey   string `json:"familyKey"`
	Disposition string `json:"disposition"`
}

type portfolioPlannedSeries struct {
	Prefix      string `json:"prefix"`
	Start       int    `json:"start"`
	Count       int    `json:"count"`
	Disposition string `json:"disposition"`
}

type portfolioSupersededFamily struct {
	FamilyKey        string `json:"familyKey"`
	Disposition      string `json:"disposition"`
	ReplacementOwner string `json:"replacementOwner"`
	Reason           string `json:"reason"`
}

type portfolioKindCount struct {
	Families  int
	Revisions int
}

var expectedPortfolioKinds = map[string]portfolioKindCount{
	"algorithm":      {Families: 60, Revisions: 300},
	"shared_backend": {Families: 12, Revisions: 60},
	"native":         {Families: 80, Revisions: 80},
	"postgresql":     {Families: 16, Revisions: 16},
}

var expectedCompatibilitySets = map[string][]portfolioRevisionCompatibility{
	"cross-language-five": {
		{Language: "javascript", Profile: "node"},
		{Language: "typescript", Profile: "node"},
		{Language: "java", Profile: "java"},
		{Language: "go", Profile: "go"},
		{Language: "csharp", Profile: "dotnet"},
	},
	"native-node":    {{Language: "javascript", Profile: "node"}},
	"native-java":    {{Language: "java", Profile: "java"}},
	"native-go":      {{Language: "go", Profile: "go"}},
	"native-dotnet":  {{Language: "csharp", Profile: "dotnet"}},
	"native-vue":     {{Language: "typescript", Profile: "browser"}},
	"postgresql-sql": {{Language: "sql", Profile: "postgres"}},
}

var allowedPortfolioProfiles = map[string]struct{}{
	"browser":  {},
	"dotnet":   {},
	"go":       {},
	"java":     {},
	"node":     {},
	"postgres": {},
}

func loadTaskPortfolioManifest(portfolioPath, familyManifestPath string) (*taskPortfolioManifest, error) {
	portfolioBody, err := os.ReadFile(portfolioPath)
	if err != nil {
		return nil, fmt.Errorf("read task portfolio manifest %q: %w", portfolioPath, err)
	}
	var portfolio taskPortfolioManifest
	decoder := json.NewDecoder(strings.NewReader(string(portfolioBody)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&portfolio); err != nil {
		return nil, fmt.Errorf("decode task portfolio manifest %q: %w", portfolioPath, err)
	}

	familyBody, err := os.ReadFile(familyManifestPath)
	if err != nil {
		return nil, fmt.Errorf("read task family manifest %q: %w", familyManifestPath, err)
	}
	var families taskFamilyManifest
	decoder = json.NewDecoder(strings.NewReader(string(familyBody)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&families); err != nil {
		return nil, fmt.Errorf("decode task family manifest %q: %w", familyManifestPath, err)
	}
	if err := validateTaskFamilyManifest(families); err != nil {
		return nil, fmt.Errorf("task family manifest %q: %w", familyManifestPath, err)
	}
	if err := validateTaskPortfolioManifest(portfolio, families); err != nil {
		return nil, fmt.Errorf("task portfolio manifest %q: %w", portfolioPath, err)
	}
	return &portfolio, nil
}

func validateTaskPortfolioManifest(portfolio taskPortfolioManifest, families taskFamilyManifest) error {
	if portfolio.ContractVersion != taskPortfolioContract {
		return fmt.Errorf("unsupported contractVersion %q", portfolio.ContractVersion)
	}
	if strings.TrimSpace(portfolio.PortfolioID) == "" {
		return fmt.Errorf("portfolioId is required")
	}
	if portfolio.SourceTaskFamilyReleaseID != families.ReleaseID {
		return fmt.Errorf("source TaskFamily release mismatch: portfolio pins %q, loaded %q", portfolio.SourceTaskFamilyReleaseID, families.ReleaseID)
	}
	if portfolio.Target.FamilyCount != targetFamilyCount || portfolio.Target.RevisionCount != targetRevisionCount {
		return fmt.Errorf("target count drift: got %d families/%d revisions, want %d/%d", portfolio.Target.FamilyCount, portfolio.Target.RevisionCount, targetFamilyCount, targetRevisionCount)
	}
	if err := validatePortfolioKindTargets(portfolio.Target.Kinds); err != nil {
		return err
	}

	compatibility, err := validatePortfolioCompatibilitySets(portfolio.CompatibilitySets)
	if err != nil {
		return err
	}

	current := make(map[string]taskFamilyEntry, len(families.Families))
	for _, family := range families.Families {
		current[family.Key] = family
	}
	classified := make(map[string]string, targetFamilyCount+len(portfolio.SupersededFamilies))
	seenGroups := make(map[string]struct{}, len(portfolio.Groups))
	actualKinds := make(map[string]portfolioKindCount, len(expectedPortfolioKinds))
	nativeTrackCounts := make(map[string]int, 5)

	for _, group := range portfolio.Groups {
		if strings.TrimSpace(group.ID) == "" {
			return fmt.Errorf("group id is required")
		}
		if _, duplicate := seenGroups[group.ID]; duplicate {
			return fmt.Errorf("duplicate group id %q", group.ID)
		}
		seenGroups[group.ID] = struct{}{}
		_, known := expectedPortfolioKinds[group.Kind]
		if !known {
			return fmt.Errorf("group %q has unknown kind %q", group.ID, group.Kind)
		}
		set, exists := compatibility[group.CompatibilitySet]
		if !exists {
			return fmt.Errorf("group %q references unknown compatibility set %q", group.ID, group.CompatibilitySet)
		}
		if err := validateGroupCompatibility(group); err != nil {
			return err
		}
		if group.TargetFamilyCount < 1 || group.TargetRevisionsPerFamily != len(set) {
			return fmt.Errorf("group %q target count drift: %d families x %d revisions does not match compatibility set size %d", group.ID, group.TargetFamilyCount, group.TargetRevisionsPerFamily, len(set))
		}
		if group.PlannedSeries.Count < 0 || (group.PlannedSeries.Count > 0 && group.PlannedSeries.Start < 1) {
			return fmt.Errorf("group %q has invalid planned series range", group.ID)
		}
		if group.PlannedSeries.Count > 0 && group.PlannedSeries.Disposition != "new" {
			return fmt.Errorf("group %q planned series must have new disposition", group.ID)
		}
		if len(group.ExistingFamilies)+group.PlannedSeries.Count != group.TargetFamilyCount {
			return fmt.Errorf("group %q family count drift: %d existing + %d new != target %d", group.ID, len(group.ExistingFamilies), group.PlannedSeries.Count, group.TargetFamilyCount)
		}

		for _, existing := range group.ExistingFamilies {
			if existing.Disposition != "existing" {
				return fmt.Errorf("family %q in group %q must have existing disposition", existing.FamilyKey, group.ID)
			}
			family, exists := current[existing.FamilyKey]
			if !exists {
				return fmt.Errorf("existing family %q in group %q is absent from pinned release", existing.FamilyKey, group.ID)
			}
			if family.Status != "released" {
				return fmt.Errorf("existing family %q in group %q is not released", existing.FamilyKey, group.ID)
			}
			if err := classifyPortfolioFamily(classified, existing.FamilyKey, "existing:"+group.ID); err != nil {
				return err
			}
			if err := validateExistingFamilyCompatibility(family, set); err != nil {
				return fmt.Errorf("group %q: %w", group.ID, err)
			}
		}
		for offset := 0; offset < group.PlannedSeries.Count; offset++ {
			key := fmt.Sprintf("%s%03d", group.PlannedSeries.Prefix, group.PlannedSeries.Start+offset)
			if !taskFamilyKeyPattern.MatchString(key) {
				return fmt.Errorf("group %q generates invalid family key %q", group.ID, key)
			}
			if _, exists := current[key]; exists {
				return fmt.Errorf("new family %q in group %q already exists in pinned release", key, group.ID)
			}
			if err := classifyPortfolioFamily(classified, key, "new:"+group.ID); err != nil {
				return err
			}
		}

		kindCount := actualKinds[group.Kind]
		kindCount.Families += group.TargetFamilyCount
		kindCount.Revisions += group.TargetFamilyCount * group.TargetRevisionsPerFamily
		actualKinds[group.Kind] = kindCount
		if group.Kind == "native" {
			nativeTrackCounts[group.NativeTrack] += group.TargetFamilyCount
		}
	}

	for _, superseded := range portfolio.SupersededFamilies {
		if superseded.Disposition != "superseded" {
			return fmt.Errorf("family %q must have superseded disposition", superseded.FamilyKey)
		}
		family, exists := current[superseded.FamilyKey]
		if !exists {
			return fmt.Errorf("superseded family %q is absent from pinned release", superseded.FamilyKey)
		}
		if family.Status == "released" {
			return fmt.Errorf("released family %q cannot be superseded by the portfolio", superseded.FamilyKey)
		}
		if strings.TrimSpace(superseded.ReplacementOwner) == "" || strings.TrimSpace(superseded.Reason) == "" {
			return fmt.Errorf("superseded family %q requires replacementOwner and reason", superseded.FamilyKey)
		}
		if err := classifyPortfolioFamily(classified, superseded.FamilyKey, "superseded"); err != nil {
			return err
		}
	}

	for kind, expected := range expectedPortfolioKinds {
		if actualKinds[kind] != expected {
			actual := actualKinds[kind]
			return fmt.Errorf("kind %q count drift: got %d families/%d revisions, want %d/%d", kind, actual.Families, actual.Revisions, expected.Families, expected.Revisions)
		}
	}
	for _, track := range []string{"node", "java", "go", "dotnet", "vue"} {
		if nativeTrackCounts[track] != 16 {
			return fmt.Errorf("native track %q count drift: got %d families, want 16", track, nativeTrackCounts[track])
		}
	}
	if len(nativeTrackCounts) != 5 {
		return fmt.Errorf("native track count drift: got tracks %v", sortedMapKeys(nativeTrackCounts))
	}
	for key := range current {
		if _, accounted := classified[key]; !accounted {
			return fmt.Errorf("pinned TaskFamily %q is unclassified", key)
		}
	}
	return nil
}

func validatePortfolioKindTargets(targets []taskPortfolioKindTarget) error {
	seen := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		if _, duplicate := seen[target.Kind]; duplicate {
			return fmt.Errorf("duplicate target kind %q", target.Kind)
		}
		seen[target.Kind] = struct{}{}
		expected, exists := expectedPortfolioKinds[target.Kind]
		if !exists {
			return fmt.Errorf("unknown target kind %q", target.Kind)
		}
		if target.FamilyCount != expected.Families || target.RevisionCount != expected.Revisions {
			return fmt.Errorf("target kind %q count drift: got %d/%d, want %d/%d", target.Kind, target.FamilyCount, target.RevisionCount, expected.Families, expected.Revisions)
		}
	}
	if len(seen) != len(expectedPortfolioKinds) {
		return fmt.Errorf("target kind count drift: got %d kinds, want %d", len(seen), len(expectedPortfolioKinds))
	}
	return nil
}

func validatePortfolioCompatibilitySets(sets []portfolioCompatibilitySet) (map[string][]portfolioRevisionCompatibility, error) {
	result := make(map[string][]portfolioRevisionCompatibility, len(sets))
	for _, set := range sets {
		if _, duplicate := result[set.ID]; duplicate {
			return nil, fmt.Errorf("duplicate compatibility set %q", set.ID)
		}
		expected, exists := expectedCompatibilitySets[set.ID]
		if !exists {
			return nil, fmt.Errorf("unknown compatibility set %q", set.ID)
		}
		seenPairs := make(map[string]struct{}, len(set.Revisions))
		for _, revision := range set.Revisions {
			profile := strings.ToLower(strings.TrimSpace(revision.Profile))
			language := strings.ToLower(strings.TrimSpace(revision.Language))
			if _, allowed := allowedPortfolioProfiles[profile]; !allowed {
				return nil, fmt.Errorf("compatibility set %q uses foreign profile %q", set.ID, revision.Profile)
			}
			pair := language + "/" + profile
			if _, duplicate := seenPairs[pair]; duplicate {
				return nil, fmt.Errorf("compatibility set %q repeats %q", set.ID, pair)
			}
			seenPairs[pair] = struct{}{}
		}
		if !sameCompatibilitySet(set.Revisions, expected) {
			return nil, fmt.Errorf("compatibility set %q has foreign language/profile pairs", set.ID)
		}
		result[set.ID] = set.Revisions
	}
	if len(result) != len(expectedCompatibilitySets) {
		return nil, fmt.Errorf("compatibility set count drift: got %d, want %d", len(result), len(expectedCompatibilitySets))
	}
	return result, nil
}

func validateGroupCompatibility(group taskPortfolioGroup) error {
	if group.Kind == "native" {
		expectedSet := "native-" + group.NativeTrack
		if group.NativeTrack == "" || group.CompatibilitySet != expectedSet {
			return fmt.Errorf("native group %q must bind its native-%s compatibility set", group.ID, group.NativeTrack)
		}
		return nil
	}
	if group.NativeTrack != "" {
		return fmt.Errorf("non-native group %q cannot declare nativeTrack", group.ID)
	}
	expectedSet := "cross-language-five"
	if group.Kind == "postgresql" {
		expectedSet = "postgresql-sql"
	}
	if group.CompatibilitySet != expectedSet {
		return fmt.Errorf("group %q kind %q must use compatibility set %q", group.ID, group.Kind, expectedSet)
	}
	return nil
}

func validateExistingFamilyCompatibility(family taskFamilyEntry, compatibility []portfolioRevisionCompatibility) error {
	allowed := make(map[string]struct{}, len(compatibility))
	for _, revision := range compatibility {
		allowed[strings.ToLower(revision.Language)+"/"+strings.ToLower(revision.Profile)] = struct{}{}
	}
	for _, revision := range family.Revisions {
		pair := strings.ToLower(strings.TrimSpace(revision.Language)) + "/" + strings.ToLower(strings.TrimSpace(revision.Profile))
		if _, exists := allowed[pair]; !exists {
			return fmt.Errorf("existing family %q revision %s@%d has foreign compatibility %q", family.Key, revision.TaskID, revision.Revision, pair)
		}
	}
	return nil
}

func classifyPortfolioFamily(classified map[string]string, key, disposition string) error {
	if !taskFamilyKeyPattern.MatchString(key) {
		return fmt.Errorf("invalid family key %q", key)
	}
	if previous, duplicate := classified[key]; duplicate {
		return fmt.Errorf("duplicate family id %q classified as %s and %s", key, previous, disposition)
	}
	classified[key] = disposition
	return nil
}

func sameCompatibilitySet(actual, expected []portfolioRevisionCompatibility) bool {
	if len(actual) != len(expected) {
		return false
	}
	canonical := func(values []portfolioRevisionCompatibility) []string {
		pairs := make([]string, 0, len(values))
		for _, value := range values {
			pairs = append(pairs, strings.ToLower(strings.TrimSpace(value.Language))+"/"+strings.ToLower(strings.TrimSpace(value.Profile)))
		}
		sort.Strings(pairs)
		return pairs
	}
	left, right := canonical(actual), canonical(expected)
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func sortedMapKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
