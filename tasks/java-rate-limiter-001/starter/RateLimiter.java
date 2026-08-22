import java.time.Instant;
import java.util.HashMap;
import java.util.Map;

public final class RateLimiter {
  private static final class Bucket {
    double tokens;
    Instant last;
    Bucket(double tokens, Instant last) { this.tokens = tokens; this.last = last; }
  }

  private final int capacity;
  private final double refillPerSecond;
  private final Map<String, Bucket> buckets = new HashMap<>();

  public RateLimiter(int capacity, double refillPerSecond) {
    this.capacity = capacity;
    this.refillPerSecond = refillPerSecond;
  }

  public boolean allow(String key, Instant now) {
    // TODO: refill the key's bucket, consume one token, and reject empty buckets.
    return false;
  }
}
