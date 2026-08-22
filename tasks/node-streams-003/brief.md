# Backpressure under load

Export `createJsonLineTransform()` from `stream.js`.

It must return a Node.js `Transform` stream that accepts UTF-8 chunks of
newline-delimited JSON and emits parsed objects in `objectMode`. A JSON record
may be split across chunks, blank lines should be ignored, and records must
come out in their original order.

Use the normal Transform callback contract so downstream backpressure can
pause the source. When a non-blank line is not valid JSON, call the callback
with an error that identifies malformed JSON; do not silently drop the line.
Flush any final line that does not end with a newline.
