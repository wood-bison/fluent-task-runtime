package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wood-bison/fluent-task-runtime/internal/engine"
)

func main() {
	portfolioPath := flag.String("portfolio", "task-portfolio/manifest.json", "TaskPortfolioManifest path")
	familyPath := flag.String("families", "task-families/manifest.json", "TaskFamily manifest path")
	outPath := flag.String("out", "task-portfolio/backlog-2026-08-27.json", "backlog JSON output path")
	markdownPath := flag.String("md", "task-portfolio/backlog-2026-08-27.md", "backlog Markdown output path")
	check := flag.Bool("check", false, "compare the generated digest with the stored report")
	flag.Parse()

	report, err := engine.BuildTaskPortfolioBacklog(*portfolioPath, *familyPath)
	if err != nil {
		fatal(err)
	}
	if *check {
		storedBody, readErr := os.ReadFile(*outPath)
		if readErr != nil {
			fatal(fmt.Errorf("read stored backlog %q: %w", *outPath, readErr))
		}
		var stored engine.PortfolioBacklogReport
		if err := json.Unmarshal(storedBody, &stored); err != nil {
			fatal(fmt.Errorf("decode stored backlog %q: %w", *outPath, err))
		}
		if stored.ReportVersion != report.ReportVersion || stored.ContentDigest != report.ContentDigest || stored.ProductionReady {
			fatal(fmt.Errorf("portfolio backlog drift: stored version=%s digest=%s productionReady=%t, generated version=%s digest=%s", stored.ReportVersion, stored.ContentDigest, stored.ProductionReady, report.ReportVersion, report.ContentDigest))
		}
		fmt.Printf("{\"valid\":true,\"contentDigest\":%q,\"openItemCount\":%d,\"openFamilyItems\":%d,\"openRevisionItems\":%d}\n", report.ContentDigest, report.Summary.OpenItemCount, report.Summary.OpenFamilyItems, report.Summary.OpenRevisionItems)
		return
	}
	report.GeneratedAt = time.Now().UTC().Format(time.RFC3339Nano)
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fatal(fmt.Errorf("encode backlog: %w", err))
	}
	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		fatal(fmt.Errorf("create output directory: %w", err))
	}
	if err := os.WriteFile(*outPath, append(body, '\n'), 0o644); err != nil {
		fatal(fmt.Errorf("write backlog %q: %w", *outPath, err))
	}
	if err := os.WriteFile(*markdownPath, []byte(renderMarkdown(report)), 0o644); err != nil {
		fatal(fmt.Errorf("write backlog markdown %q: %w", *markdownPath, err))
	}
	fmt.Printf("{\"valid\":true,\"output\":%q,\"contentDigest\":%q,\"openItemCount\":%d,\"openFamilyItems\":%d,\"openRevisionItems\":%d}\n", *outPath, report.ContentDigest, report.Summary.OpenItemCount, report.Summary.OpenFamilyItems, report.Summary.OpenRevisionItems)
}

func renderMarkdown(report engine.PortfolioBacklogReport) string {
	lines := []string{
		"# Task portfolio authoring backlog",
		"",
		fmt.Sprintf("Portfolio: `%s`", report.SourcePortfolioID),
		"Status: **OPEN**; `productionReady=false`",
		"",
		"Это answer-free очередь для закрытия TaskPortfolio target. Она не создаёт starter, hidden test или verdict и не изменяет опубликованные revisions.",
		"",
		fmt.Sprintf("- Open items: **%d**; family items: **%d**; revision items: **%d**; bounded waves: **%d**.", report.Summary.OpenItemCount, report.Summary.OpenFamilyItems, report.Summary.OpenRevisionItems, report.Summary.WaveCount),
		"- Batch size: **100**; auto-publish: **нет**; filler: **запрещён**.",
		"",
		"## По группам",
		"",
		"| Group | Kind | Existing | Planned | Open family | Open revisions |",
		"| --- | --- | ---: | ---: | ---: | ---: |",
	}
	for _, group := range report.Groups {
		lines = append(lines, fmt.Sprintf("| `%s` | `%s` | %d | %d | %d | %d |", group.GroupID, group.Kind, group.ExistingFamilyCount, group.PlannedFamilyCount, group.OpenFamilyItems, group.OpenRevisionItems))
	}
	lines = append(lines,
		"",
		"## Порядок",
		"",
		"1. Дособрать compatibility revisions для уже существующих семейств.",
		"2. Авторить новые TaskFamily с EN/RU brief, rubric и capability outcome.",
		"3. Авторить revisions только для объявленной language/profile compatibility set.",
		"4. Прогнать deterministic pass/fail/error, sandbox security и failure-injection; затем выпустить immutable runtime release.",
		"",
		"## Reproduction",
		"",
		"```bash",
		"cd /Users/sergeyzhechko/developer/fluent-interview/fluent-task-runtime",
		"docker run --rm -v \"$PWD\":/src -w /src golang:1.24 go run ./cmd/portfolio-backlog",
		"docker run --rm -v \"$PWD\":/src -w /src golang:1.24 go run ./cmd/portfolio-backlog --check",
		"```",
		"",
		fmt.Sprintf("Stable content digest: `%s`", report.ContentDigest),
		"",
	)
	return strings.Join(lines, "\n")
}

func fatal(err error) {
	if err == nil {
		err = errors.New("portfolio backlog failed")
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
