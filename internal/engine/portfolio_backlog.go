package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const portfolioBacklogContract = "fluent-task-runtime.task-portfolio-backlog.v1"

// PortfolioBacklogItem is an addressable authoring/review unit. It deliberately
// carries no task prompt, starter, hidden test or answer. Those remain owned by
// the immutable TaskFamily/TaskRevision release and are added only by a
// reviewed child commit.
type PortfolioBacklogItem struct {
	ItemID        string   `json:"itemId"`
	GroupID       string   `json:"groupId"`
	Kind          string   `json:"kind"`
	Action        string   `json:"action"`
	Sequence      int      `json:"sequence"`
	Status        string   `json:"status"`
	Priority      int      `json:"priority"`
	Wave          int      `json:"wave"`
	BatchPosition int      `json:"batchPosition"`
	FamilyKey     string   `json:"familyKey,omitempty"`
	Language      string   `json:"language,omitempty"`
	Profile       string   `json:"profile,omitempty"`
	Source        string   `json:"source"`
	Acceptance    []string `json:"acceptance"`
}

type portfolioBacklogGroupSummary struct {
	GroupID             string `json:"groupId"`
	Kind                string `json:"kind"`
	TargetFamilyCount   int    `json:"targetFamilyCount"`
	ExistingFamilyCount int    `json:"existingFamilyCount"`
	PlannedFamilyCount  int    `json:"plannedFamilyCount"`
	OpenFamilyItems     int    `json:"openFamilyItems"`
	OpenRevisionItems   int    `json:"openRevisionItems"`
}

// PortfolioBacklogReport is the stable, answer-free work queue for the
// TaskPortfolioManifest. GeneratedAt is intentionally not part of the digest.
type PortfolioBacklogReport struct {
	ReportVersion           string                         `json:"reportVersion"`
	ManifestID              string                         `json:"manifestId"`
	Status                  string                         `json:"status"`
	ProductionReady         bool                           `json:"productionReady"`
	SourcePortfolioID       string                         `json:"sourcePortfolioId"`
	SourceTaskFamilyRelease string                         `json:"sourceTaskFamilyReleaseId"`
	MaxOpenBatch            int                            `json:"maxOpenBatch"`
	Path                    string                         `json:"sourcePath,omitempty"`
	Summary                 PortfolioBacklogSummary        `json:"summary"`
	Groups                  []portfolioBacklogGroupSummary `json:"groups"`
	Items                   []PortfolioBacklogItem         `json:"items"`
	ContentDigest           string                         `json:"contentDigest"`
	GeneratedAt             string                         `json:"generatedAt,omitempty"`
}

type PortfolioBacklogSummary struct {
	GroupCount        int            `json:"groupCount"`
	OpenFamilyItems   int            `json:"openFamilyItems"`
	OpenRevisionItems int            `json:"openRevisionItems"`
	OpenItemCount     int            `json:"openItemCount"`
	WaveCount         int            `json:"waveCount"`
	NextBatchIDs      []string       `json:"nextBatchIds"`
	KindCounts        map[string]int `json:"kindCounts"`
}

const portfolioBacklogMaxBatch = 100

var portfolioBacklogAcceptance = map[string][]string{
	"family": {
		"EN/RU title and brief, capability keys and rubric are reviewed",
		"family receives an immutable release entry with an explicit execution kind",
		"at least one compatible revision passes runtime smoke before release",
	},
	"revision": {
		"language/profile pair is exactly the compatibility set declared by the portfolio",
		"starter, hidden tests, immutable hash and resource limits are reviewed",
		"released revision passes deterministic pass/fail/error and failure-injection checks",
	},
}

// BuildTaskPortfolioBacklog validates the pinned portfolio and produces a
// deterministic queue for all family and revision deficits. It is intentionally
// read-only: authoring happens in a subsequent reviewed release.
func BuildTaskPortfolioBacklog(portfolioPath, familyManifestPath string) (PortfolioBacklogReport, error) {
	portfolio, err := loadTaskPortfolioManifest(portfolioPath, familyManifestPath)
	if err != nil {
		return PortfolioBacklogReport{}, err
	}
	families, err := readTaskFamilyManifestForBacklog(familyManifestPath)
	if err != nil {
		return PortfolioBacklogReport{}, err
	}
	familyByKey := make(map[string]taskFamilyEntry, len(families.Families))
	for _, family := range families.Families {
		familyByKey[family.Key] = family
	}
	compatibilityByID := make(map[string][]portfolioRevisionCompatibility, len(portfolio.CompatibilitySets))
	for _, set := range portfolio.CompatibilitySets {
		compatibilityByID[set.ID] = append([]portfolioRevisionCompatibility(nil), set.Revisions...)
	}

	items := make([]PortfolioBacklogItem, 0, 600)
	groups := make([]portfolioBacklogGroupSummary, 0, len(portfolio.Groups))
	for _, group := range portfolio.Groups {
		summary := portfolioBacklogGroupSummary{
			GroupID: group.ID, Kind: group.Kind,
			TargetFamilyCount:   group.TargetFamilyCount,
			ExistingFamilyCount: len(group.ExistingFamilies),
			PlannedFamilyCount:  group.PlannedSeries.Count,
		}
		compatibility := compatibilityByID[group.CompatibilitySet]
		for _, existing := range group.ExistingFamilies {
			family := familyByKey[existing.FamilyKey]
			pairs := make(map[string]taskFamilyRevisionEntry, len(family.Revisions))
			for _, revision := range family.Revisions {
				pairs[portfolioPair(revision.Language, revision.Profile)] = revision
			}
			sequence := 1
			for _, expected := range compatibility {
				pair := portfolioPair(expected.Language, expected.Profile)
				revision, found := pairs[pair]
				if found && revision.Availability == "runnable" && family.Status == "released" {
					continue
				}
				items = append(items, newPortfolioBacklogItem(
					group.ID, "revision", "complete-compatible-revision", sequence,
					family.Key, expected.Language, expected.Profile,
					"existing-family-release", 1,
				))
				sequence++
				summary.OpenRevisionItems++
			}
		}
		for offset := 0; offset < group.PlannedSeries.Count; offset++ {
			familySequence := offset + 1
			familyKey := fmt.Sprintf("%s%03d", group.PlannedSeries.Prefix, group.PlannedSeries.Start+offset)
			items = append(items, newPortfolioBacklogItem(
				group.ID, "family", "author-task-family", familySequence,
				familyKey, "", "", "planned-family-series", 2,
			))
			summary.OpenFamilyItems++
			for revisionSequence, expected := range compatibility {
				items = append(items, newPortfolioBacklogItem(
					group.ID, "revision", "author-task-revision", revisionSequence+1,
					familyKey, expected.Language, expected.Profile,
					"planned-family-series", 3,
				))
				summary.OpenRevisionItems++
			}
		}
		groups = append(groups, summary)
	}

	sort.SliceStable(items, func(left, right int) bool {
		if items[left].Priority != items[right].Priority {
			return items[left].Priority < items[right].Priority
		}
		leftGroup := groupOrder(portfolio.Groups, items[left].GroupID)
		rightGroup := groupOrder(portfolio.Groups, items[right].GroupID)
		if leftGroup != rightGroup {
			return leftGroup < rightGroup
		}
		if items[left].FamilyKey != items[right].FamilyKey {
			return items[left].FamilyKey < items[right].FamilyKey
		}
		if items[left].Kind != items[right].Kind {
			return items[left].Kind < items[right].Kind
		}
		return items[left].Sequence < items[right].Sequence
	})
	for index := range items {
		items[index].Wave = index/portfolioBacklogMaxBatch + 1
		items[index].BatchPosition = index%portfolioBacklogMaxBatch + 1
	}
	kindCounts := map[string]int{}
	openFamilyItems, openRevisionItems := 0, 0
	for _, item := range items {
		kindCounts[item.Kind]++
		if item.Kind == "family" {
			openFamilyItems++
		} else if item.Kind == "revision" {
			openRevisionItems++
		}
	}
	stable := struct {
		ContractVersion           string                         `json:"contractVersion"`
		ManifestID                string                         `json:"manifestId"`
		SourcePortfolioID         string                         `json:"sourcePortfolioId"`
		SourceTaskFamilyReleaseID string                         `json:"sourceTaskFamilyReleaseId"`
		MaxOpenBatch              int                            `json:"maxOpenBatch"`
		Groups                    []portfolioBacklogGroupSummary `json:"groups"`
		Items                     []PortfolioBacklogItem         `json:"items"`
	}{
		ContractVersion:           portfolioBacklogContract,
		ManifestID:                portfolio.PortfolioID,
		SourcePortfolioID:         portfolio.PortfolioID,
		SourceTaskFamilyReleaseID: portfolio.SourceTaskFamilyReleaseID,
		MaxOpenBatch:              portfolioBacklogMaxBatch,
		Groups:                    groups,
		Items:                     items,
	}
	stableBody, err := json.Marshal(stable)
	if err != nil {
		return PortfolioBacklogReport{}, fmt.Errorf("marshal portfolio backlog digest: %w", err)
	}
	hash := sha256.Sum256(stableBody)
	nextBatchIDs := make([]string, 0, portfolioBacklogMaxBatch)
	for _, item := range items {
		if len(nextBatchIDs) == portfolioBacklogMaxBatch {
			break
		}
		nextBatchIDs = append(nextBatchIDs, item.ItemID)
	}
	return PortfolioBacklogReport{
		ReportVersion:           portfolioBacklogContract,
		ManifestID:              "task-portfolio-backlog-" + strings.ReplaceAll(portfolio.PortfolioID, "/", "-"),
		Status:                  map[bool]string{true: "open", false: "empty"}[len(items) > 0],
		ProductionReady:         false,
		SourcePortfolioID:       portfolio.PortfolioID,
		SourceTaskFamilyRelease: portfolio.SourceTaskFamilyReleaseID,
		MaxOpenBatch:            portfolioBacklogMaxBatch,
		Summary: PortfolioBacklogSummary{
			GroupCount: len(portfolio.Groups), OpenFamilyItems: openFamilyItems,
			OpenRevisionItems: openRevisionItems, OpenItemCount: len(items),
			WaveCount:    (len(items) + portfolioBacklogMaxBatch - 1) / portfolioBacklogMaxBatch,
			NextBatchIDs: nextBatchIDs, KindCounts: kindCounts,
		},
		Groups:        groups,
		Items:         items,
		ContentDigest: hex.EncodeToString(hash[:]),
		Path:          filepath.ToSlash(portfolioPath),
	}, nil
}

func newPortfolioBacklogItem(groupID, kind, action string, sequence int, familyKey, language, profile, source string, priority int) PortfolioBacklogItem {
	identity := familyKey
	if language != "" || profile != "" {
		identity += ":" + language + "-" + profile
	}
	if identity == "" {
		identity = fmt.Sprintf("slot-%03d", sequence)
	}
	identity = strings.NewReplacer("/", "-", ":", "-", " ", "-").Replace(identity)
	item := PortfolioBacklogItem{
		ItemID:  fmt.Sprintf("backlog:task-portfolio:%s:%s:%s", groupID, kind, identity),
		GroupID: groupID, Kind: kind, Action: action, Sequence: sequence,
		Status: "open", Priority: priority, FamilyKey: familyKey,
		Language: language, Profile: profile, Source: source,
		Acceptance: append([]string(nil), portfolioBacklogAcceptance[kind]...),
	}
	return item
}

func portfolioPair(language, profile string) string {
	return strings.ToLower(strings.TrimSpace(language)) + "/" + strings.ToLower(strings.TrimSpace(profile))
}

func groupOrder(groups []taskPortfolioGroup, id string) int {
	for index, group := range groups {
		if group.ID == id {
			return index
		}
	}
	return len(groups)
}

func readTaskFamilyManifestForBacklog(path string) (taskFamilyManifest, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return taskFamilyManifest{}, fmt.Errorf("read task family manifest %q: %w", path, err)
	}
	var manifest taskFamilyManifest
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return taskFamilyManifest{}, fmt.Errorf("decode task family manifest %q: %w", path, err)
	}
	if err := validateTaskFamilyManifest(manifest); err != nil {
		return taskFamilyManifest{}, fmt.Errorf("task family manifest %q: %w", path, err)
	}
	return manifest, nil
}
