# Rate limiter in TypeScript

Implement `RateLimiter.allow(key)` as a deterministic per-key token bucket.
`capacity` is the burst size, `refillPerSecond` is the refill rate, and the
injected `now()` clock returns milliseconds. Each accepted call consumes one
token; elapsed time restores tokens up to `capacity`. Keys must never share a
bucket, and a zero refill rate must remain bounded forever.

This is the TypeScript revision of the rate-limiter capability. It runs in the
same Node.js 24 sandbox as the JavaScript revision, but keeps the public
contract explicitly typed and uses Node's bounded type stripping at execution.
