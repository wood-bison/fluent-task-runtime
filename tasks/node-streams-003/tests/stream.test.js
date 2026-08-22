import test from 'node:test';
import assert from 'node:assert/strict';
import { Readable } from 'node:stream';
import { createJsonLineTransform } from '../stream.js';

async function collect(chunks) {
  const output = Readable.from(chunks).pipe(createJsonLineTransform());
  const records = [];
  for await (const record of output) records.push(record);
  return records;
}

test('parses records split across chunks', async () => {
  assert.deepEqual(await collect(['{"id":', '1}\n{"id":2}']), [{ id: 1 }, { id: 2 }]);
});

test('ignores blank lines', async () => {
  assert.deepEqual(await collect(['\n {"ok":true}\n\n']), [{ ok: true }]);
});

test('preserves record order under flow', async () => {
  const records = await collect(['{"id":1}\n', '{"id":2}\n', '{"id":3}\n']);
  assert.deepEqual(records.map((record) => record.id), [1, 2, 3]);
});

test('emits a useful error for malformed JSON', async () => {
  await assert.rejects(collect(['{"id":}\n']), /malformed JSON/u);
});
