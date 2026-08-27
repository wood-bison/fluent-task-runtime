package engine

import (
	"strings"
	"testing"

	"github.com/wood-bison/fluent-task-runtime/contracts"
)

func TestSanitizeErrorRedactsRuntimeMountPaths(t *testing.T) {
	input := `/hidden-tests/tests/rate-limiter.test.js:5:3\n` +
		`at require (/solution/rate-limiter.js:1:7)\n` +
		`wrote /output/results.json`
	output := sanitizeError(input)
	for _, secret := range []string{"/hidden-tests", "/solution", "/output", "rate-limiter.test.js"} {
		if strings.Contains(output, secret) {
			t.Fatalf("sanitized diagnostics still contain %q: %q", secret, output)
		}
	}
	if !strings.Contains(output, "<private>:5:3") || !strings.Contains(output, "<submission>:1:7") {
		t.Fatalf("sanitized diagnostics lost useful source locations: %q", output)
	}
}

func TestSanitizeErrorPreservesSafeDiagnostics(t *testing.T) {
	const input = "SyntaxError: Unexpected token ';'"
	if got := sanitizeError(input); got != input {
		t.Fatalf("safe diagnostic changed: %q", got)
	}
}

func TestSanitizeResultsDropsPrivateTestCodeAndPaths(t *testing.T) {
	results := contracts.TestResults{
		Version: 2,
		Status:  "error",
		Message: "/hidden-tests/tests/private.test.js:1:1",
		Tests: []contracts.TestResult{{
			Name:     "public check",
			Status:   "error",
			Message:  "at /solution/main.js:1:2",
			Output:   "read /output/private.log",
			TestCode: "assert.equal(secret, true)",
		}},
	}
	sanitizeResults(&results)
	if strings.Contains(results.Message, "/hidden-tests") || strings.Contains(results.Tests[0].Message, "/solution") || strings.Contains(results.Tests[0].Output, "/output") {
		t.Fatalf("private runtime paths survived result sanitization: %#v", results)
	}
	if results.Tests[0].TestCode != "" {
		t.Fatalf("private test code survived result sanitization: %#v", results.Tests[0])
	}
}
