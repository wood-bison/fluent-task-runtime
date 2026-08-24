# Rate limiter in PostgreSQL

Implement `allow_request(client_id, at, max_requests, window_seconds)` in
`solution.sql`. Count events in the half-open window `[at - window, at)`,
insert a new event only when the limit allows it, and serialize concurrent
decisions for the same client. The hidden harness runs PostgreSQL 17, checks a
full window, an expired-event boundary, and independent clients, then stores
the observed decision as `rate-limit.json` evidence.

This revision is intentionally a database-backed variant of the same
rate-limiter QuestionCard; it is not the Node.js source copied into SQL.
