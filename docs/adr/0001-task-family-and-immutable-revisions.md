# ADR-0001 — Task Runtime owns TaskFamily and immutable TaskRevision

**Status:** accepted  
**Date:** 2026-08-24  
**Builds on:** `docs/contracts/question-brain-release-binding.md`

## Context

The same interview capability may be assessed in Go, Java, C#, TypeScript, or
SQL. A task directory is an executable revision, not the learner concept. If
the directory or language is used as the capability identity, Question Brain,
Lab, and Runtime cannot evolve independently.

## Decision

Task Runtime owns:

- language-neutral `TaskFamily` identity, localized brief metadata, explicit
  capability keys, and optional revision-pinned `QuestionBinding` context;
- immutable `TaskRevision` identity (`taskId`, positive revision, exactly one
  language and runtime profile, immutable content hash);
- starter workspace, solution, hidden tests, harness, OCI image, sandbox
  policy, limits, Run results, and bounded traces.

`TaskFamily` never contains a language/profile field. Every executable revision
belongs to exactly one family and declares its own language/profile. A new
language is additive: it adds a revision and does not clone or mutate the
QuestionCard or Capability. The runtime never fetches question prose during a
Run and never writes Question Brain or Lab Evidence.

`question-capability-task.v1` is the cross-repository identity contract. The
`questionBindings` array is authoritative for question-backed revisions and
`capabilityKeys` is authoritative for capability-only revisions. The removed
overloaded `questionKeys` projection is rejected by current readers; manifests
without immutable revision/hash pins are rejected.

## Consequences

- One token-bucket family can have Go/Java/C#/TypeScript revisions with the same
  capability and preparation card.
- Runtime release IDs and Question Brain release IDs remain independently
  immutable and are joined by a manifest.
- A task can be runnable only after its own image/harness smoke and release
  manifest validation; a declared profile is not a runnable task.

## Rejected alternatives

- Copying QuestionCard answers or hidden tests into Lab: violates ownership and
  makes release rollback impossible.
- Making each language a different capability: confuses implementation choice
  with the observable skill unless technology is intrinsically part of that
  skill (for example, Node event-loop ordering).
