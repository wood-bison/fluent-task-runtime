# Rate limiter in Node.js

Implement `RateLimiter.allow(key)` as a deterministic per-key token bucket.
`capacity` is the burst size, `refillPerSecond` is the refill rate, and the
injected `now()` clock returns milliseconds. Each accepted call consumes one
token; elapsed time restores tokens up to `capacity`. Keys must never share a
bucket, and a zero refill rate must remain bounded forever.

This is the Node.js executable revision of the rate-limiter capability. The
same QuestionCard also has Go, Java, and PostgreSQL revisions; the contract is
shared, while the runtime mechanics and evidence are language-specific.
