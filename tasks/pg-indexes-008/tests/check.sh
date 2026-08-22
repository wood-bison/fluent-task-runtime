#!/usr/bin/env bash
set -Eeuo pipefail

mkdir -p /output

if ! psql -X -v ON_ERROR_STOP=1 -f /solution/solution.sql >/tmp/solution.out 2>/tmp/solution.err; then
  cat /tmp/solution.err >&2
  printf '{"version":2,"status":"error","message":"The submitted SQL did not execute."}\n' >/output/results.json
  exit 0
fi

index_exists=$(psql -X -Atqc "SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = 'idx_orders_status');")
row_count=$(psql -X -Atqc "SELECT count(*) FROM orders WHERE status = 'paid';")
plan_text=$(psql -X -Atqc "EXPLAIN (ANALYZE, BUFFERS) SELECT count(*) FROM orders WHERE status = 'paid';" || true)
printf '%s\n' "$plan_text" > /tmp/plan.txt
psql -X -qAtc "EXPLAIN (FORMAT JSON, ANALYZE, BUFFERS) SELECT count(*) FROM orders WHERE status = 'paid';" > /output/plan.json

if [[ "$index_exists" == "t" ]]; then
  index_status='pass'
else
  index_status='fail'
fi

if [[ "$row_count" == "49500" ]]; then
  rows_status='pass'
else
  rows_status='fail'
fi

if grep -q 'idx_orders_status' /tmp/plan.txt; then
  plan_status='fail'
else
  plan_status='pass'
fi

overall='pass'
if [[ "$index_status" != 'pass' || "$rows_status" != 'pass' || "$plan_status" != 'pass' ]]; then
  overall='fail'
fi

printf '{"version":2,"status":"%s","tests":[{"name":"creates the low-selectivity status index","status":"%s","message":"index catalog check"},{"name":"counts the common status rows","status":"%s","message":"query row-count check"},{"name":"planner rejects the low-selectivity index","status":"%s","message":"EXPLAIN plan check"}]}\n' \
  "$overall" "$index_status" "$rows_status" "$plan_status" >/output/results.json
