# Task Portfolio Contract v1

`task-portfolio/manifest.json` is the fail-closed production-closure plan for
Task Runtime. It allocates target `TaskFamily` slots and their required
language/profile compatibility without making an unfinished task runnable.

The portfolio is **not** an execution catalogue. The immutable
`task-families/manifest.json`, pinned by the runtime release, remains the only
source of executable family and revision metadata. The portfolio deliberately
contains no source path, image, test command, availability, or immutable hash.

## Arithmetic invariant

The v1 production target is exact:

| Kind | Families | Revisions per family | Revisions |
| --- | ---: | ---: | ---: |
| Algorithms | 60 | 5 | 300 |
| Shared backend | 12 | 5 | 60 |
| Native runtime/framework | 80 (16 per track) | 1 | 80 |
| PostgreSQL | 16 | 1 | 16 |
| **Total** | **168** |  | **456** |

The five cross-language revisions are JavaScript/Node, TypeScript/Node,
Java/JVM, Go/Go, and C#/.NET. Native tracks are Node, Java, Go, .NET, and Vue;
each owns exactly 16 families. PostgreSQL families use SQL/PostgreSQL.

## Dispositions

- `existing` consumes a target slot with a family already present and released
  in the pinned TaskFamily release. Existing revisions may be a subset of the
  target compatibility set; missing revisions are closure work.
- `new` reserves deterministic family keys through a compact `plannedSeries`.
  A reserved key is neither published nor runnable.
- `superseded` accounts for a historical or misplaced family outside the
  production target and names its replacement owner. Its immutable release is
  not rewritten.

Every family in the pinned TaskFamily release must be classified exactly once.
Released families must be `existing`; non-released families must be
`superseded`. Planned keys must not collide with any current family.

## Promotion rule

A `new` slot becomes real only through the normal Task Runtime release process:
author a task descriptor and hidden tests, add an immutable `TaskRevision`, bind
it to a `TaskFamily`, build and pin its profile image, and publish a new release.
Promotion updates the portfolio in a later commit from `new` to `existing` and
advances `sourceTaskFamilyReleaseId`; it never mutates a published release.

## Validator guarantees

The static validator rejects unknown JSON fields and fails closed on:

- target or per-kind arithmetic drift from 168 families / 456 revisions;
- duplicate group, compatibility-set, family, or language/profile identities;
- foreign language/profile combinations;
- invalid disposition or deterministic series ranges;
- missing, stale, or unclassified families in the pinned TaskFamily release;
- planned keys that already exist; and
- native-track totals other than 16 families each.

The validator is intentionally independent of runtime startup: an unfinished
portfolio cannot change the current executable release.
