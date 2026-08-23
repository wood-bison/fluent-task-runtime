# AGENTS.md — Fluent Task Runtime

Task Runtime is the independent execution boundary for Fluent products. It
accepts a declared immutable task revision, runs it through a pinned language
profile and hidden-test harness in an isolated container, and returns a typed
result envelope with evidence and traces. It is not a question service and it
is not a browser-side code runner.

## Goals

- Run the same task contract from Fluent Lab and future clients across Node.js,
  .NET/C#, PostgreSQL, Go, Java and additional profiles when each has a real
  image and harness proof.
- Keep execution deterministic, bounded, reproducible and auditable.
- Keep task content, profile images, hidden tests and result schemas versioned so
  a task can be released, rolled back or re-smoked independently.
- Expose a small HTTP control plane with health, profile, task and run APIs plus
  OpenTelemetry traces and metrics.
- Make project-book tasks possible without coupling the runtime to Lab's
  curriculum or UI.

## Ownership and boundaries

| Component | Owns | Never owns |
| --- | --- | --- |
| Task Runtime | task revisions, profiles, sandbox policy, hidden tests, run results and traces | questions, graph placement, learner progress, UI verdicts |
| Question Brain | question text, locales, graph and releases | source files submitted to a task runner |
| Fluent Lab | curriculum, learner UX, attempts and evidence projection | hidden tests or a local execution engine |

Lab calls the versioned runtime API. A task/profile that is not explicitly
released returns `runtime_not_ready`; there is no fallback catalogue, default
profile, browser execution, or fake pass.

## Structure

```text
cmd/runtime/                HTTP control-plane entrypoint
internal/engine/             catalogue validation and Docker executor
internal/httpapi/             health, profile, task and run handlers
internal/telemetry/           OpenTelemetry setup and bounded attributes
contracts/                    versioned public runtime/result contracts
tasks/<id>/                   task descriptors and learner-facing briefs
task-images/<profile>/        pinned OCI sandbox image definitions
profiles/                     profile metadata and compatibility notes
deploy/compose/               local runtime + Jaeger stack
docs/verification/            release-smoke and hardening evidence
```

## Execution invariants

1. Only a declared, released task revision may run.
2. The learner solution and hidden tests are mounted separately; hidden tests
   never enter the browser or result payload.
3. Containers run with no network, read-only solution input, bounded CPU,
   memory, PIDs and wall-clock time. The control plane never executes learner
   code in-process.
4. Results are versioned and typed (`pass`, `fail`, `timeout`, `runtime_error`,
   `runtime_not_ready`); transport success is not a task pass.
5. Profile and task changes require a real Docker smoke and updated evidence.
6. Correlation/task/revision IDs belong in trace attributes and structured logs,
   not unbounded metric labels. Do not log source or hidden-test bodies.
7. The Docker socket is a local development boundary only. Production should
   place sandbox execution behind a dedicated worker service.

## Operational contract

```bash
go test ./...
go run ./cmd/runtime
curl http://127.0.0.1:48227/v1/health/ready
docker compose -f deploy/compose/compose.yaml up -d
curl http://127.0.0.1:56687/api/services
```

Compose reserves the runtime API and Jaeger ports documented in `README.md`.
Use one Compose project name and the same absolute work-root path for nested
Docker mounts. Keep resource limits and health checks enabled in every profile.

## Working agreement

- Keep the public contracts and Lab adapter synchronized when an API changes.
- Add a task directory, descriptor, brief, profile image and hidden-test smoke
  together; do not expose a profile merely because its image builds.
- Delete unused catalogue entries and compatibility aliases rather than keeping
  fallback behavior.
- Run Go tests, Docker-backed smoke for affected profiles, `git diff --check`,
  and the relevant Compose readiness checks before committing to `main`.
- Treat release evidence as reproducible verification, not as a substitute for
  the runtime contract.
