# CPU-bound request incident

The API process must stay responsive while a CPU-heavy operation runs. Export
`runCpuTask(input, options)` from `cpu.js`.

`input` has this shape:

```js
{ value: number, durationMs?: number }
```

The returned promise resolves to `value * value`. The work must run in a
`worker_threads` Worker, not in the caller's event loop. `options.signal` is
optional; if it is aborted before or during the work, terminate the worker and
reject with an error whose name is `AbortError`.

The hidden test uses a short artificial duration so the boundary is visible
without turning this into a benchmark. Always clean up the Worker on success,
failure, and cancellation.
