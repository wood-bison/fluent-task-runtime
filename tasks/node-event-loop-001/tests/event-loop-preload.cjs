'use strict';

const fs = require('node:fs');
const { performance } = require('node:perf_hooks');

const startedAt = performance.now();
const events = [];

function mark(id, label, detail, queue, state) {
  events.push({
    id: String(id),
    label: String(label),
    detail: String(detail),
    queue: String(queue),
    state: String(state),
    offsetMs: Number((performance.now() - startedAt).toFixed(3)),
  });
}

globalThis.schedule = (id, label, detail, queue) =>
  mark(id, label, detail, queue, 'scheduled');

globalThis.emit = (
  value,
  queue = 'sync',
  id = value,
  detail = String(value),
  label = String(value),
) => {
  console.log(String(value));
  mark(id, label, detail, queue, 'executed');
};

process.on('exit', () => {
  fs.writeFileSync(
    '/output/trace.json',
    JSON.stringify(events),
    'utf8',
  );
});
