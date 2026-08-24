# G3 evidence — TaskFamily and language revisions

Date: 2026-08-25
Status: **complete**
Gate: G3 — TaskFamily, immutable TaskRevision metadata, and runtime projections

## Decision

Task Runtime now has a separate language-neutral family manifest. It contains
15 families and all 18 current revision directories. The four rate-limiter
implementations are four revisions of one `task-family.rate-limiter`; they
share one capability and preparation context while keeping independent
language/profile execution. `project-book-boundary-001@1` is retained in the
manifest as `unreleased`, so it cannot be mistaken for an executable learner
task until its brief/workspace gate is complete.

Executable source, starter files, tests, harnesses, images, limits, and sandbox
policy remain in the existing `tasks/<taskId>/` revision directories. The
family manifest contains only localized metadata, canonical capability keys,
rubric references, availability, and immutable directory hashes.

## Release and API

New immutable release:
`releases/task-release-2026-08-25-qb-d550846f-g3.json`
Family release:
`task-families/manifest.json` (`task-family-release-2026-08-25`)
Question Brain release:
`question-release-d550846f4743c4d3`

The previous `task-release-2026-08-24*.json` files remain historical and are
not edited. The v2 runtime release pins `taskFamilyKey` for all 18 revisions
and canonical capability keys for the 11 renamed stations.

New safe projections:

- `GET /v1/task-families` returns all 15 families and revision availability.
- `GET /v1/task-families/{familyKey}` returns one family.
- Neither endpoint exposes source, starter files, solutions, hidden tests,
  image names, commands, or sandbox policy.
- A family is `runnable=true` only when at least one revision is actually
  runnable. The project-book family is `runnable=false`/`unreleased`.

## Fail-closed checks

- A released task revision missing from the family manifest fails catalogue
  startup.
- A family revision with a language/profile or immutable directory hash that
  differs from `task.json` fails catalogue startup.
- A v2 runtime release without `taskFamilyReleaseId` or without a matching
  family manifest fails startup.
- A task workspace/run for an `unreleased` revision returns `runtime_not_ready`;
  no fallback workspace or fake pass is synthesized.

## Docker verification

Runtime image digest:
`sha256:9048934b86f72ad78b861cb8d287d249cff4795f9ef00d4a5bc7503285cff6a2`

Compose readiness was `200/ready`; Jaeger exposed both
`fluent-task-runtime` and `jaeger-all-in-one`. For every affected profile a
real Docker-backed pass and fail run was executed:

| Profile | Task | Pass | Fail |
|---|---|---|---|
| Node.js | `node-event-loop-001` | `g3-node-pass` | `g3-node-fail` |
| Go | `go-rate-limiter-001` | `g3-go-pass` | `g3-go-fail` |
| Java | `java-rate-limiter-001` | `g3-java-pass` | `g3-java-fail` |
| .NET | `dotnet-cancellation-001` | `g3-dotnet-pass` | `g3-dotnet-fail` |
| PostgreSQL | `pg-rate-limiter-001` | `g3-sql-pass` | `g3-sql-fail` |

All pass runs returned `results.status=pass`; all intentionally incomplete
submissions returned `results.status=fail`.

Machine-readable evidence is in
`docs/verification/runtime-task-families-2026-08-25.json`.
