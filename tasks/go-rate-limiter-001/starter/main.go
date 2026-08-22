package main

import "time"

type bucket struct {
	tokens float64
	last   time.Time
}

// RateLimiter is intentionally incomplete. Keep the public shape stable while
// making the refill and key-isolation policy explicit in the implementation.
type RateLimiter struct {
	capacity int
	refill   float64
	buckets  map[string]bucket
}

func NewRateLimiter(capacity int, refillPerSecond float64) *RateLimiter {
	return &RateLimiter{capacity: capacity, refill: refillPerSecond, buckets: map[string]bucket{}}
}

func (r *RateLimiter) Allow(key string, now time.Time) bool {
	// TODO: refill the key's bucket, consume one token, and reject empty buckets.
	return false
}

func main() {}
