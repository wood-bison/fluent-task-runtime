# Task ↔ Question Brain release binding

The task runtime owns executable task revisions. Question Brain owns question
content and content releases. This contract joins them without sharing a
database or copying question prose into the runtime repository.

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
  "capabilityKeys": [],
  "questionKeys": ["question.q315"]
}
```

`questionBindings` is the authoritative join. `questionKeys` is a deprecated
compatibility projection and may be removed only after all Lab clients consume
`questionBindings`.

## Manifest and immutability

The release manifest is [`../../releases/task-release-2026-08-24.json`](../../releases/task-release-2026-08-24.json).
It has contract `fluent-task-runtime.task-release.v1` and contains one entry
per released `(taskId, revision)` pair. The manifest pins one Question Brain
release and the exact identity of each referenced revision:

- `stableKey` is the durable Question Brain address;
- `revisionId` is the immutable Question Brain revision UUID;
- `contentHash` is the SHA-256 payload identity;
- `capabilityKeys` are explicit cross-system capability references and are
  never inferred from `breadcrumb`, `concepts`, or a task title.

The loader applies this manifest as an overlay. Existing released `task.json`
files remain unchanged, so a historical execution context cannot be silently
rewritten. If a descriptor and manifest both declare a value, they must agree;
conflicts fail catalogue startup. A released task missing from an active
manifest also fails startup. This makes a release incomplete rather than
allowing a partially bound catalogue into the API.

For a future Question Brain release:

1. fetch the published `revision_id` and `content_hash` for every referenced
   stable key;
2. create a new runtime manifest with a new `releaseId` and the new
   `questionReleaseId`;
3. run the catalogue and relation checks before changing the workspace pin;
4. publish the manifest with the runtime release; never edit the old manifest.

The runtime does not fetch Question Brain during a task run. Lab may fetch the
referenced card for display, but it must verify the returned revision and hash
against the binding before presenting the question as the task's context.

## Compatibility policy

Old descriptors may still contain `questionKeys` and `questionReleaseId` while
the migration is in progress. They are accepted only as a read-only
compatibility input and are projected into the new response. New release
manifests require full `questionBindings` for question-backed tasks and valid
`capabilityKeys` for capability-only tasks. Legacy identifiers such as `Q123`,
`C123`, or `CAP-01` are rejected.
