#!/usr/bin/env bash
set -Eeuo pipefail

mkdir -p /output

if ! psql -X -v ON_ERROR_STOP=1 -f /solution/solution.sql >/tmp/solution.out 2>/tmp/solution.err; then
  cat /tmp/solution.err >&2
  printf '{"version":2,"status":"error","message":"The submitted SQL did not execute."}\n' >/output/results.json
  exit 0
fi

fail_all() {
  printf '{"version":2,"status":"fail","tests":[{"name":"defines the claim_jobs function","status":"fail","message":"%s"}]}\n' "$1" >/output/results.json
  exit 0
}

exists=$(psql -X -Atqc "SELECT EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'claim_jobs');")
if [[ "$exists" != "t" ]]; then
  # Keep the result contract complete even when the learner has not created
  # the function yet. Returning one short-circuiting assertion used to make
  # the workbench report 1/5 tests and hid the rest of the task contract.
  printf '{"version":2,"status":"fail","tests":[{"name":"defines the claim_jobs function","status":"fail","message":"claim_jobs was not created"},{"name":"claims exactly the requested batch size","status":"fail","message":"cannot claim a batch until claim_jobs exists"},{"name":"two concurrent workers never claim the same job","status":"fail","message":"cannot run concurrent claims until claim_jobs exists"},{"name":"a locked row is skipped, not waited on","status":"fail","message":"cannot inspect lock skipping until claim_jobs exists"},{"name":"claimed jobs are marked running, not left pending","status":"fail","message":"cannot inspect claim state until claim_jobs exists"}]}' >/output/results.json
  exit 0
fi
defines_status='pass'

batch=$(psql -X -Atqc "SELECT count(*) FROM claim_jobs('w0', 5);")
if [[ "$batch" == "5" ]]; then batch_status='pass'; else batch_status='fail'; fi

# Two workers claim concurrently from separate sessions. The overlap between
# their results is the whole test.
psql -X -Atqc "SELECT id FROM claim_jobs('w1', 20);" >/tmp/w1.txt &
w1=$!
psql -X -Atqc "SELECT id FROM claim_jobs('w2', 20);" >/tmp/w2.txt &
w2=$!
wait "$w1" || true
wait "$w2" || true

overlap=$(sort /tmp/w1.txt /tmp/w2.txt | uniq -d | grep -c . || true)
if [[ "$overlap" == "0" ]]; then disjoint_status='pass'; else disjoint_status='fail'; fi

claimed_total=$(sort -u /tmp/w1.txt /tmp/w2.txt | grep -c . || true)
if [[ "$claimed_total" == "40" ]]; then skip_status='pass'; else skip_status='fail'; fi

running=$(psql -X -Atqc "SELECT count(*) FROM jobs WHERE status = 'running' AND claimed_by IS NOT NULL;")
if [[ "$running" == "45" ]]; then marked_status='pass'; else marked_status='fail'; fi

psql -X -qAtc "SELECT json_agg(row_to_json(t)) FROM (SELECT claimed_by, count(*) AS claimed FROM jobs WHERE status = 'running' GROUP BY claimed_by ORDER BY claimed_by) t;" > /output/claims.json

overall='pass'
for status in "$defines_status" "$batch_status" "$disjoint_status" "$skip_status" "$marked_status"; do
  [[ "$status" == 'pass' ]] || overall='fail'
done

printf '{"version":2,"status":"%s","tests":[{"name":"defines the claim_jobs function","status":"%s","message":"pg_proc catalog check"},{"name":"claims exactly the requested batch size","status":"%s","message":"claimed %s of 5"},{"name":"two concurrent workers never claim the same job","status":"%s","message":"%s overlapping ids"},{"name":"a locked row is skipped, not waited on","status":"%s","message":"%s distinct ids across both workers, expected 40"},{"name":"claimed jobs are marked running, not left pending","status":"%s","message":"%s rows running, expected 45"}]}\n' \
  "$overall" "$defines_status" "$batch_status" "$batch" "$disjoint_status" "$overlap" "$skip_status" "$claimed_total" "$marked_status" "$running" >/output/results.json
