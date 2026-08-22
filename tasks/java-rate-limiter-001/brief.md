# Rate limiter in Java

Implement `RateLimiter.allow` as a per-key token bucket. The constructor
defines the burst capacity and refill rate in tokens per second. A call consumes
one token; elapsed time refills up to capacity. Use the supplied `Instant` so
the behavior stays deterministic and testable.
