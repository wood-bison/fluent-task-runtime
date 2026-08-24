# Question → Capability → Task contract v1

The workspace machine-readable schema and fixture are maintained at
`fluent-interview/docs/contracts/question-capability-task.v1.*`. Runtime owns
the `TaskFamily`, `TaskRevision`, `QuestionBinding`, and Run subset. Before a
release, tooling must verify the schema/fixture hash and reject:

- a capability key in `questionKeys`;
- a TaskFamily with language/profile fields;
- a TaskRevision without exactly one language/profile and immutable hash;
- a released binding without stable key, revision ID, and SHA-256 content hash.

Executable source, hidden tests, harnesses, images, and sandbox limits never
cross this boundary.

