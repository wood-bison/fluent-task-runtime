# Task ↔ Question Brain release binding

The task runtime owns executable task revisions. Question Brain owns question
content and content releases. This contract joins them without sharing a
database or copying question prose into the runtime repository.

> **Current production contract (2026-08-25):** the active join is
> [`task-release-2026-08-25-qb-d00a1493-g8.json`](../../releases/task-release-2026-08-25-qb-d00a1493-g8.json),
> contract `fluent-task-runtime.task-release.v3`. It pins the Question Brain
> source snapshot, capability binding/registry releases, and TaskFamily
> release in addition to the exact question revision/hash bindings. The
> v1/v2 examples below are immutable migration history only; they are not a
> production fallback. See [`task-release.v3.md`](task-release.v3.md) for the
> normative current contract.

## Runtime response

`GET /v1/tasks` returns the following metadata for a released revision:

```json
{
  "taskId": "node-rate-limiter-001",
  "revision": 1,
  "status": "released",
  "questionReleaseId": "question-release-15e032d7b732f8c1",
  "questionBindings": [
    {
      "stableKey": "question.q315",
      "revisionId": "7df22e91-c351-4cd3-9bee-7ab321c72efd",
      "contentHash": "4d3598baa00926e1a62e48ecc8544d5597f289be04331968b891b020eebf496d"
    }
  ],
  "capabilityKeys": ["capability.distributed-systems.rate-limiter"],
  "questionKeys": [
    "question.q315",
    "capability.distributed-systems.rate-limiter"
  ]
}
```

`questionBindings` is the authoritative join. `questionKeys` is a deprecated
compatibility projection: it contains the `stableKey` of every
`questionBindings` entry followed by every `capabilityKeys` entry, including
hierarchical keys such as `capability.distributed-systems.rate-limiter`. It
may be removed only after all Lab clients consume `questionBindings` and
`capabilityKeys`.

## Manifest and immutability

The historical candidate release manifest is [`../../releases/task-release-2026-08-24.json`](../../releases/task-release-2026-08-24.json).
It has contract `fluent-task-runtime.task-release.v1` and contains one entry
per released `(taskId, revision)` pair. The manifest pins one Question Brain
release and the exact identity of each referenced revision:

- `stableKey` is the durable Question Brain address;
- `revisionId` is the immutable Question Brain revision UUID;
- `contentHash` is the SHA-256 payload identity;
- `capabilityKeys` are explicit cross-system capability references and are
  never inferred from `breadcrumb`, `concepts`, or a task title.

The loader applies this manifest as an overlay only when
`RUNTIME_RELEASE_MANIFEST` is explicitly set. Existing released `task.json`
files remain unchanged, so a historical execution context cannot be silently
rewritten. The active manifest is authoritative for the legacy
`questionReleaseId`/`questionKeys` projection; those old descriptor values do
not block generating a new overlay. If a descriptor already contains full
immutable `questionBindings`, a disagreement is an integrity error and fails
catalogue startup. A released task missing from an active manifest also fails
startup. This makes a release incomplete rather than allowing a partially
bound catalogue into the API.

The historical candidate points at `question-release-15e032d7b732f8c1` and is
kept immutable for audit purposes. The reconciled candidate
[`task-release-2026-08-24-qb-d550846f.json`](../../releases/task-release-2026-08-24-qb-d550846f.json)
pins `question-release-d550846f4743c4d3`; the active I2 manifest
[`task-release-2026-08-24-qb-d550846f-i2.json`](../../releases/task-release-2026-08-24-qb-d550846f-i2.json)
adds the reviewed executable capability keys. It is selected only through
`RUNTIME_RELEASE_MANIFEST`; no file is an implicit fallback. If Question Brain
publishes a new release, generate another manifest, run the catalogue and
relation checks, and select it through the deployment environment.

For a future Question Brain release:

1. fetch the published `revision_id` and `content_hash` for every referenced
   stable key;
2. create a new runtime manifest with a new `releaseId` and the new
   `questionReleaseId`;
3. run the catalogue and relation checks before changing the workspace pin;
4. publish the manifest with the runtime release; never edit the old manifest.

The reproducible helper is
[`scripts/release/generate-question-release.py`](../../scripts/release/generate-question-release.py):

```bash
python3 scripts/release/generate-question-release.py \\
  --source releases/task-release-2026-08-24.json \\
  --question-brain http://127.0.0.1:48127 \\
  --workspace fluent-interview \\
  --tasks-root tasks \\
  --release-id runtime-task-release-YYYY-MM-DD-qb-<release-suffix> \\
  --output releases/task-release-YYYY-MM-DD-qb-<release-suffix>.json
```

The current v3 generator reads the released Question Brain API and the
released TaskFamily API first, then uses authored descriptors only for the
task source identity. A family-level capability snapshot is preferred over a
descriptor breadcrumb. It copies no question prose and requires every
question binding to resolve to a published production card. A capability can
be shared by multiple language revisions (for example, the Node.js, Go, Java
and PostgreSQL rate-limiter tasks) without pretending that their source code
is interchangeable.

The runtime does not fetch Question Brain during a task run. Lab may fetch the
referenced card for display, but it must verify the returned revision and hash
against the binding before presenting the question as the task's context.

`GET /v1/tasks/summary` reports whether an overlay is loaded and includes the
runtime release ID, Question Brain source snapshot, capability binding and
registry release IDs, TaskFamily release ID, binding state, and safe task
metadata. With no explicit manifest the summary is `runnable: false`, the
readiness probe is degraded, and both workspace and run requests return
`runtime_not_ready`; descriptor compatibility is never a runnable fallback.
`GET /v1/tasks/{taskId}/workspace` is the learner-material boundary: it returns
`brief.md` and files under `starter/`, never `tests/`, hidden tests, or harness
commands. If a released task has no authored `brief.md`, the endpoint returns
`workspace_unavailable`; it does not invent a fallback brief.

## Compatibility policy

Old descriptors may still contain `questionKeys` and `questionReleaseId` while
the migration is in progress. They are accepted only as a read-only
compatibility input and are projected into the new response. Without an
explicit manifest they cannot make a task runnable. New release
manifests require full `questionBindings` for question-backed tasks and valid
`capabilityKeys` for capability-only tasks. Legacy identifiers such as `Q123`,
`C123`, or `CAP-01` are rejected.
