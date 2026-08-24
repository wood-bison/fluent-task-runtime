'use strict';

class RateLimiter {
  constructor({ capacity, refillPerSecond, now = () => Date.now() }) {
    this.capacity = capacity;
    this.refillPerSecond = refillPerSecond;
    this.now = now;
    this.buckets = new Map();
  }

  allow(key) {
    // TODO: refill this key from elapsed time, consume one token, and reject
    // an empty bucket without allowing another key to borrow its state.
    return false;
  }
}

module.exports = { RateLimiter };
