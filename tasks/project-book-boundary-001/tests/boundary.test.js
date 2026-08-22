import test from 'node:test';
import assert from 'node:assert/strict';
import { ownsBoundary } from '../src/boundary.ts';

test('accepts a server-owned boundary', () => {
  assert.equal(ownsBoundary({ owner: 'server', privateContent: false }), true);
});

test('rejects a browser-owned boundary', () => {
  assert.equal(ownsBoundary({ owner: 'browser', privateContent: false }), false);
});

test('rejects private content crossing the boundary', () => {
  assert.equal(ownsBoundary({ owner: 'server', privateContent: true }), false);
});
