# At-least-once delivery without doing the work twice

Every broker worth using promises at-least-once, not exactly-once. The same
message arrives again after a consumer crash, a visibility timeout, or a
partition rebalance. Exactly-once is something the *consumer* provides, by
being idempotent — not something the broker hands you.

Three things have to hold at once:

- **Dedupe by id.** A second delivery of a completed id returns the first
  result without re-running the work. For a payment, "re-running" means
  charging twice.
- **A retry bound.** A transient failure deserves another attempt; an
  unparseable message deserves a dead-letter queue. Without the bound a poison
  message is retried forever.
- **Isolation.** One bad message must not stop the ones behind it.
