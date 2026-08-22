/**
 * Cache-aside over a slow source.
 *
 * createCache({ load, ttlMs, now }) returns:
 *   { get(key), invalidate(key), size }
 *
 * - `load(key)` is the expensive read. Call it only on a miss.
 * - `ttlMs` expires an entry; `now()` returns the current millisecond clock
 *   and the tests replace it, so never read `Date.now()` directly.
 * - Concurrent `get(key)` calls that all miss must produce exactly ONE `load`
 *   call — the others wait on the same in-flight promise. Without this a
 *   popular key that expires sends every request to the source at once.
 * - A rejected `load` must not be cached: the next `get` retries.
 * - `invalidate(key)` drops the entry so the next read reloads.
 */
export function createCache({ load, ttlMs = 1000, now = () => Date.now() }) {
  throw new Error('not implemented');
}
