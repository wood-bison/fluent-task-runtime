# Bounded concurrency without losing order

`Promise.all(jobs.map(run))` starts everything at once. With ten thousand jobs
that floods the dependency and exhausts sockets or file descriptors long before
it finishes. The fix is a pool: a fixed number of workers that each pull the
next job until the list is drained.

Two properties are easy to lose while doing that:

- **Order.** Completion order is random; output order must match input order.
  Writing each result to its own index is what preserves it.
- **The ceiling.** A pool that queues every promise upfront and awaits later
  has a limit on paper only. The test watches actual in-flight count.

One failing job must not take the batch down — settle, report, continue.
