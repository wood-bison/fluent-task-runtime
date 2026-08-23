import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import test from 'node:test';

const publishedOrders = [
  ['start', 'end', 'nextTick', 'promise', 'timeout'],
  ['sync', 'end', 'nextTick-top', 'promise-top', 'promise-from-nextTick', 'nextTick-from-promise'],
  ['start', 'end', 'nextTick', 'promise', 'queueMicrotask'],
  ['start', 'end', 'queueMicrotask', 'promise', 'nextTick'],
  ['start', 'io', 'immediate', 'timeout'],
  ['start', 'nextTick-3', 'nextTick-2', 'nextTick-1', 'yield'],
];

test('runs the submitted challenge, checks its order, and records trace evidence', () => {
  const result = spawnSync(
    process.execPath,
    ['--require', '/hidden-tests/tests/event-loop-preload.cjs', '/solution/index.js'],
    {
      cwd: '/solution',
      encoding: 'utf8',
      timeout: 15_000,
      env: { NODE_NO_WARNINGS: '1' },
    },
  );

  assert.equal(result.error, undefined, result.error?.message);
  assert.equal(result.status, 0, result.stderr || 'submitted event-loop program failed');

  const output = result.stdout.trim().split(/\r?\n/u).filter(Boolean);
  assert.ok(
    publishedOrders.some((expected) => JSON.stringify(expected) === JSON.stringify(output)),
    `unexpected event-loop order: ${JSON.stringify(output)}`,
  );
  // The Lab's evidence adapter consumes the controlled program's stdout from
  // the runtime envelope. Re-emit only the bounded learner-visible lines;
  // hidden assertions and trace internals never cross this boundary.
  process.stdout.write(result.stdout);

  const trace = JSON.parse(fs.readFileSync('/output/trace.json', 'utf8'));
  assert.ok(Array.isArray(trace) && trace.length > 0, 'trace.json must contain measured events');
  for (const event of trace) {
    assert.equal(typeof event.id, 'string');
    assert.equal(typeof event.label, 'string');
    assert.equal(typeof event.detail, 'string');
    assert.ok(['sync', 'nextTick', 'microtask', 'timers'].includes(event.queue));
    assert.ok(['scheduled', 'executed'].includes(event.state));
    assert.equal(typeof event.offsetMs, 'number');
  }
});
