# fluent-task-runtime

Reusable, task-oriented execution boundary for Fluent products.

This repository is the independent execution boundary for Fluent products. It
does not accept arbitrary source code and it does not pretend that a task is
ready merely because a language profile exists. The engine executes an
immutable task revision through a pinned OCI profile, hidden-test harness and a
bounded result contract.

## Current release surface

The runtime is the standalone execution boundary used by Fluent Lab. The
current release surface is the Docker-backed R2/R3 slice plus the first
published Event Loop bridge:

- Go HTTP control plane boots locally;
- `/v1/health/live` and `/v1/health/ready` are explicit;
- `/v1/profiles` exposes the declared Node, .NET, PostgreSQL, Go and Java
  profiles;
- `/v1/tasks` exposes 18 pinned task-revision descriptors (12 Node.js,
  one .NET, three PostgreSQL, one Go and one Java); all 18 revisions are
  `released` after a real Docker-backed smoke for every descriptor. The
  profile catalogue remains explicit, so a future descriptor is `declared`
  until its own harness proof lands;
- every released task carries canonical `question.*` or `capability.*` bindings
  plus the exact Question Brain release ID. The runtime rejects legacy
  `Q123`/`C123`/`CAP-01` bindings at catalogue load, while Lab exposes a
  server-side relation audit at `/api/runtime/relations`;
- `/v1/runs` executes a released revision through Docker with no network,
  bounded CPU/memory/PIDs, read-only solution and hidden-test mounts, and a
  versioned result envelope;
- Jaeger is available in the local Compose profile on a unique port; readiness
  reports the Docker sandbox probe separately from the catalogue and the HTTP
  control plane emits OTLP spans for every request plus a child span for each
  task run.
- The image build is architecture-portable: Compose supplies BuildKit's
  `TARGETARCH` and the binary listens on `RUNTIME_PORT` (default `48227`).
- `node-event-loop-001@1` is the canonical Lab bridge: it accepts one
  editable `index.js`, runs the six published ordering challenges in a real
  Node.js 24 process, and emits a bounded `trace.json` artifact for the Lab
  evidence projection.

The next gate is a dedicated remote sandbox provider. The first project-book task,
`project-book-boundary-001@1`, is released and Docker-smoked; Lab still keeps
its project-book release and Tier 1 gates closed until the corresponding
evidence join is complete. Until a future revision is released, callers
receive a truthful `runtime_not_ready` response rather than a fake pass or a
browser-owned verdict.

The release smoke is recorded in
[`docs/verification/runtime-release-2026-08-22.json`](docs/verification/runtime-release-2026-08-22.json).

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
docker build -t fluent-runtime-task-node:1 task-images/node
docker build -t fluent-runtime-task-dotnet:1 task-images/dotnet
docker build -t fluent-runtime-task-postgres:1 task-images/postgres
docker build -t fluent-runtime-task-go:1 task-images/go
docker build -t fluent-runtime-task-java:1 task-images/java
docker compose -f deploy/compose/compose.yaml up -d
```

The `fluent-runtime-task-*` image namespace is dedicated to this runtime, so
rebuilding the Lab cannot silently replace a task sandbox image.

The Compose profile sends OTLP/gRPC to Jaeger's `4317` receiver. Verify that
the service is receiving traces after a smoke run:

```sh
curl http://127.0.0.1:56687/api/services
```

When running the binary directly, set `OTEL_EXPORTER_OTLP_ENDPOINT` to an
OTLP/gRPC endpoint (for example `127.0.0.1:14317`). Omitting it intentionally
uses OpenTelemetry's no-op provider, which keeps local unit tests isolated.

The local Compose execution profile mounts the Docker socket only to launch
disposable task containers; this is intentionally a development boundary. A
production deployment must put the sandbox behind a dedicated worker service
instead of granting the control plane a host socket.

For local Compose, the default `RUNTIME_HOST_WORK_ROOT` is the absolute
`.runtime-work` directory in this checkout, so `docker compose -f
deploy/compose/compose.yaml up -d` works without a hand-written `.env`. Set
`RUNTIME_HOST_WORK_ROOT` only when running a second stack or deliberately
moving the disposable work root; the runtime and Docker daemon must see the
same path or the hidden-test harness fails closed.
