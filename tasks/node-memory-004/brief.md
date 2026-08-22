# Find the retained object

Export `createRetainer()` from `retainer.js`.

The retainer models a cache or subscription registry that owns values until a
caller explicitly releases them. It must expose:

- `.retain(value)` → a token for that value;
- `.release(token)` → removes that value; releasing the same token again is a
  no-op;
- `.clear()` → removes all values;
- `.retainedCount` → the number of currently live values;
- `.snapshot()` → a small JSON-safe report containing `retainedCount` and the
  active token ids.

Do not keep a second hidden reference after release. The final hidden test
writes the snapshot to `/output/retention.json`; that file appears in the
evidence panel only after the runner has verified it.
