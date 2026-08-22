/**
 * Build a consumer for a queue that delivers AT LEAST once — the same message
 * can arrive again after a crash, a timeout, or a rebalance.
 *
 * createConsumer({ handle, maxAttempts }) returns:
 *   { deliver(message), processed, deadLettered }
 *
 * - `message` is { id, payload }. The same `id` means the same work.
 * - `handle(payload)` does the work and may throw. It must run at most once
 *   per id, however many times that id is delivered.
 * - A throwing handler is retried on the NEXT delivery of that id, up to
 *   `maxAttempts` total. After that the message is dead-lettered and never
 *   handed to `handle` again.
 * - `deliver` resolves with { status: 'done' | 'retry' | 'dead-letter',
 *   value } — `value` is the handler's result, and a duplicate of an
 *   already-done id resolves with the SAME value without re-running the work.
 * - `processed` and `deadLettered` are arrays of ids, in the order they got
 *   there.
 */
export function createConsumer({ handle, maxAttempts = 3 }) {
  throw new Error('not implemented');
}
