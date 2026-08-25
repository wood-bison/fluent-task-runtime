# Task Runtime release join v3

`fluent-task-runtime.task-release.v3` is the production boundary between the
independent Task Runtime and Question Brain repositories. It is generated from
the two released APIs and the TaskFamily release; it is never assembled from
titles, filenames, breadcrumbs, or copied question prose.

## What is pinned

The manifest pins one immutable release identity for every side of the join:

| Field | Owner | Meaning |
| --- | --- | --- |
| `releaseId` | Task Runtime | The exact executable task release |
| `workspaceKey` | shared contract | Prevents cross-workspace joins |
| `questionReleaseId` | Question Brain | Published QuestionCard release |
| `questionSourceSnapshotId` | Question Brain | Source snapshot used to build it |
| `capabilityBindingReleaseId` | Question Brain | Reviewed Question ↔ Capability bindings |
| `capabilityRegistryReleaseId` | Question Brain | Canonical capability registry |
| `taskFamilyReleaseId` | Task Runtime | Grouping of language revisions |
| `capabilityKeys` | Question Brain | Complete canonical capability snapshot |

Each question-backed revision has only the following authoritative join:

```json
{
  "taskId": "node-rate-limiter-001",
  "revision": 1,
  "taskFamilyKey": "task-family.rate-limiter",
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

`questionBindings` are immutable revision-plus-hash identities. A capability is
not a question and is never encoded into `questionBindings`. A capability-only
capstone has an empty binding array and explicit `capabilityKeys`, for example
`project-book-boundary-001@1`. The deprecated `questionKeys` projection may be
returned by the legacy `/v1/tasks` endpoint for old consumers, but it is not
read by the v3 generator or by the Lab relation join.

## Fail-closed rules

The Go catalogue refuses to become ready when a v3 manifest has a wrong
workspace, missing Question Brain contract/source snapshot, missing capability
release pin, missing TaskFamily release, malformed binding, duplicate task
revision, or a capability outside the pinned registry snapshot. Every task
entry must name a TaskFamily and contain at least one question binding or
capability key. The runtime does not call Question Brain during execution, so a
run cannot silently drift to newer content.

The Lab adapter sets `TASK_RUNTIME_RELEASE_JOIN=v3` (the default) and reads
`GET /v1/tasks/summary` before exposing the relation audit. If the summary is
missing, degraded, not `manifest-loaded`, not runnable, or pins a different
Question Brain release than the live catalog, Lab returns a typed degraded
state. There is no v1/v2 descriptor fallback and no fabricated runnable state.

## Generation and verification

Generate a candidate from a released source manifest:

```bash
python3 scripts/release/generate-question-release.py \
  --source releases/task-release-2026-08-25-qb-d550846f-g3.json \
  --question-brain http://127.0.0.1:48127 \
  --runtime-api http://127.0.0.1:48227 \
  --workspace fluent-interview \
  --tasks-root tasks \
  --release-id runtime-task-release-YYYY-MM-DD-qb-<brain-suffix> \
  --output /tmp/task-release-v3.json
```

The generator fails if any referenced card is absent, unpublished, non-
production, or has no current revision/hash; it also fails if a TaskFamily or
capability reference cannot be resolved. The committed production manifest is
`releases/task-release-2026-08-25-qb-d00a1493-g8.json`.

The runtime verification surface is:

```bash
go test ./...
curl -fsS http://127.0.0.1:48227/v1/tasks/summary
```

The Lab verification surface is:

```bash
pnpm nx test lab-contracts --skip-nx-cache
pnpm nx test learning-api --skip-nx-cache --runInBand
```

The `/api/runtime/relations` projection includes the runtime, Question Brain,
capability binding/registry, and TaskFamily release IDs so a retrospective can
prove exactly which question graph and executable revisions were joined.
