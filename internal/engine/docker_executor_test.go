package engine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wood-bison/fluent-task-runtime/contracts"
)

const testImageDigest = "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func fakeDockerInspect(t *testing.T, output string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "docker")
	script := "#!/bin/sh\nprintf '%s' " + shellQuote(output) + "\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func TestVerifyImageDigestRejectsMutableReference(t *testing.T) {
	executor := &DockerExecutor{docker: fakeDockerInspect(t, `[]`)}
	if err := executor.verifyImageDigest(context.Background(), "fluent-runtime-task-node:1"); err == nil || !strings.Contains(err.Error(), "not an immutable") {
		t.Fatalf("mutable reference was accepted: %v", err)
	}
}

func TestVerifyImageDigestRequiresExactInspectedReference(t *testing.T) {
	image := "fluent-runtime-task-node@" + testImageDigest
	executor := &DockerExecutor{docker: fakeDockerInspect(t, `["fluent-runtime-task-node@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"]`)}
	if err := executor.verifyImageDigest(context.Background(), image); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("mismatched digest was accepted: %v", err)
	}
}

func TestVerifyImageDigestAcceptsExactInspectedReference(t *testing.T) {
	image := "fluent-runtime-task-node@" + testImageDigest
	executor := &DockerExecutor{docker: fakeDockerInspect(t, `["fluent-runtime-task-node@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"]`)}
	if err := executor.verifyImageDigest(context.Background(), image); err != nil {
		t.Fatalf("exact digest was rejected: %v", err)
	}
}

func TestDockerArgsEnforceResourceAndIsolationLimits(t *testing.T) {
	task := contracts.Task{
		ID:       "bounded-task",
		Image:    "fluent-runtime-task-node@" + testImageDigest,
		MemoryMB: 256,
		CPUs:     1.5,
	}
	args := dockerArgs(task, "/tmp/solution", "/tmp/hidden", "/tmp/output", "audit-run")
	joined := strings.Join(args, " ")
	for _, expected := range []string{
		"--network none",
		"--memory 256m",
		"--memory-swap 256m",
		"--cpus 1.5",
		"--pids-limit 256",
		"--cap-drop ALL",
		"--security-opt no-new-privileges",
		"--read-only",
		"--log-driver none",
		"--pull never",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("docker run args missing %q: %s", expected, joined)
		}
	}
}
