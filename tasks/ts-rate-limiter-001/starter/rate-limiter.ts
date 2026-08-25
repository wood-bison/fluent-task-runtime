export type Clock = () => number;

type Bucket = {
  tokens: number;
  lastMs: number;
};

export type RateLimiterOptions = {
  capacity: number;
  refillPerSecond: number;
  now?: Clock;
};

export class RateLimiter {
  private readonly capacity: number;
  private readonly refillPerSecond: number;
  private readonly now: Clock;
  private readonly buckets = new Map<string, Bucket>();

  constructor({ capacity, refillPerSecond, now = () => Date.now() }: RateLimiterOptions) {
    this.capacity = capacity;
    this.refillPerSecond = refillPerSecond;
    this.now = now;
  }

  allow(key: string): boolean {
    // TODO: refill this key from elapsed time, consume one token, and reject
    // an empty bucket without allowing another key to borrow its state.
    return false;
  }
}
