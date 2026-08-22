# Cache-aside that survives a stampede

Cache-aside is four lines until it meets load. Three failures show up in that
order in production:

- **Stampede.** A popular key expires and every concurrent reader misses at
  once, so the source takes the full request rate it was cached to avoid. The
  fix is single-flight: the first miss starts the load, the rest await the same
  promise.
- **Stale after write.** A TTL is not invalidation. If a write does not drop
  the key, readers see the old row for as long as the TTL says.
- **Cached failure.** Storing a rejection turns a blip into an outage that
  lasts a full TTL.

The clock is injected so the tests can move time without sleeping — treat
`now()` as the only source of time.
