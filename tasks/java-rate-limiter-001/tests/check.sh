#!/usr/bin/env bash
set -Eeuo pipefail
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
cp /solution/RateLimiter.java "$tmp/RateLimiter.java"
cat > "$tmp/RateLimiterTest.java" <<'JAVA'
import java.time.Instant;

public final class RateLimiterTest {
  private static void check(boolean condition, String message) { if (!condition) throw new AssertionError(message); }

  public static void main(String[] args) {
    Instant now = Instant.EPOCH;
    RateLimiter burst = new RateLimiter(2, 1);
    check(burst.allow("a", now), "burst first");
    check(burst.allow("a", now), "burst second");
    check(!burst.allow("a", now), "burst third");

    RateLimiter refill = new RateLimiter(1, 1);
    check(refill.allow("a", now), "refill first");
    check(!refill.allow("a", now), "refill empty");
    check(refill.allow("a", now.plusSeconds(1)), "refill after second");

    RateLimiter keys = new RateLimiter(1, 1);
    check(keys.allow("a", now) && keys.allow("b", now), "keys independent");

    RateLimiter empty = new RateLimiter(1, 0);
    check(empty.allow("a", now) && !empty.allow("a", now.plusSeconds(3600)), "empty bucket");
  }
}
JAVA
(cd "$tmp" && javac RateLimiter.java RateLimiterTest.java && java RateLimiterTest)
