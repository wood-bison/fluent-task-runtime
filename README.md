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
- every released task carries the exact Question Brain release ID. The
  active immutable revision identity is supplied by
  [`releases/task-release-2026-08-24-qb-d550846f-i2.json`](releases/task-release-2026-08-24-qb-d550846f-i2.json):
  each entry pins `stableKey`, Question Brain `revisionId`, and
  `contentHash`, and exposes explicit `capabilityKeys` for the executable
  station crosswalk. The older `questionKeys` property remains a read-only
  projection for Lab clients that have not yet migrated; it is never used to
  replace a full binding. The runtime rejects malformed or conflicting
  bindings at catalogue load, while Lab exposes a server-side relation audit
  at `/api/runtime/relations`;
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

## Question Brain release binding

Task code and hidden tests are still owned by the task revision directory. The
question identity is deliberately kept in a separate release manifest so a
previously released `task.json` is not rewritten when Question Brain publishes
new content. A manifest entry is keyed by `(taskId, revision)` and is applied
only when the runtime loads the immutable catalogue.

```json
{
  "taskId": "node-rate-limiter-001",
  "revision": 1,
  "questionBindings": [
    {
      "stableKey": "question.q315",
      "revisionId": "7df22e91-c351-4cd3-9bee-7ab321c72efd",
      "contentHash": "4d3598baa00926e1a62e48ecc8544d5597f289be04331968b891b020eebf496d"
    }
  ],
  "capabilityKeys": ["capability.distributed-systems.rate-limiter"]
}
```

The release ID is a content-release pin, not a live lookup. Updating a
question creates a new Question Brain release and a new runtime release
manifest; it must not silently change the evidence context of an old run. Set
`RUNTIME_RELEASE_MANIFEST` to test a candidate manifest before publishing it.
The local Compose stack selects
`/opt/releases/task-release-2026-08-24-qb-d550846f-i2.json`, whose
`questionReleaseId` matches the current Question Brain deployment. There is no
implicit legacy fallback: without an explicit manifest the runtime reports
`manifest-not-configured`, readiness is degraded, and workspace/run requests
return `runtime_not_ready`.

`GET /v1/tasks/summary` makes this state machine explicit. It returns
`manifest-loaded` with the runtime and Question Brain release IDs when an
overlay is selected, or `manifest-not-configured` when the runtime is serving
compatibility descriptors. The response and each task include `runnable` so a
client cannot mistake historical descriptors for an executable release. The
capability key is the only accepted join into Lab's station taxonomy; it is
never inferred from a task title or breadcrumb.
`GET /v1/tasks/{taskId}/workspace` returns the learner brief and starter files
only once a release is selected; hidden tests and harness files never cross
this API boundary. A released task without an authored `brief.md` is reported
as `workspace_unavailable` rather than receiving a synthesized fallback; this
currently keeps the project-book task closed until its learner brief is
authored.

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
