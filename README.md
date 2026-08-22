# fluent-task-runtime

Reusable, task-oriented execution boundary for Fluent products.

This repository is the independent execution boundary for Fluent products. It
does not accept arbitrary source code and it does not pretend that a task is
ready merely because a language profile exists. The engine executes an
immutable task revision through a pinned OCI profile, hidden-test harness and a
bounded result contract.

## Current gate

The current gate from the Lab migration plan is the deliberately small R2
runtime slice:

- Go HTTP control plane boots locally;
- `/v1/health/live` and `/v1/health/ready` are explicit;
- `/v1/profiles` exposes the declared Node, .NET, PostgreSQL, Go and Java
  profiles;
- `/v1/tasks` exposes the 14 pinned task-revision descriptors (nine Node.js,
  one .NET, two PostgreSQL, one Go and one Java); all 14 revisions are now
  `released` after a real Docker-backed smoke for every descriptor. The
  profile catalogue remains explicit, so a future descriptor is `declared`
  until its own harness proof lands;
- `/v1/runs` executes a released revision through Docker with no network,
  bounded CPU/memory/PIDs, read-only solution and hidden-test mounts, and a
  versioned result envelope;
- Jaeger is available in the local Compose profile on a unique port; readiness
  reports the Docker sandbox probe separately from the catalogue and the HTTP
  control plane emits OTLP spans for every request plus a child span for each
  task run.
- The image build is architecture-portable: Compose supplies BuildKit's
  `TARGETARCH` and the binary listens on `RUNTIME_PORT` (default `48227`).

The next gate adds dual-run evidence for every profile and a dedicated remote
sandbox provider. Until a future revision is released, callers receive a
truthful `runtime_not_ready` response rather than a fake pass or a
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
docker compose -f deploy/compose/compose.yaml up -d
```

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

For a local nested-container smoke, set `RUNTIME_HOST_WORK_ROOT` in `.env` to
an absolute disposable directory in this checkout. The runtime and Docker
daemon must see the same path; otherwise the daemon would mount an empty path
and the hidden-test harness would fail closed.
