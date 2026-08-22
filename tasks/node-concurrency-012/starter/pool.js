/**
 * Run `jobs` with at most `limit` running at the same time.
 *
 * Each job is a function returning a promise. Resolve with an array of
 * settled results in the SAME order as `jobs`, shaped:
 *   { status: 'fulfilled', value }  |  { status: 'rejected', reason }
 *
 * One failing job must not cancel the others, and the pool must never have
 * more than `limit` promises in flight — starting them all and awaiting later
 * is exactly the bug this task is about.
 */
export async function runPool(jobs, limit) {
  throw new Error('not implemented');
}
