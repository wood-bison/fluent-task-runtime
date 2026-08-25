# G8 release join verification — 2026-08-25

Status: **passed** for the local production Compose stack.

## Active release evidence

`GET http://127.0.0.1:48127/v1/release?workspace=fluent-interview` and
`GET http://127.0.0.1:48227/v1/tasks/summary` agree on the following pins:

| Projection | Value |
| --- | --- |
| Question Brain contract | `question-brain.release.v1` |
| workspace | `fluent-interview` |
| Question release / source snapshot | `question-release-d00a14931e607336` |
| Capability registry release | `capability-registry-2026-08-25-v3` |
| Capability binding release | `question-capability-release-3c38b4c8c0fa7f47` |
| TaskFamily release | `task-family-release-2026-08-25` |
| Runtime release | `runtime-task-release-2026-08-25-qb-d00a1493-g8` |
| Question Brain production cards | 1,591 |
| canonical capability keys | 16 |
| Task revisions in summary | 18 |
| released revisions | 18 |
| runnable revisions | 17 |
| explicit capability-only revisions | 1 (`project-book-boundary-001@1`) |
| revisions without a TaskFamily | 0 |

The capability-only project-book revision has an empty `questionBindings` array
and the canonical `capability.delivery-observability.execution-boundary` key;
it does not abuse the deprecated `questionKeys` field.

## Automated checks

```text
$ docker run --rm -v fluent-task-runtime:/src -w /src golang:1.24-bookworm \
    sh -c 'PATH=/usr/local/go/bin:$PATH gofmt -w contracts/runtime.go internal/engine/catalogue.go internal/engine/catalogue_test.go internal/httpapi/server_test.go && go test ./...'
ok  contracts
ok  internal/engine
ok  internal/httpapi

$ ./scripts/release/g8-release-join-smoke.sh
G8 release join smoke passed: 18 tasks, runtime=runtime-task-release-smoke-g8,
question=question-release-d00a14931e607336,
task-families=task-family-release-2026-08-25

$ git diff --check
passed
```

The smoke generates a new v3 manifest from both live APIs in a temporary file,
asserts all 18 revisions have TaskFamily identity and explicit bindings or
capabilities, and asserts that no `questionKeys` compatibility field is
generated. Go unit coverage includes a capability-outside-registry rejection,
G8 summary pin projection, and explicit capability-only handling.

## Lab adapter checks

The Lab adapter runs with `TASK_RUNTIME_RELEASE_JOIN=v3`. It fetches the
health, profiles, task catalogue, and `/v1/tasks/summary` together. The
relation audit refuses a missing/degraded summary, a non-runnable manifest, or
a Question Brain release mismatch; the response carries all release pins for
operator retrospectives. There is no descriptor, local snapshot, or legacy
fallback path.

```text
$ pnpm nx test lab-contracts --skip-nx-cache
Test Suites: 241 passed, Tests: 1218 passed

$ pnpm nx test learning-api --skip-nx-cache --runInBand \
    --runTestsByPath src/app/task-runtime/task-runtime.client.spec.ts \
    src/app/task-runtime/task-runtime-relation.service.spec.ts
Test Suites: 2 passed, Tests: 11 passed
```

The remaining full Lab check is run by the repository release gate after the
adapter build is restarted against the active runtime container.
