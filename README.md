# fluent-task-runtime

Reusable, task-oriented execution boundary for Fluent products.

This repository is intentionally starting as a contract-first vertical slice.
It does not accept arbitrary source code and it does not pretend that a task is
ready merely because a language profile exists. The engine will execute an
immutable task revision through a pinned OCI profile, hidden-test harness and a
bounded result contract.

## Current gate

R2 is the active gate from the Lab migration plan:

- Go HTTP control plane boots locally;
- `/v1/health/live` and `/v1/health/ready` are explicit;
- `/v1/profiles` exposes the declared Node, .NET, PostgreSQL, Go and Java
  profiles;
- `/v1/runs` is deliberately not advertised as executable until the sandbox
  adapter and task-pack revisions land;
- Jaeger is available in the local Compose profile on a unique port.

The next gate adds one real task revision and a Docker-backed harness. Until
then, a caller receives a truthful `runtime_not_ready` response rather than a
fake pass or a browser-owned verdict.

## Boundaries

- Question Brain owns questions, graph placement and releases.
- Lab owns curriculum, learner UX, attempts and evidence projections.
- This repository owns task revisions, profile images, sandbox policy, hidden
  tests, run results and runtime traces.

## Local development

```sh
go test ./...
go run ./cmd/runtime
curl http://127.0.0.1:48227/v1/health/ready
```

Compose adds Jaeger at <http://127.0.0.1:56687>:

```sh
docker compose -f deploy/compose/compose.yaml up -d
```

No learner source, credentials, Docker socket or host filesystem is mounted by
this contract-first slice.
