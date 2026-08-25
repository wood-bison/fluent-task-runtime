# Rate limiter in .NET

Implement `RateLimiter.Allow(string key, DateTimeOffset now)` as a
deterministic per-key token bucket. The constructor defines the burst capacity
and refill rate in tokens per second. An accepted call consumes one token;
elapsed time restores tokens up to `capacity`. Keys must never share a bucket,
and a zero refill rate must remain bounded forever.

The runner compiles the file in the official .NET 10 SDK image and executes the
same four invariants as the Go, Java, TypeScript, and PostgreSQL revisions.
