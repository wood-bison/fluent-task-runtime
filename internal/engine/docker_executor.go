package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/wood-bison/fluent-task-runtime/contracts"
)

const (
	resultsCaptureLimit     = 1 << 20
	outputCaptureLimit      = 1 << 20
	requestFilesLimit       = 32
	requestBytesLimit       = 1 << 20
	imageInspectTimeout     = 2 * time.Second
	containerCleanupTimeout = 5 * time.Second
)

var immutableImagePattern = regexp.MustCompile(`@sha256:[0-9a-f]{64}$`)

type ExecutionError struct {
	Code      string
	Message   string
	Retryable bool
	Status    int
}

func (e *ExecutionError) Error() string { return e.Message }

// RunExecutor is the narrow seam between the HTTP control plane and the
// sandbox. Tests can inject a deterministic fake; production uses Docker with
// network, filesystem and resource limits assembled below.
type RunExecutor interface {
	Run(context.Context, contracts.RunRequest) (contracts.RunResponse, error)
}

type DockerExecutor struct {
	catalogue *Catalogue
	docker    string
	tasksRoot string
}

func (e *DockerExecutor) Ready(parent context.Context) string {
	ctx, cancel := context.WithTimeout(parent, 750*time.Millisecond)
	defer cancel()
	if err := exec.CommandContext(ctx, e.docker, "version", "--format", "{{.Server.Version}}").Run(); err != nil {
		return "unavailable"
	}
	return "ready"
}

func NewDockerExecutor(catalogue *Catalogue) *DockerExecutor {
	binary := os.Getenv("RUNTIME_DOCKER_BINARY")
	if binary == "" {
		binary = "docker"
	}
	return &DockerExecutor{catalogue: catalogue, docker: binary, tasksRoot: catalogue.TasksRoot()}
}

func (e *DockerExecutor) Run(parent context.Context, request contracts.RunRequest) (contracts.RunResponse, error) {
	task, ok := e.catalogue.Task(request.TaskID, request.TaskRevision)
	if !ok {
		return contracts.RunResponse{}, &ExecutionError{Code: "unknown_task", Message: fmt.Sprintf("task %q revision %d is not in the runtime catalogue", request.TaskID, request.TaskRevision), Status: 404}
	}
	if !e.catalogue.ReleaseBound() {
		return contracts.RunResponse{}, &ExecutionError{Code: "question_release_not_bound", Message: "task execution requires an explicit runtime release manifest", Status: 503, Retryable: false}
	}
	if task.Status != "released" {
		return contracts.RunResponse{}, &ExecutionError{Code: "runtime_not_ready", Message: fmt.Sprintf("task %q revision %d is not released", request.TaskID, request.TaskRevision), Status: 501}
	}
	if len(task.CheckCommand) == 0 || len(task.EditableFiles) == 0 {
		return contracts.RunResponse{}, &ExecutionError{Code: "task_not_packaged", Message: fmt.Sprintf("task %q has no executable harness metadata", request.TaskID), Status: 503, Retryable: false}
	}
	if err := e.verifyImageDigest(parent, task.Image); err != nil {
		return contracts.RunResponse{}, &ExecutionError{Code: "sandbox_unavailable", Message: err.Error(), Status: 503, Retryable: true}
	}
	if err := validateFiles(request.Files, task.EditableFiles); err != nil {
		return contracts.RunResponse{}, &ExecutionError{Code: "invalid_request", Message: err.Error(), Status: 400}
	}

	correlationID := strings.TrimSpace(request.Correlation)
	if correlationID == "" {
		correlationID = fmt.Sprintf("runtime-%d", time.Now().UTC().UnixNano())
	}
	started := time.Now()
	workRoot := strings.TrimSpace(os.Getenv("RUNTIME_WORK_ROOT"))
	if workRoot == "" {
		workRoot = os.TempDir()
	} else if err := os.MkdirAll(workRoot, 0o755); err != nil {
		return contracts.RunResponse{}, &ExecutionError{Code: "sandbox_setup_failed", Message: "could not prepare the runtime work root", Status: 503, Retryable: true}
	}
	work, err := os.MkdirTemp(workRoot, "fluent-task-runtime-")
	if err != nil {
		return contracts.RunResponse{}, &ExecutionError{Code: "sandbox_setup_failed", Message: "could not allocate a disposable workspace", Status: 503, Retryable: true}
	}
	defer os.RemoveAll(work)
	inputDir := filepath.Join(work, "solution")
	hiddenDir := filepath.Join(work, "hidden")
	outputDir := filepath.Join(work, "output")
	for _, dir := range []string{inputDir, hiddenDir, outputDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return contracts.RunResponse{}, &ExecutionError{Code: "sandbox_setup_failed", Message: "could not prepare the disposable workspace", Status: 503, Retryable: true}
		}
	}
	// The task process may run as a non-root profile user (PostgreSQL uses
	// uid/gid 999). Output is disposable and must be writable by that user;
	// learner input and hidden tests remain read-only mounts below.
	if err := os.Chmod(outputDir, 0o777); err != nil {
		return contracts.RunResponse{}, &ExecutionError{Code: "sandbox_setup_failed", Message: "could not make the disposable output writable", Status: 503, Retryable: true}
	}
	if err := e.prepareTask(task, inputDir, hiddenDir, request.Files); err != nil {
		return contracts.RunResponse{}, &ExecutionError{Code: "sandbox_setup_failed", Message: sanitizeError(err.Error()), Status: 503, Retryable: false}
	}

	timeout := time.Duration(task.TimeoutMS) * time.Millisecond
	if timeout <= 0 || timeout > 120*time.Second {
		return contracts.RunResponse{}, &ExecutionError{Code: "invalid_task_limits", Message: fmt.Sprintf("task %q declares an invalid timeout", request.TaskID), Status: 500, Retryable: false}
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	name := dockerContainerName(task.ID, correlationID)
	args := dockerArgs(task, inputDir, hiddenDir, outputDir, correlationID)
	command := exec.CommandContext(ctx, e.docker, args...)
	stdout := &limitedBuffer{limit: outputCaptureLimit}
	stderr := &limitedBuffer{limit: outputCaptureLimit}
	command.Stdout = stdout
	command.Stderr = stderr
	err = command.Run()
	// `--rm` only removes a container after the Docker CLI receives a normal
	// completion. When context timeout kills the CLI, the daemon can keep the
	// child running. Always issue an independent, bounded cleanup request so a
	// timed-out learner run cannot leak containers or keep consuming resources.
	e.cleanupContainer(name)
	if ctx.Err() != nil {
		return contracts.RunResponse{}, &ExecutionError{Code: "timeout", Message: fmt.Sprintf("task %q exceeded its %dms wall-clock limit", request.TaskID, timeout.Milliseconds()), Status: 504, Retryable: true}
	}
	if err != nil {
		message := sanitizeError(strings.TrimSpace(stderr.String()))
		if message == "" {
			message = sanitizeError(err.Error())
		}
		if strings.Contains(strings.ToLower(message), "no such image") || strings.Contains(strings.ToLower(message), "unable to find image") {
			return contracts.RunResponse{}, &ExecutionError{Code: "sandbox_unavailable", Message: fmt.Sprintf("runtime image %q is not available locally", task.Image), Status: 503, Retryable: true}
		}
		return contracts.RunResponse{}, &ExecutionError{Code: "harness_failed", Message: fmt.Sprintf("task harness exited unsuccessfully: %s", message), Status: 502, Retryable: false}
	}

	resultsPath := filepath.Join(outputDir, "results.json")
	rawResults, readErr := readBounded(resultsPath, resultsCaptureLimit)
	if readErr != nil {
		return contracts.RunResponse{}, &ExecutionError{Code: "runner_protocol", Message: "task harness did not write results.json", Status: 502, Retryable: false}
	}
	var results contracts.TestResults
	if decodeErr := json.Unmarshal(rawResults, &results); decodeErr != nil {
		return contracts.RunResponse{}, &ExecutionError{Code: "runner_protocol", Message: "task harness wrote malformed results.json", Status: 502, Retryable: false}
	}
	if validationErr := validateResults(results); validationErr != nil {
		return contracts.RunResponse{}, &ExecutionError{Code: "runner_protocol", Message: validationErr.Error(), Status: 502, Retryable: false}
	}
	// The result document is authored by a private harness.  Sanitize every
	// learner-visible diagnostic and drop optional test source before it crosses
	// the HTTP boundary; sanitizing only Docker stderr is insufficient because a
	// Node load failure is serialized in results.message.
	sanitizeResults(&results)
	artifacts := map[string]string{}
	for _, relative := range task.Artifacts {
		if !safeRelativePath(relative) {
			return contracts.RunResponse{}, &ExecutionError{Code: "runner_protocol", Message: "task declared an unsafe artifact path", Status: 502, Retryable: false}
		}
		if body, artifactErr := readBounded(filepath.Join(outputDir, relative), 256*1024); artifactErr == nil {
			artifacts[relative] = sanitizeError(string(body))
		}
	}
	status := 0
	return contracts.RunResponse{
		ContractVersion: contracts.RunContractVersion,
		Results:         results,
		CorrelationID:   correlationID,
		DurationMS:      float64(time.Since(started).Microseconds()) / 1000,
		ExitCode:        &status,
		Stdout:          sanitizeError(stdout.String()),
		Stderr:          sanitizeError(stderr.String()),
		Artifacts:       artifacts,
	}, nil
}

// verifyImageDigest is deliberately performed immediately before preparing a
// run. A mutable local tag can point at a different image between catalogue
// load and execution; inspecting the exact immutable reference closes that
// TOCTOU window and makes a stale/missing sandbox an explicit retryable error.
func (e *DockerExecutor) verifyImageDigest(parent context.Context, image string) error {
	image = strings.TrimSpace(image)
	if !immutableImagePattern.MatchString(image) {
		return fmt.Errorf("runtime image %q is not an immutable @sha256 reference", image)
	}
	ctx, cancel := context.WithTimeout(parent, imageInspectTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, e.docker, "image", "inspect", "--format", "{{json .RepoDigests}}", image)
	stdout := &limitedBuffer{limit: 16 * 1024}
	stderr := &limitedBuffer{limit: 16 * 1024}
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("runtime image %q digest inspection timed out", image)
		}
		message := strings.TrimSpace(sanitizeError(stderr.String()))
		if message == "" {
			message = sanitizeError(err.Error())
		}
		return fmt.Errorf("runtime image %q is not available locally: %s", image, message)
	}
	var repoDigests []string
	if err := json.Unmarshal([]byte(stdout.String()), &repoDigests); err != nil {
		return fmt.Errorf("runtime image %q returned malformed digest metadata", image)
	}
	for _, repoDigest := range repoDigests {
		if strings.TrimSpace(repoDigest) == image {
			return nil
		}
	}
	return fmt.Errorf("runtime image %q digest does not match the inspected image", image)
}

func (e *DockerExecutor) prepareTask(task contracts.Task, inputDir, hiddenDir string, files map[string]string) error {
	taskDir := filepath.Join(e.tasksRoot, task.ID)
	if err := copyDirectory(filepath.Join(taskDir, "starter"), inputDir); err != nil {
		return err
	}
	for name, content := range files {
		path := filepath.Join(inputDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
	}
	if err := copyDirectory(filepath.Join(taskDir, "tests"), filepath.Join(hiddenDir, "tests")); err != nil {
		return err
	}
	// Authored tests keep their relative imports (for example
	// `../calculator.js`). The hidden mount receives a read-only mirror of the
	// submitted workspace so those imports resolve without exposing tests in
	// `/solution`.
	if err := copyDirectory(inputDir, hiddenDir); err != nil {
		return err
	}
	return nil
}

func dockerArgs(task contracts.Task, inputDir, hiddenDir, outputDir, correlationID string) []string {
	memory := task.MemoryMB
	cpus := task.CPUs
	if memory <= 0 || cpus <= 0 {
		panic("task limits must be validated before dockerArgs")
	}
	// Keep task container names in a dedicated namespace. Lab and runtime may
	// be smoke-tested on the same Docker daemon; explicit names prevent cleanup
	// and collision mistakes.
	name := dockerContainerName(task.ID, correlationID)
	// The CLI captures stdout/stderr directly, so the task container does not
	// need a Docker json-file log. Disabling it prevents a noisy or malicious
	// submission from growing an unbounded daemon-side log between `run` and
	// `--rm` cleanup.
	args := []string{"run", "--rm", "--name", name, "--pull", "never", "--log-driver", "none", "--network", "none", "--memory", fmt.Sprintf("%dm", memory), "--memory-swap", fmt.Sprintf("%dm", memory), "--cpus", fmt.Sprintf("%g", cpus), "--pids-limit", "256", "--cap-drop", "ALL", "--security-opt", "no-new-privileges", "--read-only", "--tmpfs", "/tmp:rw,noexec,nosuid,size=128m", "-v", inputDir + ":/solution:ro", "-v", hiddenDir + ":/hidden-tests:ro", "-v", outputDir + ":/output", task.Image}
	if task.User != "" {
		args = append(args[:len(args)-1], "--user", task.User, args[len(args)-1])
	}
	return append(args, task.CheckCommand...)
}

func dockerContainerName(taskID, correlationID string) string {
	return "fluent-runtime-task-" + safeToken(taskID) + "-" + safeToken(correlationID)
}

func (e *DockerExecutor) cleanupContainer(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), containerCleanupTimeout)
	defer cancel()
	// A successful run normally self-removes via --rm; rm -f is intentionally
	// idempotent and also handles the timeout/cancel path. Cleanup failures are
	// not learner-visible verdicts, but are kept in the daemon logs for ops.
	_ = exec.CommandContext(ctx, e.docker, "rm", "-f", name).Run()
}

func validateFiles(files map[string]string, editable []string) error {
	if len(files) > requestFilesLimit {
		return fmt.Errorf("at most %d files may be submitted", requestFilesLimit)
	}
	allowed := make(map[string]struct{}, len(editable))
	for _, name := range editable {
		allowed[name] = struct{}{}
	}
	total := 0
	for name, content := range files {
		if !safeRelativePath(name) {
			return fmt.Errorf("file %q is not a safe relative path", name)
		}
		if _, ok := allowed[name]; !ok {
			return fmt.Errorf("file %q is not editable for this task", name)
		}
		total += len(content)
		if total > requestBytesLimit {
			return fmt.Errorf("submitted files exceed %d bytes", requestBytesLimit)
		}
	}
	return nil
}

func validateResults(results contracts.TestResults) error {
	if results.Version <= 0 {
		return errors.New("results.json version must be positive")
	}
	if results.Status != "pass" && results.Status != "fail" && results.Status != "error" {
		return errors.New("results.json status is invalid")
	}
	if results.Status != "error" && len(results.Tests) == 0 {
		return errors.New("results.json must contain tests")
	}
	for _, test := range results.Tests {
		if strings.TrimSpace(test.Name) == "" {
			return errors.New("results.json contains a test without a name")
		}
		if test.Status != "pass" && test.Status != "fail" && test.Status != "error" {
			return errors.New("results.json contains an invalid test status")
		}
		if (test.Status == "fail" || test.Status == "error") && strings.TrimSpace(test.Message) == "" {
			return errors.New("failed tests must include a message")
		}
	}
	return nil
}

func copyDirectory(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, relErr := filepath.Rel(source, path)
		if relErr != nil {
			return relErr
		}
		if relative == "." {
			return os.MkdirAll(destination, 0o755)
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("task packs cannot contain symlinks")
		}
		input, openErr := os.Open(path)
		if openErr != nil {
			return openErr
		}
		defer input.Close()
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		output, createErr := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if createErr != nil {
			return createErr
		}
		_, copyErr := io.Copy(output, input)
		closeErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}

func readBounded(path string, limit int) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	body, err := io.ReadAll(io.LimitReader(file, int64(limit)+1))
	if err != nil {
		return nil, err
	}
	if len(body) > limit {
		return nil, errors.New("bounded output exceeded")
	}
	return body, nil
}

func safeRelativePath(value string) bool {
	if value == "" || filepath.IsAbs(value) || strings.Contains(value, "\\") {
		return false
	}
	clean := filepath.Clean(filepath.FromSlash(value))
	return clean == filepath.FromSlash(value) && clean != "." && clean != ".." && !strings.HasPrefix(clean, ".."+string(filepath.Separator))
}

var tokenPattern = regexp.MustCompile(`[^a-zA-Z0-9_.-]+`)

// Runtime diagnostics are copied from a container boundary and may contain
// absolute mount paths.  `/hidden-tests` identifies private harness files and
// `/solution`/`/output` reveal implementation details that do not belong in a
// learner-facing response.  Redact the complete path segment while keeping a
// useful line/column suffix (for example `<hidden-test>:1:7`).
var runtimeMountPathPattern = regexp.MustCompile(`(?i)/(?:hidden-tests|solution|output)(?:/[a-z0-9_.-]+)*`)

func safeToken(value string) string { return tokenPattern.ReplaceAllString(value, "-") }

func sanitizeError(value string) string {
	value = strings.ReplaceAll(value, "\x1b", "")
	value = strings.ReplaceAll(value, "\r", "")
	value = runtimeMountPathPattern.ReplaceAllStringFunc(value, func(path string) string {
		switch {
		case strings.HasPrefix(strings.ToLower(path), "/hidden-tests"):
			return "<private>"
		case strings.HasPrefix(strings.ToLower(path), "/solution"):
			return "<submission>"
		default:
			return "<runtime-output>"
		}
	})
	return strings.TrimSpace(value)
}

func sanitizeResults(results *contracts.TestResults) {
	if results == nil {
		return
	}
	results.Message = sanitizeError(results.Message)
	for index := range results.Tests {
		results.Tests[index].Message = sanitizeError(results.Tests[index].Message)
		results.Tests[index].Output = sanitizeError(results.Tests[index].Output)
		// `test_code` is an optional Exercism-compatible field.  It is private
		// harness source and must never become part of a learner response.
		results.Tests[index].TestCode = ""
	}
}

type limitedBuffer struct {
	bytes.Buffer
	limit     int
	truncated bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	remaining := b.limit - b.Len()
	if remaining <= 0 {
		b.truncated = true
		return len(p), nil
	}
	if len(p) > remaining {
		b.truncated = true
		p = p[:remaining]
	}
	return b.Buffer.Write(p)
}
