# Rate limiter in Go

Implement `RateLimiter.Allow` as a per-key token bucket. The constructor
defines the burst capacity and refill rate in tokens per second. A call consumes
one token; elapsed time refills up to capacity. Keep the operation deterministic
for the supplied `time.Time` and do not let one key consume another key's
budget.
