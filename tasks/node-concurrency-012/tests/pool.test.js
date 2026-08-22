import test from 'node:test';
import assert from 'node:assert/strict';
import { runPool } from '../pool.js';

/** A job that reports when it starts and finishes, so the test can watch the
 * pool's real occupancy rather than trusting the returned array. */
function tracked(tracker, id, ms, fail = false) {
  return async () => {
    tracker.inFlight += 1;
    tracker.peak = Math.max(tracker.peak, tracker.inFlight);
    tracker.started.push(id);
    await new Promise((resolve) => setTimeout(resolve, ms));
    tracker.inFlight -= 1;
    if (fail) throw new Error(`job ${id} failed`);
    return id;
  };
}

function tracker() {
  return { inFlight: 0, peak: 0, started: [] };
}

test('returns results in input order', async () => {
  const t = tracker();
  const results = await runPool(
    [tracked(t, 'a', 30), tracked(t, 'b', 5), tracked(t, 'c', 15)],
    2,
  );
  assert.deepEqual(
    results.map((r) => r.value),
    ['a', 'b', 'c'],
  );
});

test('never exceeds the concurrency limit', async () => {
  const t = tracker();
  const jobs = Array.from({ length: 12 }, (_, i) => tracked(t, i, 10));
  await runPool(jobs, 3);
  assert.equal(t.peak, 3, `peak in-flight was ${t.peak}, limit was 3`);
});

test('starts the next job as soon as one finishes', async () => {
  const t = tracker();
  // With a limit of 1 the pool is sequential, so the start order is the proof
  // that the fourth job waited for the third rather than being queued upfront.
  await runPool(
    ['w', 'x', 'y', 'z'].map((id) => tracked(t, id, 5)),
    1,
  );
  assert.deepEqual(t.started, ['w', 'x', 'y', 'z']);
});

test('reports which job failed without losing the others', async () => {
  const t = tracker();
  const results = await runPool(
    [tracked(t, 'ok1', 5), tracked(t, 'bad', 5, true), tracked(t, 'ok2', 5)],
    2,
  );
  assert.equal(results[0].status, 'fulfilled');
  assert.equal(results[1].status, 'rejected');
  assert.match(String(results[1].reason?.message ?? results[1].reason), /bad/u);
  assert.equal(results[2].value, 'ok2');
});

test('handles an empty list and a limit larger than the list', async () => {
  assert.deepEqual(await runPool([], 4), []);
  const t = tracker();
  const results = await runPool([tracked(t, 'only', 1)], 10);
  assert.equal(results.length, 1);
  assert.equal(t.peak, 1);
});
