#!/usr/bin/env bash
set -Eeuo pipefail

mkdir -p /output

if ! psql -X -v ON_ERROR_STOP=1 -f /solution/solution.sql >/tmp/solution.out 2>/tmp/solution.err; then
  cat /tmp/solution.err >&2
  printf '{"version":2,"status":"error","message":"The submitted SQL did not execute."}\n' >/output/results.json
  exit 0
fi

function_exists=$(psql -X -Atqc "SELECT EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'allow_request');")
if [[ "$function_exists" != "t" ]]; then
  printf '{"version":2,"status":"fail","tests":[{"name":"defines the allow_request function","status":"fail","message":"allow_request was not created"},{"name":"denies a full window","status":"fail","message":"function is unavailable"},{"name":"allows a request after the window boundary","status":"fail","message":"function is unavailable"},{"name":"isolates independent clients","status":"fail","message":"function is unavailable"}]}\n' >/output/results.json
  exit 0
fi

full=$(psql -X -Atqc "SELECT allow_request('full', timestamptz '2025-01-01 00:00:05+00', 5, 10);")
boundary=$(psql -X -Atqc "SELECT allow_request('full', timestamptz '2025-01-01 00:00:11+00', 5, 10);")
independent=$(psql -X -Atqc "SELECT allow_request('other', timestamptz '2025-01-01 00:00:05+00', 5, 10);")

[[ "$full" == "f" ]] && full_status=pass || full_status=fail
[[ "$boundary" == "t" ]] && boundary_status=pass || boundary_status=fail
[[ "$independent" == "t" ]] && independent_status=pass || independent_status=fail

psql -X -qAtc "SELECT json_build_object('fullWindow', '$full', 'boundary', '$boundary', 'independentClient', '$independent')" > /output/rate-limit.json
overall=pass
for status in "$full_status" "$boundary_status" "$independent_status"; do
  [[ "$status" == pass ]] || overall=fail
done
printf '{"version":2,"status":"%s","tests":[{"name":"defines the allow_request function","status":"pass","message":"function catalog check"},{"name":"denies a full window","status":"%s","message":"full sliding window"},{"name":"allows a request after the window boundary","status":"%s","message":"window boundary"},{"name":"isolates independent clients","status":"%s","message":"client key isolation"}]}\n' \
  "$overall" "$full_status" "$boundary_status" "$independent_status" >/output/results.json
