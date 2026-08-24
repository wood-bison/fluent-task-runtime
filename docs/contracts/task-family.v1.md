# TaskFamily v1 — one learning objective, many executable revisions

`TaskFamily` is the language-neutral runtime identity. It owns the localized
title/brief, reviewed capability keys, rubric reference, and the list of
available `TaskRevision` identities. It never contains source code, solutions,
hidden tests, harness commands, image names, or sandbox policy.

`TaskRevision` is the immutable executable directory under `tasks/<taskId>/`.
It has exactly one language and one runtime profile. The family manifest pins
its `immutableHash`; changing starter code, tests, limits, or the descriptor
requires a new revision or a new release, never an in-place learner mutation.

The current manifest is `task-families/manifest.json` and carries:

- 15 families;
- 18 revisions;
- four language/profile revisions of one rate-limiter family (Go, Java,
  Node.js, and PostgreSQL);
- explicit `runnable`, `brief_only`, `profile_unavailable`, `superseded`, and
  `unreleased` availability states.

The v2 runtime release joins each `(taskId, revision)` to exactly one family
and to the Question Brain release. Historical v1 release files remain
readable but are not selected as a fallback when the v2 manifest is requested.

Learner endpoints are deliberately split:

- `/v1/task-families` and `/v1/task-families/{key}` expose safe family and
  revision metadata;
- `/v1/tasks/{taskId}/workspace` exposes only the brief and starter files for
  an actually runnable revision;
- `/v1/runs` executes only an actually runnable revision and returns bounded
  evidence. No endpoint returns solutions or hidden tests.
