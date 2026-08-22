import test from 'node:test';
import assert from 'node:assert/strict';
import { hashPassword, verifyPassword, readRecord } from '../auth.js';

const PASSWORD = 'correct horse battery staple';

test('never stores the password itself', async () => {
  const stored = await hashPassword(PASSWORD);
  assert.equal(typeof stored, 'string');
  assert.ok(!stored.includes(PASSWORD), 'the stored value contains the password');
  assert.ok(stored.length >= 32, 'the stored verifier looks too short to be a KDF output');
});

test('gives two identical passwords different stored verifiers', async () => {
  const first = await hashPassword(PASSWORD);
  const second = await hashPassword(PASSWORD);
  // Equal verifiers mean no per-password salt, which is what makes one
  // precomputed table work against every account at once.
  assert.notEqual(first, second);
});

test('verifies the right password and rejects the wrong one', async () => {
  const stored = await hashPassword(PASSWORD);
  assert.equal(await verifyPassword(PASSWORD, stored), true);
  assert.equal(await verifyPassword('wrong password', stored), false);
  assert.equal(await verifyPassword('', stored), false);
});

test('compares the verifier without leaking length by early exit', async () => {
  const stored = await hashPassword(PASSWORD);
  // A truncated or corrupted verifier must be answered with false, not with a
  // thrown length error from timingSafeEqual.
  assert.equal(await verifyPassword(PASSWORD, stored.slice(0, stored.length - 4)), false);
  assert.equal(await verifyPassword(PASSWORD, 'not-a-verifier'), false);
});

test('authenticated is not authorized: another user\'s record is refused', () => {
  const records = new Map([
    ['r1', { ownerId: 'alice', body: 'alice private' }],
    ['r2', { ownerId: 'bob', body: 'bob private' }],
  ]);
  const alice = { id: 'alice', role: 'user' };

  assert.equal(readRecord(alice, 'r1', records).body, 'alice private');
  assert.throws(() => readRecord(alice, 'r2', records), /forbidden/iu);
  assert.throws(() => readRecord(alice, 'missing', records), /forbidden/iu);
});

test('an admin may read any record', () => {
  const records = new Map([['r2', { ownerId: 'bob', body: 'bob private' }]]);
  const admin = { id: 'root', role: 'admin' };
  assert.equal(readRecord(admin, 'r2', records).body, 'bob private');
});
