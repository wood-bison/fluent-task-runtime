# Index that the planner rejects

The read-only `schema.sql` creates and analyzes a realistic `orders` table.
Write `solution.sql` to create the intentionally low-selectivity index
`idx_orders_status`, then measure why PostgreSQL does not use it for the common
`paid` value:

```sql
SELECT count(*)
FROM orders
WHERE status = 'paid';
```

Create a simple B-tree index named `idx_orders_status` on `status`. The hidden
checks run on a real PostgreSQL server, inspect the index catalog, verify the
common row count, and capture `EXPLAIN (FORMAT JSON, ANALYZE, BUFFERS)` as
`plan.json`. The expected plan is a sequential scan: fetching almost the whole
table through an index costs more than reading the heap once.

The task is about measured planner behavior, not memorizing a string. The
evidence artifact is the plan PostgreSQL actually chose and the reason it
rejected the apparently helpful index.
