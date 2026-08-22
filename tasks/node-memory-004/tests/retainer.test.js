import test from 'node:test';
import assert from 'node:assert/strict';
import { writeFileSync } from 'node:fs';
import { createRetainer } from '../retainer.js';

test('retains a value and returns a release token', () => {
  const retainer = createRetainer();
  const token = retainer.retain({ id: 'a' });
  assert.equal(typeof token, 'string');
  assert.equal(retainer.retainedCount, 1);
});

test('releases one value without disturbing another', () => {
  const retainer = createRetainer();
  const first = retainer.retain('first');
  retainer.retain('second');
  retainer.release(first);
  assert.equal(retainer.retainedCount, 1);
});

test('release is idempotent', () => {
  const retainer = createRetainer();
  const token = retainer.retain('once');
  retainer.release(token);
  retainer.release(token);
  assert.equal(retainer.retainedCount, 0);
});

test('clears all retained values', () => {
  const retainer = createRetainer();
  retainer.retain('a');
  retainer.retain('b');
  retainer.clear();
  assert.deepEqual(retainer.snapshot(), { retainedCount: 0, tokenIds: [] });
});

test('writes bounded retention evidence', () => {
  const retainer = createRetainer();
  retainer.retain({ kind: 'request-cache' });
  writeFileSync('/output/retention.json', JSON.stringify(retainer.snapshot()));
  assert.equal(retainer.snapshot().retainedCount, 1);
});
