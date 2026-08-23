# Node.js Event Loop: prediction to evidence

The Lab sends one published event-loop challenge as `index.js`. The source is
wrapped by the hidden runner with two server-owned helpers:

- `schedule(id, label, detail, queue)` records that work entered a queue;
- `emit(value, queue, id, detail, label)` prints the learner-visible value and
  records the observed execution step.

Run the submitted program in a real Node.js 24 process. The hidden checks keep
the output private, compare it with the six published challenge orderings, and
write the bounded measured trace to `trace.json`. The trace is evidence for the
Lab UI; it is never authored by the browser or accepted from learner input.

The editable contract is deliberately one file: `index.js`. Do not add a
second executor, network access, or an alternate task revision.
