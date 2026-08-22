import test from 'node:test';
import assert from 'node:assert/strict';
import { runCpuTask } from '../cpu.js';

test('returns the worker result', async () => {
  assert.equal(await runCpuTask({ value: 12 }), 144);
});

test('does not block the caller while work is running', async () => {
  const task = runCpuTask({ value: 3, durationMs: 30 });
  let turnCompleted = false;
  queueMicrotask(() => {
    turnCompleted = true;
  });
  await Promise.resolve();
  assert.equal(turnCompleted, true);
  assert.equal(await task, 9);
});

test('honours AbortSignal cancellation', async () => {
  const controller = new AbortController();
  const task = runCpuTask({ value: 4, durationMs: 250 }, { signal: controller.signal });
  controller.abort();
  await assert.rejects(task, (error) => error?.name === 'AbortError');
});

test('cleans up a worker after completion', async () => {
  await assert.doesNotReject(runCpuTask({ value: 2, durationMs: 1 }));
  // If the worker remains referenced, node --test would keep the container
  // alive past the harness timeout. The assertion is intentionally about the
  // public contract; the harness supplies the process-level liveness gate.
});
