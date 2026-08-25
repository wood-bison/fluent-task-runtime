using System;
using System.Collections.Generic;

public sealed class RateLimiter
{
    private sealed class Bucket
    {
        public double Tokens;
        public DateTimeOffset Last;

        public Bucket(double tokens, DateTimeOffset last)
        {
            Tokens = tokens;
            Last = last;
        }
    }

    private readonly int capacity;
    private readonly double refillPerSecond;
    private readonly Dictionary<string, Bucket> buckets = new();

    public RateLimiter(int capacity, double refillPerSecond)
    {
        this.capacity = capacity;
        this.refillPerSecond = refillPerSecond;
    }

    public bool Allow(string key, DateTimeOffset now)
    {
        // TODO: refill this key from elapsed time, consume one token, and
        // reject an empty bucket without sharing state with another key.
        return false;
    }
}
