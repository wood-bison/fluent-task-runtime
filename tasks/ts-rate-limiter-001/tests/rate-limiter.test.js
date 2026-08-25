'use strict';

const assert = require('node:assert/strict');
const test = require('node:test');
const { RateLimiter } = require('../rate-limiter.ts');

test('allows requests inside the configured burst', () => {
  const limiter = new RateLimiter({ capacity: 2, refillPerSecond: 1, now: () => 0 });
  assert.equal(limiter.allow('a'), true);
  assert.equal(limiter.allow('a'), true);
  assert.equal(limiter.allow('a'), false);
});

test('refills tokens over elapsed time', () => {
  let now = 0;
  const limiter = new RateLimiter({ capacity: 1, refillPerSecond: 1, now: () => now });
  assert.equal(limiter.allow('a'), true);
  assert.equal(limiter.allow('a'), false);
  now = 1000;
  assert.equal(limiter.allow('a'), true);
});

test('isolates independent keys', () => {
  const limiter = new RateLimiter({ capacity: 1, refillPerSecond: 1, now: () => 0 });
  assert.equal(limiter.allow('a'), true);
  assert.equal(limiter.allow('b'), true);
});

test('rejects requests after the bucket is empty', () => {
  const limiter = new RateLimiter({ capacity: 1, refillPerSecond: 0, now: () => 0 });
  assert.equal(limiter.allow('a'), true);
  assert.equal(limiter.allow('a'), false);
});
