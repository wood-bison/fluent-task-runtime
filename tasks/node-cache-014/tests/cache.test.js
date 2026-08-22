import test from 'node:test';
import assert from 'node:assert/strict';
import { createCache } from '../cache.js';

function clock(start = 0) {
  let value = start;
  return { now: () => value, advance: (ms) => (value += ms) };
}

test('loads once and serves the rest from cache', async () => {
  let loads = 0;
  const time = clock();
  const cache = createCache({
    load: async (key) => {
      loads += 1;
      return `value:${key}`;
    },
    ttlMs: 1000,
    now: time.now,
  });

  assert.equal(await cache.get('a'), 'value:a');
  assert.equal(await cache.get('a'), 'value:a');
  assert.equal(await cache.get('a'), 'value:a');
  assert.equal(loads, 1);
});

test('collapses concurrent misses into a single load', async () => {
  let loads = 0;
  const time = clock();
  const cache = createCache({
    load: async () => {
      loads += 1;
      await new Promise((resolve) => setTimeout(resolve, 20));
      return 'hot';
    },
    ttlMs: 1000,
    now: time.now,
  });

  // Fifty readers arrive while the key is cold. A cache without single-flight
  // sends all fifty to the source — the stampede this task is named after.
  const readers = Array.from({ length: 50 }, () => cache.get('hot-key'));
  const values = await Promise.all(readers);

  assert.deepEqual(new Set(values), new Set(['hot']));
  assert.equal(loads, 1, `source was hit ${loads} times, expected 1`);
});

test('expires an entry after its ttl', async () => {
  let loads = 0;
  const time = clock();
  const cache = createCache({
    load: async () => {
      loads += 1;
      return loads;
    },
    ttlMs: 100,
    now: time.now,
  });

  assert.equal(await cache.get('k'), 1);
  time.advance(99);
  assert.equal(await cache.get('k'), 1, 'still fresh inside the ttl');
  time.advance(2);
  assert.equal(await cache.get('k'), 2, 'reloaded after the ttl');
});

test('invalidates on write so the next read is fresh', async () => {
  let stored = 'before';
  const time = clock();
  const cache = createCache({
    load: async () => stored,
    ttlMs: 10_000,
    now: time.now,
  });

  assert.equal(await cache.get('row'), 'before');
  stored = 'after';
  assert.equal(await cache.get('row'), 'before', 'still the cached copy');
  cache.invalidate('row');
  assert.equal(await cache.get('row'), 'after');
});

test('does not cache a failed load', async () => {
  let attempts = 0;
  const time = clock();
  const cache = createCache({
    load: async () => {
      attempts += 1;
      if (attempts === 1) throw new Error('source down');
      return 'recovered';
    },
    ttlMs: 1000,
    now: time.now,
  });

  await assert.rejects(cache.get('k'), /source down/u);
  assert.equal(await cache.get('k'), 'recovered');
  assert.equal(attempts, 2);
});
