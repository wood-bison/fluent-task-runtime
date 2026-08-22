import test from 'node:test';
import assert from 'node:assert/strict';
import { Deferred } from '../deferred.js';

test('starts pending', async () => {
  const deferred = new Deferred();
  let settled = false;
  deferred.promise.then(() => {
    settled = true;
  });
  await Promise.resolve();
  assert.equal(settled, false);
});

test('resolves with the settled value', async () => {
  const deferred = new Deferred();
  deferred.resolve('hello');
  const value = await deferred.promise;
  assert.equal(value, 'hello');
});

test('rejects with the settled reason', async () => {
  const deferred = new Deferred();
  deferred.reject(new Error('boom'));
  await assert.rejects(deferred.promise, /boom/u);
});

test('adopts a thenable passed directly to resolve', async () => {
  const deferred = new Deferred();
  let received;
  // A plain callback, not `await` — awaiting `deferred.promise` directly would
  // route the raw value through the language's own Promise-resolution
  // machinery, which adopts nested thenables regardless of what resolve()
  // does. Capturing via a bare .then() callback observes resolve()'s own
  // behaviour instead of the runtime's.
  deferred.promise.then((value) => {
    received = value;
  });
  deferred.resolve({
    then(onFulfilled) {
      onFulfilled('unwrapped-value');
    },
  });
  await new Promise((resolve) => setTimeout(resolve, 0));
  assert.equal(received, 'unwrapped-value');
});

test('chains a thenable returned from then', async () => {
  const deferred = new Deferred();
  let received;
  // Two plain `.then()` callbacks, not `await` on the chain: `await`
  // applies its own resolution to whatever the whole expression evaluates
  // to, which would adopt a thenable returned from the first `.then()`
  // regardless of whether the Deferred's OWN `.then()` does — the same gap
  // 'adopts a thenable passed directly to resolve' avoids above.
  // Capturing `received` through a second `.then()` callback observes only
  // what the first `.then()` actually resolved its own next link with.
  deferred.promise
    .then((n) => ({
      then(onFulfilled) {
        onFulfilled(n + 1);
      },
    }))
    .then((value) => {
      received = value;
    });
  deferred.resolve(1);
  await new Promise((resolve) => setTimeout(resolve, 0));
  assert.equal(received, 2);
});

test('propagates an error to the next catch', async () => {
  const deferred = new Deferred();
  deferred.resolve(1);
  let caught;
  await deferred.promise
    .then(() => {
      throw new Error('inner failure');
    })
    .catch((error) => {
      caught = error;
    });
  assert.equal(caught.message, 'inner failure');
});

test('ignores a second settle', async () => {
  const deferred = new Deferred();
  deferred.resolve('first');
  deferred.resolve('second');
  deferred.reject(new Error('third'));
  const value = await deferred.promise;
  assert.equal(value, 'first');
});
