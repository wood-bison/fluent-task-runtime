import test from 'node:test';
import assert from 'node:assert/strict';
import { createConsumer } from '../consumer.js';

test('processes each message id exactly once under redelivery', async () => {
  const calls = [];
  const consumer = createConsumer({
    handle: async (payload) => {
      calls.push(payload);
      return `ok:${payload}`;
    },
  });

  await consumer.deliver({ id: 'm1', payload: 'a' });
  await consumer.deliver({ id: 'm1', payload: 'a' });
  await consumer.deliver({ id: 'm1', payload: 'a' });

  assert.deepEqual(calls, ['a']);
  assert.deepEqual(consumer.processed, ['m1']);
});

test('retries a transient failure and then succeeds', async () => {
  let attempts = 0;
  const consumer = createConsumer({
    handle: async () => {
      attempts += 1;
      if (attempts < 2) throw new Error('broker hiccup');
      return 'recovered';
    },
    maxAttempts: 3,
  });

  const first = await consumer.deliver({ id: 'm2', payload: 'x' });
  assert.equal(first.status, 'retry');

  const second = await consumer.deliver({ id: 'm2', payload: 'x' });
  assert.equal(second.status, 'done');
  assert.equal(second.value, 'recovered');
  assert.deepEqual(consumer.deadLettered, []);
});

test('dead-letters a message after the retry bound', async () => {
  const consumer = createConsumer({
    handle: async () => {
      throw new Error('poison');
    },
    maxAttempts: 2,
  });

  assert.equal((await consumer.deliver({ id: 'm3', payload: 'p' })).status, 'retry');
  assert.equal((await consumer.deliver({ id: 'm3', payload: 'p' })).status, 'dead-letter');

  // A dead-lettered id must never reach the handler again, however many times
  // the broker redelivers it.
  const after = await consumer.deliver({ id: 'm3', payload: 'p' });
  assert.equal(after.status, 'dead-letter');
  assert.deepEqual(consumer.deadLettered, ['m3']);
});

test('keeps a failing message from blocking the rest', async () => {
  const consumer = createConsumer({
    handle: async (payload) => {
      if (payload === 'bad') throw new Error('poison');
      return payload.toUpperCase();
    },
    maxAttempts: 1,
  });

  await consumer.deliver({ id: 'bad', payload: 'bad' });
  const good = await consumer.deliver({ id: 'good', payload: 'fine' });

  assert.equal(good.status, 'done');
  assert.equal(good.value, 'FINE');
  assert.deepEqual(consumer.deadLettered, ['bad']);
  assert.deepEqual(consumer.processed, ['good']);
});

test('reports the same result for a duplicate arriving after success', async () => {
  let runs = 0;
  const consumer = createConsumer({
    handle: async () => {
      runs += 1;
      return { chargeId: `charge-${runs}` };
    },
  });

  const first = await consumer.deliver({ id: 'pay-1', payload: {} });
  const duplicate = await consumer.deliver({ id: 'pay-1', payload: {} });

  assert.equal(runs, 1, 'the payment must not be charged twice');
  assert.deepEqual(duplicate.value, first.value);
  assert.equal(duplicate.status, 'done');
});
