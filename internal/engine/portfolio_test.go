package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTaskPortfolioManifestProductionClosure(t *testing.T) {
	portfolioPath, familiesPath := taskPortfolioFixturePaths()
	portfolio, err := loadTaskPortfolioManifest(portfolioPath, familiesPath)
	if err != nil {
		t.Fatalf("load production portfolio: %v", err)
	}

	existing, planned := 0, 0
	for _, group := range portfolio.Groups {
		existing += len(group.ExistingFamilies)
		planned += group.PlannedSeries.Count
	}
	if existing != 15 || planned != 153 {
		t.Fatalf("unexpected closure split: got %d existing/%d new, want 15/153", existing, planned)
	}
	if len(portfolio.SupersededFamilies) != 1 {
		t.Fatalf("unexpected superseded count: got %d, want 1", len(portfolio.SupersededFamilies))
	}
	if portfolio.Target.FamilyCount != 168 || portfolio.Target.RevisionCount != 456 {
		t.Fatalf("unexpected target: got %d families/%d revisions", portfolio.Target.FamilyCount, portfolio.Target.RevisionCount)
	}
}

func TestTaskPortfolioManifestRejectsDuplicateFamilyID(t *testing.T) {
	portfolio, families := readTaskPortfolioFixtures(t)
	portfolio.Groups[0].ExistingFamilies[0].FamilyKey = "task-family.rate-limiter"

	err := validateTaskPortfolioManifest(portfolio, families)
	if err == nil || !strings.Contains(err.Error(), "duplicate family id") {
		t.Fatalf("expected duplicate family id error, got %v", err)
	}
}

func TestTaskPortfolioManifestRejectsForeignProfile(t *testing.T) {
	portfolio, families := readTaskPortfolioFixtures(t)
	portfolio.CompatibilitySets[0].Revisions[0].Profile = "python"

	err := validateTaskPortfolioManifest(portfolio, families)
	if err == nil || !strings.Contains(err.Error(), "foreign profile") {
		t.Fatalf("expected foreign profile error, got %v", err)
	}
}

func TestTaskPortfolioManifestRejectsCountDrift(t *testing.T) {
	portfolio, families := readTaskPortfolioFixtures(t)
	portfolio.Target.FamilyCount--

	err := validateTaskPortfolioManifest(portfolio, families)
	if err == nil || !strings.Contains(err.Error(), "target count drift") {
		t.Fatalf("expected target count drift error, got %v", err)
	}
}

func TestTaskPortfolioManifestRejectsUnclassifiedPinnedFamily(t *testing.T) {
	portfolio, families := readTaskPortfolioFixtures(t)
	portfolio.SupersededFamilies = nil

	err := validateTaskPortfolioManifest(portfolio, families)
	if err == nil || !strings.Contains(err.Error(), "unclassified") {
		t.Fatalf("expected unclassified family error, got %v", err)
	}
}

func TestTaskPortfolioManifestRejectsSourceReleaseMismatch(t *testing.T) {
	portfolio, families := readTaskPortfolioFixtures(t)
	portfolio.SourceTaskFamilyReleaseID = "task-family-release-foreign"

	err := validateTaskPortfolioManifest(portfolio, families)
	if err == nil || !strings.Contains(err.Error(), "release mismatch") {
		t.Fatalf("expected release mismatch error, got %v", err)
	}
}

func TestTaskPortfolioBacklogEnumeratesExactFamilyAndRevisionGap(t *testing.T) {
	portfolioPath, familiesPath := taskPortfolioFixturePaths()
	first, err := BuildTaskPortfolioBacklog(portfolioPath, familiesPath)
	if err != nil {
		t.Fatalf("build portfolio backlog: %v", err)
	}
	second, err := BuildTaskPortfolioBacklog(portfolioPath, familiesPath)
	if err != nil {
		t.Fatalf("build portfolio backlog twice: %v", err)
	}
	if first.Summary.OpenFamilyItems != 153 || first.Summary.OpenRevisionItems != 437 || first.Summary.OpenItemCount != 590 {
		t.Fatalf("unexpected portfolio backlog counts: %+v", first.Summary)
	}
	if first.ContentDigest != second.ContentDigest {
		t.Fatalf("portfolio backlog digest is not deterministic: %s != %s", first.ContentDigest, second.ContentDigest)
	}
	seen := make(map[string]struct{}, len(first.Items))
	for _, item := range first.Items {
		if _, duplicate := seen[item.ItemID]; duplicate {
			t.Fatalf("duplicate backlog item %q", item.ItemID)
		}
		seen[item.ItemID] = struct{}{}
		if item.Status != "open" || len(item.Acceptance) == 0 {
			t.Fatalf("incomplete backlog item %+v", item)
		}
	}
	if first.Items[0].FamilyKey != "task-family.fluent-calculator" || first.Items[0].Kind != "revision" {
		t.Fatalf("expected existing family compatibility gaps first, got %+v", first.Items[0])
	}
	if first.Items[0].Language != "typescript" || first.Items[0].Profile != "node" {
		t.Fatalf("expected first missing cross-language revision, got %+v", first.Items[0])
	}
	if first.Items[0].Wave != 1 || first.Items[0].BatchPosition != 1 {
		t.Fatalf("expected first item in wave one, got %+v", first.Items[0])
	}
}

func taskPortfolioFixturePaths() (string, string) {
	return filepath.Join("..", "..", "task-portfolio", "manifest.json"), filepath.Join("..", "..", "task-families", "manifest.json")
}

func readTaskPortfolioFixtures(t *testing.T) (taskPortfolioManifest, taskFamilyManifest) {
	t.Helper()
	portfolioPath, familiesPath := taskPortfolioFixturePaths()
	var portfolio taskPortfolioManifest
	readJSONFixture(t, portfolioPath, &portfolio)
	var families taskFamilyManifest
	readJSONFixture(t, familiesPath, &families)
	return portfolio, families
}

func readJSONFixture(t *testing.T, path string, target any) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %q: %v", path, err)
	}
	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("decode fixture %q: %v", path, err)
	}
}
